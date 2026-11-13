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

// FilterByConstraints returns the subset of nodes whose GPU satisfies both
// minComputeCapability (a "major.minor" CUDA compute capability string,
// e.g. "7.0") and minMemory (a Kubernetes quantity, e.g. "4Gi"). An empty
// string skips that particular check. Nodes whose GPU spec cannot be
// extracted never satisfy a non-empty constraint.
//
// The input slice is never mutated. FilterByConstraints only returns an
// error when a constraint string itself is malformed.
func FilterByConstraints(nodes []corev1.Node, minComputeCapability, minMemory string) ([]corev1.Node, error) {
	checkCompute := minComputeCapability != ""
	var minCompute int
	if checkCompute {
		major, minor, err := ParseComputeCapability(minComputeCapability)
		if err != nil {
			return nil, fmt.Errorf("minimumComputeCapability: %w", err)
		}
		minCompute = major*10 + minor
	}

	checkMemory := minMemory != ""
	var minMemoryBytes int64
	if checkMemory {
		qty, err := resource.ParseQuantity(minMemory)
		if err != nil {
			return nil, fmt.Errorf("minimumMemory: %w", err)
		}
		minMemoryBytes = qty.Value()
	}

	if !checkCompute && !checkMemory {
		return nodes, nil
	}

	result := make([]corev1.Node, 0, len(nodes))
	for _, node := range nodes {
		spec, err := ExtractGPUSpec(node)
		if err != nil {
			continue
		}
		if checkCompute && spec.ComputeMajor*10+spec.ComputeMinor < minCompute {
			continue
		}
		if checkMemory && spec.VRAMBytes < minMemoryBytes {
			continue
		}
		result = append(result, node)
	}
	return result, nil
}

// ParseComputeCapability parses a "major.minor" CUDA compute capability
// string (e.g. "7.0", "8.6") into its integer components. The minor
// component defaults to 0 when omitted (e.g. "8").
func ParseComputeCapability(s string) (major, minor int, err error) {
	parts := strings.SplitN(s, ".", 2)
	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid compute capability %q", s)
	}
	if len(parts) == 2 {
		minor, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, fmt.Errorf("invalid compute capability %q", s)
		}
	}
	return major, minor, nil
}
