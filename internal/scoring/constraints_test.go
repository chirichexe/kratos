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
	"strconv"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func nodeWithComputeAndMemory(name string, major, minor int, memMiB int64) corev1.Node {
	return makeNode(name, map[string]string{
		LabelGPUProduct:         productA100,
		LabelGPUFamily:          familyAmpere,
		LabelGPUComputeCapMajor: strconv.Itoa(major),
		LabelGPUComputeCapMinor: strconv.Itoa(minor),
		LabelGPUMemory:          strconv.FormatInt(memMiB, 10),
	}, 1)
}

func TestFilterByConstraints(t *testing.T) {
	nodeLowCompute := nodeWithComputeAndMemory(nodeOneGPU, 6, 0, 16*1024)        // CC 6.0, 16GiB
	nodeHighComputeLowMem := nodeWithComputeAndMemory(nodeFourGPU, 8, 0, 8*1024) // CC 8.0, 8GiB
	nodeHighBoth := nodeWithComputeAndMemory(nodeEightGPU, 9, 0, 80*1024)        // CC 9.0, 80GiB
	noLabelNode := makeNode("no-labels", nil, 0)

	nodes := []corev1.Node{nodeLowCompute, nodeHighComputeLowMem, nodeHighBoth, noLabelNode}

	tests := []struct {
		name        string
		minCompute  string
		minMemory   string
		wantNames   []string
		expectError bool
	}{
		{
			name:      "no constraints returns all nodes unchanged",
			wantNames: []string{nodeOneGPU, nodeFourGPU, nodeEightGPU, "no-labels"},
		},
		{
			name:       "compute capability filters out lower nodes",
			minCompute: "7.0",
			wantNames:  []string{nodeFourGPU, nodeEightGPU},
		},
		{
			name:      "memory filters out smaller nodes",
			minMemory: "32Gi",
			wantNames: []string{nodeEightGPU},
		},
		{
			name:       "both constraints combine with AND",
			minCompute: "8.0",
			minMemory:  "32Gi",
			wantNames:  []string{nodeEightGPU},
		},
		{
			name:      "no node satisfies an impossible constraint",
			minMemory: "9999Gi",
			wantNames: []string{},
		},
		{
			name:        "malformed compute capability returns an error",
			minCompute:  "not-a-number",
			expectError: true,
		},
		{
			name:        "malformed memory quantity returns an error",
			minMemory:   "not-a-quantity",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FilterByConstraints(nodes, tt.minCompute, tt.minMemory)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			gotNames := make([]string, 0, len(got))
			for _, n := range got {
				gotNames = append(gotNames, n.Name)
			}
			assertEqual(t, "node count", len(gotNames), len(tt.wantNames))
			for i, want := range tt.wantNames {
				if i >= len(gotNames) {
					break
				}
				assertEqual(t, "node name", gotNames[i], want)
			}
		})
	}
}

func TestParseComputeCapability(t *testing.T) {
	tests := []struct {
		in        string
		wantMajor int
		wantMinor int
		wantErr   bool
	}{
		{in: "7.0", wantMajor: 7, wantMinor: 0},
		{in: "8.6", wantMajor: 8, wantMinor: 6},
		{in: "9", wantMajor: 9, wantMinor: 0},
		{in: "x.0", wantErr: true},
		{in: "8.x", wantErr: true},
	}

	for _, tt := range tests {
		major, minor, err := parseComputeCapability(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseComputeCapability(%q): expected error, got nil", tt.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseComputeCapability(%q): unexpected error: %v", tt.in, err)
			continue
		}
		assertEqual(t, "major", major, tt.wantMajor)
		assertEqual(t, "minor", minor, tt.wantMinor)
	}
}
