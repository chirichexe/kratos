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
	"testing"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/chirichexe/kratos/api/v1alpha1"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// gpuNode builds a corev1.Node labelled with an NVIDIA product and the
// given number of allocatable GPUs.
func gpuNode(name, product string, gpus int64) corev1.Node {
	return corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				LabelGPUProduct: product,
			},
		},
		Status: corev1.NodeStatus{
			Allocatable: corev1.ResourceList{
				corev1.ResourceName(ResourceNVIDIAGPU): *resource.NewQuantity(gpus, resource.DecimalSI),
			},
		},
	}
}

// profileWithBound builds a minimal WorkloadProfile with the given bound type.
func profileWithBound(bt v1alpha1.WorkloadBoundType) *v1alpha1.WorkloadProfile {
	return &v1alpha1.WorkloadProfile{
		Status: v1alpha1.WorkloadProfileStatus{
			BoundType: bt,
		},
	}
}

// nodeNames extracts node names from a slice of ScoredNode.
func nodeNames(nodes []ScoredNode) []string {
	names := make([]string, len(nodes))
	for i, n := range nodes {
		names[i] = n.NodeName
	}
	return names
}

// assertOrder checks that the scored node names appear in the expected order.
func assertOrder(t *testing.T, got []ScoredNode, wantNames ...string) {
	t.Helper()
	gotNames := nodeNames(got)
	if len(gotNames) != len(wantNames) {
		t.Fatalf("got %d nodes %v, want %d nodes %v", len(gotNames), gotNames, len(wantNames), wantNames)
	}
	for i := range wantNames {
		if gotNames[i] != wantNames[i] {
			t.Errorf("position %d: got %q, want %q (full order: %v)", i, gotNames[i], wantNames[i], gotNames)
		}
	}
}

// ---------------------------------------------------------------------------
// ScoreNodes — compute-bound workloads
// ---------------------------------------------------------------------------

