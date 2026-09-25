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
	"cmp"
	"context"
	"slices"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/chirichexe/kratos/api/v1alpha1"
)

// ---------------------------------------------------------------------------
// Scoring constants
// ---------------------------------------------------------------------------

const (
	// maxNormalizedScore is the upper bound for any individual scoring
	// component before weights are applied.
	maxNormalizedScore int64 = 1000

	// Reference values used for normalization. These represent the high-end
	// of current datacenter hardware so that scores distribute well across
	// the 0–1000 range. Values above the reference are clamped to 1000.
	refSMCount             = 132             // H100-SXM
	refBandwidthGBps       = 4800            // H200
	refVRAMBytes     int64 = 141 * (1 << 30) // H200 141 GiB
	refComputeCap          = 9*10 + 0        // CC 9.0 (Hopper) → 90
)

// Breakdown keys for the individual scoring components.
const (
	componentSM        = "sm"
	componentBandwidth = "bandwidth"
	componentVRAM      = "vram"
	componentCompute   = "compute"
)

// ---------------------------------------------------------------------------
// Weight profiles per bound-type
// ---------------------------------------------------------------------------

// weights specifies how much each scoring component contributes to the
// total score for a given WorkloadBoundType.
type weights struct {
	sm        int64
	bandwidth int64
	vram      int64
	compute   int64
}

// weightProfiles maps each bound type to its weight set.
// All weights within a profile should sum to 100 for readability,
// though this is not a hard requirement (they are relative).
var weightProfiles = map[v1alpha1.WorkloadBoundType]weights{
	v1alpha1.WorkloadBoundCompute: {sm: 40, bandwidth: 10, vram: 15, compute: 35},
	v1alpha1.WorkloadBoundMemory:  {sm: 10, bandwidth: 45, vram: 30, compute: 15},
	v1alpha1.WorkloadBoundMixed:   {sm: 25, bandwidth: 25, vram: 25, compute: 25},
	v1alpha1.WorkloadBoundUnknown: {sm: 25, bandwidth: 25, vram: 25, compute: 25},
}

// defaultWeights is the balanced profile used when the bound type is empty
// or does not match any known value.
var defaultWeights = weights{sm: 25, bandwidth: 25, vram: 25, compute: 25}

// ---------------------------------------------------------------------------
// WeightedScorer
// ---------------------------------------------------------------------------

// WeightedScorer implements NodeScorer using a weighted linear combination
// of normalized GPU hardware metrics. It is safe for concurrent use.
type WeightedScorer struct {
	log logr.Logger
}

// NewWeightedScorer returns a WeightedScorer that logs to the given logger.
func NewWeightedScorer(log logr.Logger) *WeightedScorer {
	return &WeightedScorer{log: log}
}

// compile-time interface check
var _ NodeScorer = (*WeightedScorer)(nil)

// ScoreNodes scores every node against the given WorkloadProfile and returns
// the results in descending score order. Nodes whose GPUSpec cannot be
// extracted or that have zero GPUs are omitted from the result.
//
// The input slices and profile are never mutated.
func (s *WeightedScorer) ScoreNodes(
	ctx context.Context,
	profile *v1alpha1.WorkloadProfile,
	nodes []corev1.Node,
) ([]ScoredNode, error) {
	w := weightsForProfile(profile)
	s.log.V(1).Info("Resolved scoring weights",
		"boundType", boundTypeOf(profile),
		"sm", w.sm, "bandwidth", w.bandwidth,
		"vram", w.vram, "compute", w.compute,
	)

	scored := make([]ScoredNode, 0, len(nodes))
	for _, node := range nodes {
		spec, err := ExtractGPUSpec(node)
		if err != nil {
			s.log.V(1).Info("Skipped node, could not extract GPU spec",
				"node", node.Name, "error", err)
			continue
		}

		if spec.Count == 0 {
			s.log.V(1).Info("Skipped node with zero GPUs", "node", node.Name)
			continue
		}

		sn := scoreNode(node.Name, spec, w)
		scored = append(scored, sn)

		s.log.V(1).Info("Scored node",
			"node", sn.NodeName,
			"totalScore", sn.TotalScore,
			"breakdown", sn.Breakdown,
		)
	}

	sortScoredNodes(scored)
	return scored, nil
}

// ---------------------------------------------------------------------------
// Scoring helpers (pure functions, no side effects)
// ---------------------------------------------------------------------------

// scoreNode computes the weighted score for a single node.
func scoreNode(name string, spec GPUSpec, w weights) ScoredNode {
	smScore := normalize(int64(spec.SMCount), int64(refSMCount))
	bwScore := normalize(int64(spec.MemoryBandwidthGBps), int64(refBandwidthGBps))
	vramScore := normalize(spec.VRAMBytes, refVRAMBytes)
	computeScore := normalize(int64(spec.ComputeMajor*10+spec.ComputeMinor), int64(refComputeCap))

	breakdown := map[string]int64{
		componentSM:        smScore * w.sm / 100,
		componentBandwidth: bwScore * w.bandwidth / 100,
		componentVRAM:      vramScore * w.vram / 100,
		componentCompute:   computeScore * w.compute / 100,
	}

	var total int64
	for _, v := range breakdown {
		total += v
	}

	return ScoredNode{
		NodeName:   name,
		TotalScore: total,
		Breakdown:  breakdown,
	}
}

// normalize maps a raw value to the 0–maxNormalizedScore range using
// linear interpolation against a reference maximum. Values at or above
// the reference are clamped to maxNormalizedScore.
func normalize(value, reference int64) int64 {
	if reference <= 0 || value <= 0 {
		return 0
	}
	return min(value*maxNormalizedScore/reference, maxNormalizedScore)
}

// sortScoredNodes sorts the slice in-place: descending by TotalScore,
// then ascending lexically by NodeName for deterministic tie-breaking.
func sortScoredNodes(nodes []ScoredNode) {
	slices.SortFunc(nodes, func(a, b ScoredNode) int {
		if c := cmp.Compare(b.TotalScore, a.TotalScore); c != 0 {
			return c
		}
		return cmp.Compare(a.NodeName, b.NodeName)
	})
}

// weightsForProfile returns the weight set corresponding to the profile's
// bound type. If the profile is nil or its bound type is unrecognised,
// the balanced default weights are used.
func weightsForProfile(profile *v1alpha1.WorkloadProfile) weights {
	if profile == nil {
		return defaultWeights
	}
	bt := profile.Status.BoundType
	if w, ok := weightProfiles[bt]; ok {
		return w
	}
	return defaultWeights
}

// boundTypeOf safely extracts the BoundType string for logging.
func boundTypeOf(profile *v1alpha1.WorkloadProfile) string {
	if profile == nil {
		return "<nil>"
	}
	if profile.Status.BoundType == "" {
		return "<empty>"
	}
	return string(profile.Status.BoundType)
}
