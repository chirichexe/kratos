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
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// makeNode builds a minimal corev1.Node for testing.
func makeNode(name string, labels map[string]string, allocatableGPUs int64) corev1.Node {
	node := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Status: corev1.NodeStatus{
			Allocatable: corev1.ResourceList{},
		},
	}
	if allocatableGPUs > 0 {
		node.Status.Allocatable[corev1.ResourceName(ResourceNVIDIAGPU)] = *resource.NewQuantity(allocatableGPUs, resource.DecimalSI)
	}
	return node
}

// ---------------------------------------------------------------------------
// ExtractGPUSpec
// ---------------------------------------------------------------------------

func TestExtractGPUSpec(t *testing.T) {
	tests := []struct {
		name   string
		node   corev1.Node
		expect func(t *testing.T, spec GPUSpec)
	}{
		{
			name: "Full labels with allocatable GPUs",
			node: makeNode("gpu-node-1", map[string]string{
				LabelGPUProduct:         "NVIDIA-A100-SXM4-80GB",
				LabelGPUFamily:          "ampere",
				LabelGPUMemory:          "81920",
				LabelGPUComputeCapMajor: "8",
				LabelGPUComputeCapMinor: "0",
			}, 8),
			expect: func(t *testing.T, spec GPUSpec) {
				t.Helper()
				assertEqual(t, "Product", spec.Product, "NVIDIA-A100-SXM4-80GB")
				assertEqual(t, "ArchFamily", spec.ArchFamily, "Ampere")
				assertEqual(t, "ComputeMajor", spec.ComputeMajor, 8)
				assertEqual(t, "ComputeMinor", spec.ComputeMinor, 0)
				assertEqual(t, "Count", spec.Count, int64(8))
				// VRAM from label: 81920 MiB
				if spec.VRAMBytes != 81920*1024*1024 {
					t.Errorf("VRAMBytes: got %d, want %d", spec.VRAMBytes, 81920*1024*1024)
				}
				// Bandwidth filled by fallback lookup
				if spec.MemoryBandwidthGBps == 0 {
					t.Error("MemoryBandwidthGBps should be filled from fallback lookup")
				}
				// SM count filled by fallback lookup
				if spec.SMCount == 0 {
					t.Error("SMCount should be filled from fallback lookup")
				}
			},
		},
		{
			name: "Product label only triggers fallback",
			node: makeNode("gpu-node-2", map[string]string{
				LabelGPUProduct: "NVIDIA-T4",
			}, 1),
			expect: func(t *testing.T, spec GPUSpec) {
				t.Helper()
				assertEqual(t, "Product", spec.Product, "NVIDIA-T4")
				assertEqual(t, "ArchFamily", spec.ArchFamily, "Turing")
				assertEqual(t, "ComputeMajor", spec.ComputeMajor, 7)
				assertEqual(t, "ComputeMinor", spec.ComputeMinor, 5)
				assertEqual(t, "SMCount", spec.SMCount, 40)
				assertEqual(t, "Count", spec.Count, int64(1))
				if spec.VRAMBytes == 0 {
					t.Error("VRAMBytes should be filled from fallback")
				}
				if spec.MemoryBandwidthGBps == 0 {
					t.Error("MemoryBandwidthGBps should be filled from fallback")
				}
			},
		},
		{
			name: "GPU count falls back to label when allocatable is zero",
			node: makeNode("gpu-node-3", map[string]string{
				LabelGPUProduct: "NVIDIA-A100-SXM4-80GB",
				LabelGPUCount:   "4",
			}, 0),
			expect: func(t *testing.T, spec GPUSpec) {
				t.Helper()
				assertEqual(t, "Count", spec.Count, int64(4))
			},
		},
		{
			name: "No labels returns zero-valued spec",
			node: makeNode("cpu-node", nil, 0),
			expect: func(t *testing.T, spec GPUSpec) {
				t.Helper()
				assertEqual(t, "Product", spec.Product, "")
				assertEqual(t, "ArchFamily", spec.ArchFamily, "")
				assertEqual(t, "Count", spec.Count, int64(0))
			},
		},
		{
			name: "Labels take precedence over fallback for VRAM and compute capability",
			node: makeNode("gpu-node-4", map[string]string{
				LabelGPUProduct:         "NVIDIA-A100-SXM4-80GB",
				LabelGPUFamily:          "ampere",
				LabelGPUMemory:          "40960",
				LabelGPUComputeCapMajor: "8",
				LabelGPUComputeCapMinor: "6",
			}, 2),
			expect: func(t *testing.T, spec GPUSpec) {
				t.Helper()
				// Label values win over fallback.
				if spec.VRAMBytes != 40960*1024*1024 {
					t.Errorf("VRAMBytes: got %d, want %d (from label)", spec.VRAMBytes, 40960*1024*1024)
				}
				assertEqual(t, "ComputeMajor", spec.ComputeMajor, 8)
				assertEqual(t, "ComputeMinor", spec.ComputeMinor, 6)
			},
		},
		{
			name: "Unknown product does not crash",
			node: makeNode("gpu-node-5", map[string]string{
				LabelGPUProduct: "SOME-FUTURE-GPU-9000",
			}, 1),
			expect: func(t *testing.T, spec GPUSpec) {
				t.Helper()
				assertEqual(t, "Product", spec.Product, "SOME-FUTURE-GPU-9000")
				// No fallback data available.
				assertEqual(t, "SMCount", spec.SMCount, 0)
				if spec.MemoryBandwidthGBps != 0 {
					t.Errorf("MemoryBandwidthGBps should be 0 for unknown GPU, got %f", spec.MemoryBandwidthGBps)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := ExtractGPUSpec(tt.node)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.expect(t, spec)
		})
	}
}

// ---------------------------------------------------------------------------
// IsGPUNode
// ---------------------------------------------------------------------------

func TestIsGPUNode(t *testing.T) {
	tests := []struct {
		name   string
		node   corev1.Node
		expect bool
	}{
		{
			name:   "Node with allocatable GPUs",
			node:   makeNode("gpu-1", nil, 2),
			expect: true,
		},
		{
			name: "Node with product label only",
			node: makeNode("gpu-2", map[string]string{
				LabelGPUProduct: "NVIDIA-T4",
			}, 0),
			expect: true,
		},
		{
			name:   "Node without GPUs or labels",
			node:   makeNode("cpu-1", nil, 0),
			expect: false,
		},
		{
			name: "Node with unrelated labels",
			node: makeNode("cpu-2", map[string]string{
				"topology.kubernetes.io/zone": "us-east-1a",
			}, 0),
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsGPUNode(tt.node)
			if got != tt.expect {
				t.Errorf("IsGPUNode() = %v, want %v", got, tt.expect)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FilterGPUNodes
// ---------------------------------------------------------------------------

func TestFilterGPUNodes(t *testing.T) {
	nodes := []corev1.Node{
		makeNode("no-gpu", nil, 0),
		makeNode("one-gpu", map[string]string{LabelGPUProduct: "NVIDIA-T4"}, 1),
		makeNode("four-gpus", map[string]string{LabelGPUProduct: "NVIDIA-A100-SXM4-80GB"}, 4),
		makeNode("eight-gpus", map[string]string{LabelGPUProduct: "NVIDIA-A100-SXM4-80GB"}, 8),
	}

	tests := []struct {
		name      string
		minGPUs   int64
		wantLen   int
		wantNames []string
	}{
		{
			name:      "minGPUs=1 filters out no-gpu node",
			minGPUs:   1,
			wantLen:   3,
			wantNames: []string{"one-gpu", "four-gpus", "eight-gpus"},
		},
		{
			name:      "minGPUs=4 keeps nodes with >= 4",
			minGPUs:   4,
			wantLen:   2,
			wantNames: []string{"four-gpus", "eight-gpus"},
		},
		{
			name:      "minGPUs=8 keeps only eight-gpus",
			minGPUs:   8,
			wantLen:   1,
			wantNames: []string{"eight-gpus"},
		},
		{
			name:    "minGPUs=16 filters everything",
			minGPUs: 16,
			wantLen: 0,
		},
		{
			name:      "minGPUs=0 keeps all nodes",
			minGPUs:   0,
			wantLen:   4,
			wantNames: []string{"no-gpu", "one-gpu", "four-gpus", "eight-gpus"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterGPUNodes(nodes, tt.minGPUs)
			if len(got) != tt.wantLen {
				t.Fatalf("FilterGPUNodes() returned %d nodes, want %d", len(got), tt.wantLen)
			}
			for i, name := range tt.wantNames {
				if got[i].Name != name {
					t.Errorf("node[%d].Name = %q, want %q", i, got[i].Name, name)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// normalizeProduct (internal, but important for lookup correctness)
// ---------------------------------------------------------------------------

func TestNormalizeProduct(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"NVIDIA-A100-SXM4-80GB", "a100-sxm4-80gb"},
		{"NVIDIA A100 SXM4 80GB", "a100-sxm4-80gb"},
		{"NVIDIA-T4", "t4"},
		{"Tesla-V100-SXM2", "tesla-v100-sxm2"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeProduct(tt.input)
			if got != tt.want {
				t.Errorf("normalizeProduct(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// normalizeArch (internal)
// ---------------------------------------------------------------------------

func TestNormalizeArch(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"ampere", "Ampere"},
		{"hopper", "Hopper"},
		{"Turing", "Turing"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeArch(tt.input)
			if got != tt.want {
				t.Errorf("normalizeArch(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// test helpers
// ---------------------------------------------------------------------------

func assertEqual[T comparable](t *testing.T, field string, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %v, want %v", field, got, want)
	}
}
