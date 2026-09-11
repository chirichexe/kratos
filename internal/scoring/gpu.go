/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package scoring

import (
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// ---------------------------------------------------------------------------
// NVIDIA GPU Operator label keys
// ---------------------------------------------------------------------------

const (
	// LabelGPUProduct is set by the NVIDIA GPU Operator to the GPU product
	// name (e.g. "NVIDIA-A100-SXM4-80GB").
	LabelGPUProduct = "nvidia.com/gpu.product"

	// LabelGPUFamily is set to the architecture family (e.g. "ampere").
	LabelGPUFamily = "nvidia.com/gpu.family"

	// LabelGPUMemory is the total framebuffer memory in MiB.
	LabelGPUMemory = "nvidia.com/gpu.memory"

	// LabelGPUComputeCapMajor is the major CUDA compute capability version.
	LabelGPUComputeCapMajor = "nvidia.com/gpu.compute.major"

	// LabelGPUComputeCapMinor is the minor CUDA compute capability version.
	LabelGPUComputeCapMinor = "nvidia.com/gpu.compute.minor"

	// LabelGPUCount is an optional label indicating the number of GPUs.
	LabelGPUCount = "nvidia.com/gpu.count"

	// ResourceNVIDIAGPU is the extended resource name for NVIDIA GPUs.
	ResourceNVIDIAGPU = "nvidia.com/gpu"
)

// ---------------------------------------------------------------------------
// Known GPU hardware specifications (fallback lookup)
// ---------------------------------------------------------------------------

// gpuHardwareSpec holds hardware parameters that cannot be read from
// Kubernetes labels and must be inferred from the product name.
type gpuHardwareSpec struct {
	ArchFamily          string
	ComputeMajor        int
	ComputeMinor        int
	SMCount             int
	VRAMBytes           int64
	MemoryBandwidthGBps float64
}

// knownGPUs maps normalized GPU product substrings to their specs.
// The key is compared against the lower-cased, dash-normalized product label.
var knownGPUs = map[string]gpuHardwareSpec{
	// Hopper
	"h100-sxm":  {ArchFamily: "Hopper", ComputeMajor: 9, ComputeMinor: 0, SMCount: 132, VRAMBytes: 80 * giB, MemoryBandwidthGBps: 3350},
	"h100-pcie": {ArchFamily: "Hopper", ComputeMajor: 9, ComputeMinor: 0, SMCount: 114, VRAMBytes: 80 * giB, MemoryBandwidthGBps: 2000},
	"h200":      {ArchFamily: "Hopper", ComputeMajor: 9, ComputeMinor: 0, SMCount: 132, VRAMBytes: 141 * giB, MemoryBandwidthGBps: 4800},
	// Ada Lovelace
	"l40s":     {ArchFamily: "Ada Lovelace", ComputeMajor: 8, ComputeMinor: 9, SMCount: 142, VRAMBytes: 48 * giB, MemoryBandwidthGBps: 864},
	"rtx-4090": {ArchFamily: "Ada Lovelace", ComputeMajor: 8, ComputeMinor: 9, SMCount: 128, VRAMBytes: 24 * giB, MemoryBandwidthGBps: 1008},
	"rtx-4080": {ArchFamily: "Ada Lovelace", ComputeMajor: 8, ComputeMinor: 9, SMCount: 76, VRAMBytes: 16 * giB, MemoryBandwidthGBps: 717},
	// Ampere
	"a100-sxm4-80gb": {ArchFamily: "Ampere", ComputeMajor: 8, ComputeMinor: 0, SMCount: 108, VRAMBytes: 80 * giB, MemoryBandwidthGBps: 2039},
	"a100-sxm4-40gb": {ArchFamily: "Ampere", ComputeMajor: 8, ComputeMinor: 0, SMCount: 108, VRAMBytes: 40 * giB, MemoryBandwidthGBps: 1555},
	"a100-pcie-80gb": {ArchFamily: "Ampere", ComputeMajor: 8, ComputeMinor: 0, SMCount: 108, VRAMBytes: 80 * giB, MemoryBandwidthGBps: 1935},
	"a100-pcie-40gb": {ArchFamily: "Ampere", ComputeMajor: 8, ComputeMinor: 0, SMCount: 108, VRAMBytes: 40 * giB, MemoryBandwidthGBps: 1555},
	"a10":            {ArchFamily: "Ampere", ComputeMajor: 8, ComputeMinor: 6, SMCount: 72, VRAMBytes: 24 * giB, MemoryBandwidthGBps: 600},
	"a30":            {ArchFamily: "Ampere", ComputeMajor: 8, ComputeMinor: 0, SMCount: 56, VRAMBytes: 24 * giB, MemoryBandwidthGBps: 933},
	"rtx-3090":       {ArchFamily: "Ampere", ComputeMajor: 8, ComputeMinor: 6, SMCount: 82, VRAMBytes: 24 * giB, MemoryBandwidthGBps: 936},
	"rtx-3080":       {ArchFamily: "Ampere", ComputeMajor: 8, ComputeMinor: 6, SMCount: 68, VRAMBytes: 10 * giB, MemoryBandwidthGBps: 760},
	// Turing
	"t4":          {ArchFamily: "Turing", ComputeMajor: 7, ComputeMinor: 5, SMCount: 40, VRAMBytes: 16 * giB, MemoryBandwidthGBps: 320},
	"rtx-2080-ti": {ArchFamily: "Turing", ComputeMajor: 7, ComputeMinor: 5, SMCount: 68, VRAMBytes: 11 * giB, MemoryBandwidthGBps: 616},
	// Volta
	"v100-sxm2": {ArchFamily: "Volta", ComputeMajor: 7, ComputeMinor: 0, SMCount: 80, VRAMBytes: 32 * giB, MemoryBandwidthGBps: 900},
	"v100-pcie": {ArchFamily: "Volta", ComputeMajor: 7, ComputeMinor: 0, SMCount: 80, VRAMBytes: 32 * giB, MemoryBandwidthGBps: 900},
	// Pascal
	"p100": {ArchFamily: "Pascal", ComputeMajor: 6, ComputeMinor: 0, SMCount: 56, VRAMBytes: 16 * giB, MemoryBandwidthGBps: 732},
}

const giB = 1 << 30 // 1 GiB in bytes

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// ExtractGPUSpec builds a GPUSpec from a corev1.Node by reading NVIDIA GPU
// Operator labels, the allocatable GPU resource quantity, and falling back
// to the built-in lookup table for fields that are not available as labels.
func ExtractGPUSpec(node corev1.Node) (GPUSpec, error) {
	labels := node.Labels
	if labels == nil {
		labels = map[string]string{}
	}

	spec := GPUSpec{
		Product:    labels[LabelGPUProduct],
		ArchFamily: normalizeArch(labels[LabelGPUFamily]),
	}

	// Compute capability from labels.
	if v, ok := labels[LabelGPUComputeCapMajor]; ok {
		spec.ComputeMajor, _ = strconv.Atoi(v)
	}
	if v, ok := labels[LabelGPUComputeCapMinor]; ok {
		spec.ComputeMinor, _ = strconv.Atoi(v)
	}

	// VRAM from label (in MiB).
	if v, ok := labels[LabelGPUMemory]; ok {
		mib, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			spec.VRAMBytes = mib * 1024 * 1024 // MiB → bytes
		}
	}

	// GPU count: prefer the allocatable resource, fall back to label.
	spec.Count = allocatableGPUs(node)
	if spec.Count == 0 {
		if v, ok := labels[LabelGPUCount]; ok {
			spec.Count, _ = strconv.ParseInt(v, 10, 64)
		}
	}

	// Merge in any fallback data from the lookup table.
	applyFallback(&spec)

	return spec, nil
}