func TestScoreNodes_ComputeBound(t *testing.T) {
	scorer := NewWeightedScorer(logr.Discard())
	profile := profileWithBound(v1alpha1.WorkloadBoundCompute)

	nodes := []corev1.Node{
		gpuNode("t4-node", productT4, 1),           // Turing 7.5, 40 SMs
		gpuNode("a100-node", productA100, 1),       // Ampere 8.0, 108 SMs
		gpuNode("h100-node", "NVIDIA-H100-SXM", 1), // Hopper 9.0, 132 SMs
	}

	got, err := scorer.ScoreNodes(context.Background(), profile, nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// H100 should win (most SMs + highest compute capability),
	// then A100, then T4.
	assertOrder(t, got, "h100-node", "a100-node", "t4-node")

	// Verify scores are strictly descending.
	for i := 1; i < len(got); i++ {
		if got[i].TotalScore >= got[i-1].TotalScore {
			t.Errorf("expected descending scores: node %q (%d) >= node %q (%d)",
				got[i].NodeName, got[i].TotalScore,
				got[i-1].NodeName, got[i-1].TotalScore)
		}
	}

	// For compute-bound, the "sm" and "compute" breakdown components
	// should dominate.
	h100 := got[0]
	if h100.Breakdown[componentSM]+h100.Breakdown[componentCompute] <= h100.Breakdown[componentBandwidth]+h100.Breakdown[componentVRAM] {
		t.Errorf("expected compute-bound profile to weight sm+compute > bandwidth+vram, got %v", h100.Breakdown)
	}
}

// ---------------------------------------------------------------------------
// ScoreNodes — memory-bound workloads
// ---------------------------------------------------------------------------

func TestScoreNodes_MemoryBound(t *testing.T) {
	scorer := NewWeightedScorer(logr.Discard())
	profile := profileWithBound(v1alpha1.WorkloadBoundMemory)

	nodes := []corev1.Node{
		gpuNode("t4-node", productT4, 1),       // 320 GB/s, 16 GiB
		gpuNode("a100-node", productA100, 1),   // 2039 GB/s, 80 GiB
		gpuNode("h200-node", "NVIDIA-H200", 1), // 4800 GB/s, 141 GiB
	}

	got, err := scorer.ScoreNodes(context.Background(), profile, nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// H200 should win (highest bandwidth + most VRAM), then A100, then T4.
	assertOrder(t, got, "h200-node", "a100-node", "t4-node")

	// For memory-bound, bandwidth+vram should dominate.
	h200 := got[0]
	if h200.Breakdown[componentBandwidth]+h200.Breakdown[componentVRAM] <= h200.Breakdown[componentSM]+h200.Breakdown[componentCompute] {
		t.Errorf("expected memory-bound profile to weight bandwidth+vram > sm+compute, got %v", h200.Breakdown)
	}
}

// ---------------------------------------------------------------------------
// ScoreNodes — mixed / unknown workloads
// ---------------------------------------------------------------------------

func TestScoreNodes_MixedWorkload(t *testing.T) {
	scorer := NewWeightedScorer(logr.Discard())
	profile := profileWithBound(v1alpha1.WorkloadBoundMixed)

	nodes := []corev1.Node{
		gpuNode("t4-node", productT4, 1),
		gpuNode("a100-node", productA100, 1),
	}

	got, err := scorer.ScoreNodes(context.Background(), profile, nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// A100 is better than T4 on every dimension.
	assertOrder(t, got, "a100-node", "t4-node")

	// All breakdown components should have equal weights (25 each).
	a100 := got[0]
	for _, key := range []string{componentSM, componentBandwidth, componentVRAM, componentCompute} {
		if a100.Breakdown[key] == 0 {
			t.Errorf("expected non-zero breakdown for %q, got 0", key)
		}
	}
}

func TestScoreNodes_UnknownBoundType(t *testing.T) {
	scorer := NewWeightedScorer(logr.Discard())
	profile := profileWithBound(v1alpha1.WorkloadBoundUnknown)

	nodes := []corev1.Node{
		gpuNode("a100-node", productA100, 1),
		gpuNode("t4-node", productT4, 1),
	}

	got, err := scorer.ScoreNodes(context.Background(), profile, nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Same balanced weights as mixed: A100 > T4.
	assertOrder(t, got, "a100-node", "t4-node")
}

// ---------------------------------------------------------------------------
// ScoreNodes — nil profile uses default balanced weights
// ---------------------------------------------------------------------------

func TestScoreNodes_NilProfile(t *testing.T) {
	scorer := NewWeightedScorer(logr.Discard())

	nodes := []corev1.Node{
		gpuNode("a100-node", productA100, 1),
		gpuNode("t4-node", productT4, 1),
	}

	got, err := scorer.ScoreNodes(context.Background(), nil, nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertOrder(t, got, "a100-node", "t4-node")
}

// ---------------------------------------------------------------------------
// ScoreNodes — ineligible / zero-GPU nodes are filtered out
// ---------------------------------------------------------------------------

func TestScoreNodes_FiltersIneligibleNodes(t *testing.T) {
	scorer := NewWeightedScorer(logr.Discard())
	profile := profileWithBound(v1alpha1.WorkloadBoundCompute)

	nodes := []corev1.Node{
		// CPU-only node: no GPUs, no product label.
		{ObjectMeta: metav1.ObjectMeta{Name: "cpu-node"}},
		// GPU node with product label but zero allocatable GPUs.
		gpuNode("label-only-node", productA100, 0),
		// Valid GPU node.
		gpuNode("real-gpu-node", productT4, 2),
	}

	got, err := scorer.ScoreNodes(context.Background(), profile, nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 scored node, got %d: %v", len(got), nodeNames(got))
	}
	if got[0].NodeName != "real-gpu-node" {
		t.Errorf("expected real-gpu-node, got %q", got[0].NodeName)
	}
}

// ---------------------------------------------------------------------------
// ScoreNodes — empty node list
// ---------------------------------------------------------------------------

func TestScoreNodes_EmptyNodeList(t *testing.T) {
	scorer := NewWeightedScorer(logr.Discard())
	profile := profileWithBound(v1alpha1.WorkloadBoundCompute)

	got, err := scorer.ScoreNodes(context.Background(), profile, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty result, got %d nodes", len(got))
	}
}

// ---------------------------------------------------------------------------
// ScoreNodes — heterogeneous cluster
// ---------------------------------------------------------------------------

func TestScoreNodes_HeterogeneousCluster(t *testing.T) {
	scorer := NewWeightedScorer(logr.Discard())
	profile := profileWithBound(v1alpha1.WorkloadBoundCompute)

	nodes := []corev1.Node{
		gpuNode("p100-node", "NVIDIA-P100", 1),
		gpuNode("v100-node", "NVIDIA-V100-SXM2", 1),
		gpuNode("t4-node", productT4, 1),
		gpuNode("a100-node", productA100, 1),
		gpuNode("h100-node", "NVIDIA-H100-SXM", 1),
		gpuNode("l40s-node", "NVIDIA-L40S", 1),
		// CPU node — will be filtered out.
		{ObjectMeta: metav1.ObjectMeta{Name: "cpu-node"}},
	}

	got, err := scorer.ScoreNodes(context.Background(), profile, nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 6 GPU nodes survive filtering.
	if len(got) != 6 {
		t.Fatalf("expected 6 scored nodes, got %d: %v", len(got), nodeNames(got))
	}

	// Top node should be H100 or L40S (both are top-tier for compute).
	// H100 has CC 9.0 + 132 SMs; L40S has CC 8.9 + 142 SMs.
	// With compute-bound weights (sm=40, compute=35), H100 should win
	// because CC 9.0 contributes more than the 10 extra SMs on L40S.
	if got[0].NodeName != "h100-node" {
		t.Errorf("expected h100-node at position 0, got %q", got[0].NodeName)
	}

	// Scores should be strictly descending (or equal with lexical tie-break).
	for i := 1; i < len(got); i++ {
		if got[i].TotalScore > got[i-1].TotalScore {
			t.Errorf("scores not sorted: %q (%d) > %q (%d)",
				got[i].NodeName, got[i].TotalScore,
				got[i-1].NodeName, got[i-1].TotalScore)
		}
	}
}

// ---------------------------------------------------------------------------
// Deterministic tie-breaking
// ---------------------------------------------------------------------------

func TestScoreNodes_DeterministicTieBreaking(t *testing.T) {
	scorer := NewWeightedScorer(logr.Discard())
	profile := profileWithBound(v1alpha1.WorkloadBoundMixed)

	// Two identical GPU nodes — same product, same GPUs.
	nodes := []corev1.Node{
		gpuNode("node-b", productA100, 1),
		gpuNode("node-a", productA100, 1),
	}

	got, err := scorer.ScoreNodes(context.Background(), profile, nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Same score → lexical order: node-a before node-b.
	assertOrder(t, got, "node-a", "node-b")

	if got[0].TotalScore != got[1].TotalScore {
		t.Errorf("expected equal scores for identical nodes, got %d vs %d",
			got[0].TotalScore, got[1].TotalScore)
	}
}

// ---------------------------------------------------------------------------
// Input immutability
// ---------------------------------------------------------------------------

func TestScoreNodes_DoesNotMutateInputs(t *testing.T) {
	scorer := NewWeightedScorer(logr.Discard())
	profile := profileWithBound(v1alpha1.WorkloadBoundCompute)

	original := []corev1.Node{
		gpuNode("t4-node", productT4, 1),
		gpuNode("a100-node", productA100, 1),
	}

	// Keep a copy of the original order.
	firstBefore := original[0].Name
	secondBefore := original[1].Name

	_, err := scorer.ScoreNodes(context.Background(), profile, original)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The input slice must not have been reordered.
	if original[0].Name != firstBefore || original[1].Name != secondBefore {
		t.Errorf("input slice was mutated: got [%s, %s], want [%s, %s]",
			original[0].Name, original[1].Name, firstBefore, secondBefore)
	}
}

// ---------------------------------------------------------------------------
// Breakdown completeness
// ---------------------------------------------------------------------------

func TestScoreNodes_BreakdownContainsAllComponents(t *testing.T) {
	scorer := NewWeightedScorer(logr.Discard())
	profile := profileWithBound(v1alpha1.WorkloadBoundCompute)

	nodes := []corev1.Node{
		gpuNode("a100-node", productA100, 1),
	}

	got, err := scorer.ScoreNodes(context.Background(), profile, nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 node, got %d", len(got))
	}

	expectedKeys := []string{componentSM, componentBandwidth, componentVRAM, componentCompute}
	for _, key := range expectedKeys {
		if _, ok := got[0].Breakdown[key]; !ok {
			t.Errorf("missing breakdown key %q", key)
		}
	}

	// TotalScore must equal the sum of all breakdown values.
	var sum int64
	for _, v := range got[0].Breakdown {
		sum += v
	}
	if got[0].TotalScore != sum {
		t.Errorf("TotalScore (%d) != sum of breakdown (%d)", got[0].TotalScore, sum)
	}
}

// ---------------------------------------------------------------------------
// normalize
// ---------------------------------------------------------------------------

func TestNormalize(t *testing.T) {
	tests := []struct {
		name      string
		value     int64
		reference int64
		want      int64
	}{
		{"zero value", 0, 100, 0},
		{"zero reference", 50, 0, 0},
		{"half of reference", 50, 100, 500},
		{"at reference", 100, 100, 1000},
		{"above reference clamped", 200, 100, 1000},
		{"negative value", -10, 100, 0},
		{"negative reference", 50, -100, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalize(tt.value, tt.reference)
			if got != tt.want {
				t.Errorf("normalize(%d, %d) = %d, want %d", tt.value, tt.reference, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// weightsForProfile
// ---------------------------------------------------------------------------

func TestWeightsForProfile(t *testing.T) {
	tests := []struct {
		name    string
		profile *v1alpha1.WorkloadProfile
		wantSM  int64
	}{
		{"compute-bound", profileWithBound(v1alpha1.WorkloadBoundCompute), 40},
		{"memory-bound", profileWithBound(v1alpha1.WorkloadBoundMemory), 10},
		{"mixed", profileWithBound(v1alpha1.WorkloadBoundMixed), 25},
		{"unknown", profileWithBound(v1alpha1.WorkloadBoundUnknown), 25},
		{"nil profile", nil, 25},
		{"empty bound type", profileWithBound(""), 25},
		{"unrecognised bound type", profileWithBound("something-else"), 25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := weightsForProfile(tt.profile)
			if got.sm != tt.wantSM {
				t.Errorf("sm weight = %d, want %d", got.sm, tt.wantSM)
			}
		})
	}
}
