// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConstructAllocationID(t *testing.T) {
	tests := []struct {
		name       string
		ensembleID string
		allocName  string
		expected   string
	}{
		{
			name:       "basic allocation",
			ensembleID: "test-ensemble",
			allocName:  "node1.alloc1",
			expected:   "test-ensemble_node1.alloc1",
		},
		{
			name:       "standby allocation",
			ensembleID: "test-ensemble",
			allocName:  "node1-standby-1.alloc1",
			expected:   "test-ensemble_node1-standby-1.alloc1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConstructAllocationID(tt.ensembleID, tt.allocName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAllocationNameFromID(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		expected string
	}{
		{
			name:     "basic allocation",
			id:       "test-ensemble_node1.alloc1",
			expected: "node1.alloc1",
		},
		{
			name:     "standby allocation",
			id:       "test-ensemble_node1-standby-1.alloc1",
			expected: "node1-standby-1.alloc1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AllocationNameFromID(tt.id)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnsembleIDFromAllocationID(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		expected string
	}{
		{
			name:     "basic allocation",
			id:       "test-ensemble_node1.alloc1",
			expected: "test-ensemble",
		},
		{
			name:     "standby allocation",
			id:       "test-ensemble_node1-standby-1.alloc1",
			expected: "test-ensemble",
		},
		{
			name:     "no underscore",
			id:       "test-ensemble",
			expected: "test-ensemble",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EnsembleIDFromAllocationID(tt.id)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAllocationIdentifier_String(t *testing.T) {
	tests := []struct {
		name     string
		aid      AllocationIdentifier
		expected string
	}{
		{
			name: "primary allocation",
			aid: AllocationIdentifier{
				EnsembleID:     "test-ensemble",
				NodeID:         "node1",
				AllocationName: "alloc1",
				IsStandby:      false,
				StandbyIndex:   0,
			},
			expected: "test-ensemble_node1.alloc1",
		},
		{
			name: "standby allocation",
			aid: AllocationIdentifier{
				EnsembleID:     "test-ensemble",
				NodeID:         "node1-standby-1",
				AllocationName: "alloc1",
				IsStandby:      true,
				StandbyIndex:   1,
			},
			expected: "test-ensemble_node1-standby-1.alloc1",
		},
		{
			name: "standby allocation index 2",
			aid: AllocationIdentifier{
				EnsembleID:     "test-ensemble",
				NodeID:         "node1-standby-2",
				AllocationName: "alloc1",
				IsStandby:      true,
				StandbyIndex:   2,
			},
			expected: "test-ensemble_node1-standby-2.alloc1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.aid.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAllocationIdentifier_ManifestKey(t *testing.T) {
	tests := []struct {
		name     string
		aid      AllocationIdentifier
		expected string
	}{
		{
			name: "primary allocation",
			aid: AllocationIdentifier{
				EnsembleID:     "test-ensemble",
				NodeID:         "node1",
				AllocationName: "alloc1",
				IsStandby:      false,
				StandbyIndex:   0,
			},
			expected: "node1.alloc1",
		},
		{
			name: "standby allocation",
			aid: AllocationIdentifier{
				EnsembleID:     "test-ensemble",
				NodeID:         "node1-standby-1",
				AllocationName: "alloc1",
				IsStandby:      true,
				StandbyIndex:   1,
			},
			expected: "node1-standby-1.alloc1",
		},
		{
			name: "standby allocation index 2",
			aid: AllocationIdentifier{
				EnsembleID:     "test-ensemble",
				NodeID:         "node1-standby-2",
				AllocationName: "alloc1",
				IsStandby:      true,
				StandbyIndex:   2,
			},
			expected: "node1-standby-2.alloc1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.aid.ManifestKey()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAllocationIdentifier_ConfigName(t *testing.T) {
	tests := []struct {
		name     string
		aid      AllocationIdentifier
		expected string
	}{
		{
			name: "primary allocation",
			aid: AllocationIdentifier{
				EnsembleID:     "test-ensemble",
				NodeID:         "node1",
				AllocationName: "alloc1",
				IsStandby:      false,
				StandbyIndex:   0,
			},
			expected: "alloc1",
		},
		{
			name: "standby allocation",
			aid: AllocationIdentifier{
				EnsembleID:     "test-ensemble",
				NodeID:         "node1-standby-1",
				AllocationName: "alloc1",
				IsStandby:      true,
				StandbyIndex:   1,
			},
			expected: "alloc1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.aid.ConfigName()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewAllocationID(t *testing.T) {
	aid := NewAllocationID("test-ensemble", "node1", "alloc1")

	assert.Equal(t, "test-ensemble", aid.EnsembleID)
	assert.Equal(t, "node1", aid.NodeID)
	assert.Equal(t, "alloc1", aid.AllocationName)
	assert.False(t, aid.IsStandby)
	assert.Equal(t, 0, aid.StandbyIndex)
	assert.Equal(t, "test-ensemble_node1.alloc1", aid.String())
}

func TestNewStandbyAllocationID(t *testing.T) {
	aid := NewStandbyAllocationID("test-ensemble", "node1-standby-1", "alloc1", 1)

	assert.Equal(t, "test-ensemble", aid.EnsembleID)
	assert.Equal(t, "node1-standby-1", aid.NodeID)
	assert.Equal(t, "alloc1", aid.AllocationName)
	assert.True(t, aid.IsStandby)
	assert.Equal(t, 1, aid.StandbyIndex)
	assert.Equal(t, "test-ensemble_node1-standby-1.alloc1", aid.String())
}

func TestParseAllocationID(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		expected    AllocationIdentifier
		expectError bool
	}{
		{
			name: "primary allocation",
			id:   "test-ensemble_node1.alloc1",
			expected: AllocationIdentifier{
				EnsembleID:     "test-ensemble",
				NodeID:         "node1",
				AllocationName: "alloc1",
				IsStandby:      false,
				StandbyIndex:   0,
			},
			expectError: false,
		},
		{
			name: "standby allocation",
			id:   "test-ensemble_node1-standby-1.alloc1",
			expected: AllocationIdentifier{
				EnsembleID:     "test-ensemble",
				NodeID:         "node1-standby-1",
				AllocationName: "alloc1",
				IsStandby:      true,
				StandbyIndex:   1,
			},
			expectError: false,
		},
		{
			name: "standby allocation index 2",
			id:   "test-ensemble_node1-standby-2.alloc1",
			expected: AllocationIdentifier{
				EnsembleID:     "test-ensemble",
				NodeID:         "node1-standby-2",
				AllocationName: "alloc1",
				IsStandby:      true,
				StandbyIndex:   2,
			},
			expectError: false,
		},
		{
			name:        "invalid format - no underscore",
			id:          "invalid-id",
			expectError: true,
		},
		{
			name:        "invalid standby format",
			id:          "test-ensemble_node1-standby.alloc1",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseAllocationID(tt.id)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseManifestKey(t *testing.T) {
	tests := []struct {
		name        string
		manifestKey string
		ensembleID  string
		expected    AllocationIdentifier
		expectError bool
	}{
		{
			name:        "primary allocation",
			manifestKey: "node1.alloc1",
			ensembleID:  "test-ensemble",
			expected: AllocationIdentifier{
				EnsembleID:     "test-ensemble",
				NodeID:         "node1",
				AllocationName: "alloc1",
				IsStandby:      false,
				StandbyIndex:   0,
			},
			expectError: false,
		},
		{
			name:        "standby allocation",
			manifestKey: "node1-standby-1.alloc1",
			ensembleID:  "test-ensemble",
			expected: AllocationIdentifier{
				EnsembleID:     "test-ensemble",
				NodeID:         "node1-standby-1",
				AllocationName: "alloc1",
				IsStandby:      true,
				StandbyIndex:   1,
			},
			expectError: false,
		},
		{
			name:        "standby allocation index 2",
			manifestKey: "node1-standby-2.alloc1",
			ensembleID:  "test-ensemble",
			expected: AllocationIdentifier{
				EnsembleID:     "test-ensemble",
				NodeID:         "node1-standby-2",
				AllocationName: "alloc1",
				IsStandby:      true,
				StandbyIndex:   2,
			},
			expectError: false,
		},
		{
			name:        "invalid format",
			manifestKey: "invalid-key",
			ensembleID:  "test-ensemble",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseManifestKey(tt.manifestKey, tt.ensembleID)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseNodeName(t *testing.T) {
	tests := []struct {
		name              string
		nodeName          string
		expectedIsStandby bool
		expectedPrimary   string
		expectedIndex     int
	}{
		{
			name:              "primary node",
			nodeName:          "node1",
			expectedIsStandby: false,
			expectedPrimary:   "node1",
			expectedIndex:     0,
		},
		{
			name:              "standby node index 1",
			nodeName:          "node1-standby-1",
			expectedIsStandby: true,
			expectedPrimary:   "node1",
			expectedIndex:     1,
		},
		{
			name:              "standby node index 2",
			nodeName:          "node1-standby-2",
			expectedIsStandby: true,
			expectedPrimary:   "node1",
			expectedIndex:     2,
		},
		{
			name:              "invalid standby format",
			nodeName:          "node1-standby-invalid",
			expectedIsStandby: false,
			expectedPrimary:   "node1-standby-invalid",
			expectedIndex:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isStandby, primary, index := ParseNodeName(tt.nodeName)
			assert.Equal(t, tt.expectedIsStandby, isStandby)
			assert.Equal(t, tt.expectedPrimary, primary)
			assert.Equal(t, tt.expectedIndex, index)
		})
	}
}
