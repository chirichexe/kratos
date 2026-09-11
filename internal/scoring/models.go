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
	"context"

	corev1 "k8s.io/api/core/v1"

	"github.com/chirichexe/kratos/api/v1alpha1"
)

// GPUSpec is a normalized representation of a node's GPU capabilities.
// Fields may be zero-valued when the information could not be extracted
// from node labels or the built-in lookup table.
type GPUSpec struct {
	// Product is the full GPU product name (e.g. "NVIDIA A100-SXM4-80GB").
	Product string

	// ArchFamily is the GPU micro-architecture family (e.g. "Ampere", "Hopper").
	ArchFamily string

	// ComputeMajor is the major version of the CUDA compute capability (e.g. 8).
	ComputeMajor int

	// ComputeMinor is the minor version of the CUDA compute capability (e.g. 0).
	ComputeMinor int

	// SMCount is the number of streaming multiprocessors.
	SMCount int

	// VRAMBytes is the total amount of GPU memory in bytes.
	VRAMBytes int64

	// MemoryBandwidthGBps is the theoretical peak memory bandwidth in GB/s.
	MemoryBandwidthGBps float64

	// Count is the number of allocatable GPUs on the node.
	Count int64
}

// ScoredNode pairs a node name with a composite score and a human-readable
// breakdown that explains how individual scoring components contributed.
type ScoredNode struct {
	// NodeName is the Kubernetes node name.
	NodeName string

	// TotalScore is the aggregate score across all scoring components.
	TotalScore int64

	// Breakdown maps component names (e.g. "vram", "bandwidth", "compute")
	// to their individual scores, for debugging and observability.
	Breakdown map[string]int64
}

// NodeScorer ranks a set of Kubernetes nodes against a WorkloadProfile.
// Implementations provide the concrete scoring heuristics; this interface
// allows the controller layer to remain agnostic of the strategy used.
type NodeScorer interface {
	ScoreNodes(ctx context.Context, profile *v1alpha1.WorkloadProfile, nodes []corev1.Node) ([]ScoredNode, error)
}