// IsGPUNode reports whether a node has at least one allocatable NVIDIA GPU
// or is labelled with a GPU product.
func IsGPUNode(node corev1.Node) bool {
	if allocatableGPUs(node) > 0 {
		return true
	}
	_, hasProduct := node.Labels[LabelGPUProduct]
	return hasProduct
}

// FilterGPUNodes returns the subset of nodes that have at least one
// allocatable GPU and meet the minimum GPU count specified by minGPUs.
// Nodes with zero allocatable GPUs are silently dropped.
func FilterGPUNodes(nodes []corev1.Node, minGPUs int64) []corev1.Node {
	result := make([]corev1.Node, 0, len(nodes))
	for _, n := range nodes {
		if count := allocatableGPUs(n); count >= minGPUs {
			result = append(result, n)
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// Internals
// ---------------------------------------------------------------------------

// allocatableGPUs reads the nvidia.com/gpu quantity from the node's
// Allocatable map and returns it as an int64.
func allocatableGPUs(node corev1.Node) int64 {
	qty, ok := node.Status.Allocatable[corev1.ResourceName(ResourceNVIDIAGPU)]
	if !ok {
		return 0
	}
	return qty.Value()
}

// normalizeArch title-cases the architecture family string so that
// comparisons are case-insensitive (the GPU Operator uses lowercase).
func normalizeArch(raw string) string {
	if raw == "" {
		return ""
	}
	// Simple title-case: uppercase the first letter, keep the rest.
	return strings.ToUpper(raw[:1]) + raw[1:]
}

// normalizeProduct converts a GPU product label into the key format used
// by the knownGPUs lookup table: lower-cased, spaces replaced with dashes,
// "nvidia-" prefix stripped.
func normalizeProduct(product string) string {
	s := strings.ToLower(product)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.TrimPrefix(s, "nvidia-")
	return s
}

// applyFallback enriches spec with data from the knownGPUs table using the
// product name as a lookup key. Only zero-valued fields are overwritten.
func applyFallback(spec *GPUSpec) {
	if spec.Product == "" {
		return
	}

	norm := normalizeProduct(spec.Product)

	// Try exact match first, then longest-prefix match.
	hw, ok := knownGPUs[norm]
	if !ok {
		hw, ok = longestPrefixMatch(norm)
	}
	if !ok {
		return
	}

	if spec.ArchFamily == "" {
		spec.ArchFamily = hw.ArchFamily
	}
	if spec.ComputeMajor == 0 {
		spec.ComputeMajor = hw.ComputeMajor
		spec.ComputeMinor = hw.ComputeMinor
	}
	if spec.SMCount == 0 {
		spec.SMCount = hw.SMCount
	}
	if spec.VRAMBytes == 0 {
		spec.VRAMBytes = hw.VRAMBytes
	}
	if spec.MemoryBandwidthGBps == 0 {
		spec.MemoryBandwidthGBps = hw.MemoryBandwidthGBps
	}
}

// longestPrefixMatch finds the knownGPUs entry whose key is the longest
// prefix of the normalized product name.
func longestPrefixMatch(norm string) (gpuHardwareSpec, bool) {
	var best gpuHardwareSpec
	bestLen := 0
	for key, hw := range knownGPUs {
		if strings.HasPrefix(norm, key) && len(key) > bestLen {
			best = hw
			bestLen = len(key)
		}
	}
	return best, bestLen > 0
}

// MustParseQuantity is a test helper that panics on invalid quantities.
func MustParseQuantity(s string) resource.Quantity {
	q, err := resource.ParseQuantity(s)
	if err != nil {
		panic(fmt.Sprintf("scoring: invalid quantity %q: %v", s, err))
	}
	return q
}
