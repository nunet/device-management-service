// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package orchestrator

import (
	"testing"

	"github.com/stretchr/testify/require"

	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
)

func TestNewConfigForRemovedNodes(t *testing.T) {
	tests := []struct {
		name           string
		oldConfig      jtypes.EnsembleConfig
		modifiedConfig jtypes.EnsembleConfig
		expected       jtypes.EnsembleConfig
		expectedErr    bool
	}{
		{
			name: "basic removed nodes",
			oldConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						node1: Node1Cfg,
						node2: Node2Cfg,
					},
					Allocations: map[string]jtypes.AllocationConfig{
						alloc1: Alloc1Cfg,
						alloc2: Alloc2Cfg,
					},
				},
			},
			modifiedConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						node1: Node1Cfg,
					},
					Allocations: map[string]jtypes.AllocationConfig{
						alloc1: Alloc1Cfg,
					},
					Scripts: testScripts,
					Keys:    testKeys,
				},
			},
			expected: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						node2: Node2Cfg,
					},
					Allocations: map[string]jtypes.AllocationConfig{
						alloc2: Alloc2Cfg,
					},
					Scripts: testScripts,
					Keys:    testKeys,
				},
			},
		},
		{
			name: "multiple removed nodes with allocations",
			oldConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						node1: Node1Cfg,
						node2: Node2Cfg,
						node3: Node3Cfg,
					},
					Allocations: map[string]jtypes.AllocationConfig{
						alloc1: Alloc1Cfg,
						alloc2: Alloc2Cfg,
						alloc3: Alloc3Cfg,
						alloc4: Alloc4Cfg,
					},
				},
			},
			modifiedConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						node1: Node1Cfg,
					},
					Allocations: map[string]jtypes.AllocationConfig{
						alloc1: Alloc1Cfg,
					},
				},
			},
			expected: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						node2: Node2Cfg,
						node3: Node3Cfg,
					},
					Allocations: map[string]jtypes.AllocationConfig{
						alloc2: Alloc2Cfg,
						alloc3: Alloc3Cfg,
						alloc4: Alloc4Cfg,
					},
				},
			},
		},
		{
			name: "no removed nodes",
			oldConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						node1: Node1Cfg,
						node2: Node2Cfg,
					},
					Allocations: map[string]jtypes.AllocationConfig{
						alloc1: Alloc1Cfg,
						alloc2: Alloc2Cfg,
					},
				},
			},
			modifiedConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						node1: Node1Cfg,
						node2: Node2Cfg,
					},
					Allocations: map[string]jtypes.AllocationConfig{
						alloc1: Alloc1Cfg,
						alloc2: Alloc2Cfg,
					},
				},
			},
			expected: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes:       map[string]jtypes.NodeConfig{},
					Allocations: map[string]jtypes.AllocationConfig{},
				},
			},
			expectedErr: false,
		},
		{
			name: "remove node with allocation",
			oldConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						node1: Node1Cfg,
						node2: Node2Cfg,
						node3: Node3Cfg,
					},
					Allocations: map[string]jtypes.AllocationConfig{
						alloc1: Alloc1Cfg,
						alloc2: Alloc2Cfg,
						alloc3: Alloc3Cfg,
						alloc4: Alloc4Cfg,
					},
				},
			},
			modifiedConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						node1: Node1Cfg,
						node2: Node2Cfg,
					},
					Allocations: map[string]jtypes.AllocationConfig{
						alloc1: Alloc1Cfg,
						alloc2: Alloc2Cfg,
					},
				},
			},
			expected: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						node3: Node3Cfg,
					},
					Allocations: map[string]jtypes.AllocationConfig{
						alloc3: Alloc3Cfg,
						alloc4: Alloc4Cfg,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := newConfigForRemovedNodes(tt.oldConfig, tt.modifiedConfig)

			if tt.expectedErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Check nodes
			require.Equal(t, len(tt.expected.V1.Nodes), len(result.V1.Nodes))
			for nodeName, expectedNode := range tt.expected.V1.Nodes {
				actualNode, exists := result.V1.Nodes[nodeName]
				require.True(t, exists, "Node %s should exist", nodeName)
				require.ElementsMatch(t, expectedNode.Allocations, actualNode.Allocations)
			}

			// Check allocations
			require.Equal(t, len(tt.expected.V1.Allocations), len(result.V1.Allocations))
			for allocName, expectedAlloc := range tt.expected.V1.Allocations {
				actualAlloc, exists := result.V1.Allocations[allocName]
				require.True(t, exists, "Allocation %s should exist", allocName)
				require.Equal(t, expectedAlloc.Type, actualAlloc.Type)
				require.Equal(t, expectedAlloc.Executor, actualAlloc.Executor)
			}

			// Check scripts and keys if they exist in expected
			if tt.expected.V1.Scripts != nil {
				require.Equal(t, tt.expected.V1.Scripts, result.V1.Scripts)
			}
			if tt.expected.V1.Keys != nil {
				require.Equal(t, tt.expected.V1.Keys, result.V1.Keys)
			}

			// Check edges
			require.Equal(t, len(tt.expected.V1.Edges), len(result.V1.Edges))

			// Check supervisor if it exists in expected
			if tt.expected.V1.Supervisor.Strategy != "" {
				require.Equal(t, tt.expected.V1.Supervisor.Strategy, result.V1.Supervisor.Strategy)
			}
		})
	}
}

func TestNewConfigForDeploymentUpdate(t *testing.T) {
	withAllocs := func(n jtypes.NodeConfig, allocs ...string) jtypes.NodeConfig { n.Allocations = allocs; return n }
	tests := []struct {
		name           string
		oldConfig      jtypes.EnsembleConfig
		modifiedConfig jtypes.EnsembleConfig
		expected       jtypes.EnsembleConfig
		expectedErr    bool
	}{
		{
			name: "basic new nodes",
			oldConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						node1: Node1Cfg,
					},
					Allocations: map[string]jtypes.AllocationConfig{
						alloc1: Alloc1Cfg,
					},
				},
			},
			modifiedConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						node1: Node1Cfg,
						node2: Node2Cfg,
					},
					Allocations: map[string]jtypes.AllocationConfig{
						alloc1: Alloc1Cfg,
						alloc2: Alloc2Cfg,
					},
					Scripts: testScripts,
					Keys:    testKeys,
				},
			},
			expected: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						node1: withAllocs(Node1Cfg),
						node2: Node2Cfg,
					},
					Allocations: map[string]jtypes.AllocationConfig{
						alloc2: Alloc2Cfg,
					},
					Scripts: testScripts,
					Keys:    testKeys,
				},
			},
		},
		{
			name: "multiple new nodes with allocations",
			oldConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						node1: Node1Cfg,
					},
					Allocations: map[string]jtypes.AllocationConfig{
						alloc1: Alloc1Cfg,
					},
				},
			},
			modifiedConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						node1: Node1Cfg,
						node2: Node2Cfg,
						node3: Node3Cfg,
					},
					Allocations: map[string]jtypes.AllocationConfig{
						alloc1: Alloc1Cfg,
						alloc2: Alloc2Cfg,
						alloc3: Alloc3Cfg,
						alloc4: Alloc4Cfg,
					},
				},
			},
			expected: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						node1: withAllocs(Node1Cfg),
						node2: Node2Cfg,
						node3: Node3Cfg,
					},
					Allocations: map[string]jtypes.AllocationConfig{
						alloc2: Alloc2Cfg,
						alloc3: Alloc3Cfg,
						alloc4: Alloc4Cfg,
					},
				},
			},
		},
		{
			name: "missing allocation",
			oldConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						node1: Node1Cfg,
					},
					Allocations: map[string]jtypes.AllocationConfig{
						alloc1: Alloc1Cfg,
					},
				},
			},
			modifiedConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						node1: Node1Cfg,
						node2: withAllocs(Node2Cfg, "missing"),
					},
					Allocations: map[string]jtypes.AllocationConfig{
						alloc1: Alloc1Cfg,
					},
				},
			},
			expectedErr: true,
		},
		{
			name: "no new nodes",
			oldConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						node1: Node1Cfg,
						node2: Node2Cfg,
					},
					Allocations: map[string]jtypes.AllocationConfig{
						alloc1: Alloc1Cfg,
						alloc2: Alloc2Cfg,
					},
				},
			},
			modifiedConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						node1: Node1Cfg,
						node2: Node2Cfg,
					},
					Allocations: map[string]jtypes.AllocationConfig{
						alloc1: Alloc1Cfg,
						alloc2: Alloc2Cfg,
					},
				},
			},
			expected: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						node1: withAllocs(Node1Cfg),
						node2: withAllocs(Node2Cfg),
					},
					Allocations: map[string]jtypes.AllocationConfig{},
				},
			},
		},
		{
			name: "add node with existing allocation",
			oldConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						node1: Node1Cfg,
					},
					Allocations: map[string]jtypes.AllocationConfig{
						alloc1: Alloc1Cfg,
						alloc2: Alloc2Cfg, // Already defined but not used
					},
				},
			},
			modifiedConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						node1: Node1Cfg,
						node2: Node2Cfg, // Uses alloc2
					},
					Allocations: map[string]jtypes.AllocationConfig{
						alloc1: Alloc1Cfg,
						alloc2: Alloc2Cfg,
					},
				},
			},
			expected: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						node1: withAllocs(Node1Cfg),
						node2: Node2Cfg,
					},
					Allocations: map[string]jtypes.AllocationConfig{
						alloc2: Alloc2Cfg,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := newConfigForDeploymentUpdate(tt.oldConfig, tt.modifiedConfig, map[string]string{})

			if tt.expectedErr == true {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Check nodes
			require.Equal(t, len(tt.expected.V1.Nodes), len(result.V1.Nodes))
			for nodeName, expectedNode := range tt.expected.V1.Nodes {
				actualNode, exists := result.V1.Nodes[nodeName]
				require.True(t, exists, "Node %s should exist", nodeName)
				require.ElementsMatch(t, expectedNode.Allocations, actualNode.Allocations)
			}

			// Check allocations
			require.Equal(t, len(tt.expected.V1.Allocations), len(result.V1.Allocations))
			for allocName, expectedAlloc := range tt.expected.V1.Allocations {
				actualAlloc, exists := result.V1.Allocations[allocName]
				require.True(t, exists, "Allocation %s should exist", allocName)
				require.Equal(t, expectedAlloc.Type, actualAlloc.Type)
				require.Equal(t, expectedAlloc.Executor, actualAlloc.Executor)
			}

			// Check scripts and keys
			require.Equal(t, tt.expected.V1.Scripts, result.V1.Scripts)
			require.Equal(t, tt.expected.V1.Keys, result.V1.Keys)

			// Check edges
			require.Equal(t, len(tt.expected.V1.Edges), len(result.V1.Edges))

			// Check supervisor
			require.Equal(t, tt.expected.V1.Supervisor.Strategy, result.V1.Supervisor.Strategy)
		})
	}
}

func TestValidateEnsembleUpdate(t *testing.T) {
	tests := []struct {
		name           string
		currentConfig  jtypes.EnsembleConfig
		modifiedConfig jtypes.EnsembleConfig
		expectError    bool
		errorContains  string
	}{
		{
			name: "valid update - no changes",
			currentConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes:       map[string]jtypes.NodeConfig{},
					Allocations: map[string]jtypes.AllocationConfig{},
				},
			},
			modifiedConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes:       map[string]jtypes.NodeConfig{},
					Allocations: map[string]jtypes.AllocationConfig{},
				},
			},
			expectError: false,
		},
		{
			name: "invalid - removing supervisor",
			currentConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes:       map[string]jtypes.NodeConfig{},
					Allocations: map[string]jtypes.AllocationConfig{},
					Supervisor:  jtypes.SupervisorConfig{Allocations: []string{"alloc1"}},
				},
			},
			modifiedConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes:       map[string]jtypes.NodeConfig{},
					Allocations: map[string]jtypes.AllocationConfig{},
					Supervisor:  jtypes.SupervisorConfig{},
				},
			},
			expectError:   true,
			errorContains: "removing supervisor is not allowed",
		},
		{
			name: "invalid - changing node peer",
			currentConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						"node1": {Peer: "peer1"},
					},
					Allocations: map[string]jtypes.AllocationConfig{},
				},
			},
			modifiedConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						"node1": {Peer: "peer2"},
					},
					Allocations: map[string]jtypes.AllocationConfig{},
				},
			},
			expectError:   true,
			errorContains: "changing node's peer for node 'node1' is not allowed",
		},
		{
			name: "invalid - changing node location constraints",
			currentConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						"node1": {
							Peer: "peer1",
							Location: jtypes.LocationConstraints{
								Accept: []jtypes.Location{{Continent: "EU"}},
							},
						},
					},
					Allocations: map[string]jtypes.AllocationConfig{},
				},
			},
			modifiedConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						"node1": {
							Peer: "peer1",
							Location: jtypes.LocationConstraints{
								Accept: []jtypes.Location{{Continent: "EU", Country: "DE"}, {Continent: "EU", Country: "FR"}},
							},
						},
					},
					Allocations: map[string]jtypes.AllocationConfig{},
				},
			},
			expectError:   true,
			errorContains: "changing node location for node 'node1' is not allowed",
		},
		{
			name: "unsupported - changing ports for existing allocations",
			currentConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						"node1": {
							Peer:        "peer1",
							Allocations: []string{"webapp"},
							Ports: []jtypes.PortConfig{
								{Public: 8080, Private: 80, Allocation: "webapp"},
							},
						},
					},
					Allocations: map[string]jtypes.AllocationConfig{
						"webapp": {DNSName: "webapp"},
					},
				},
			},
			modifiedConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						"node1": {
							Peer:        "peer1",
							Allocations: []string{"webapp"},
							Ports: []jtypes.PortConfig{
								{Public: 9090, Private: 80, Allocation: "webapp"},
							},
						},
					},
					Allocations: map[string]jtypes.AllocationConfig{
						"webapp": {DNSName: "webapp"},
					},
				},
			},
			expectError:   true,
			errorContains: "changing node's ports for existing allocations on node 'node1' is not supported",
		},
		{
			name: "unsupported - adding edge constraints for existing nodes",
			currentConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						"node1": {Peer: "peer1"},
						"node2": {Peer: "peer2"},
					},
					Allocations: map[string]jtypes.AllocationConfig{},
					Edges:       []jtypes.EdgeConstraint{},
				},
			},
			modifiedConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						"node1": {Peer: "peer1"},
						"node2": {Peer: "peer2"},
					},
					Allocations: map[string]jtypes.AllocationConfig{},
					Edges: []jtypes.EdgeConstraint{
						{S: "node1", T: "node2", RTT: 100},
					},
				},
			},
			expectError:   true,
			errorContains: "adding edge constraints for already deployed nodes 'node1' and 'node2' is not supported",
		},
		{
			name: "unsupported - changing supervisor strategy",
			currentConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes:       map[string]jtypes.NodeConfig{},
					Allocations: map[string]jtypes.AllocationConfig{},
					Supervisor:  jtypes.SupervisorConfig{Strategy: jtypes.StrategyOneForOne},
				},
			},
			modifiedConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes:       map[string]jtypes.NodeConfig{},
					Allocations: map[string]jtypes.AllocationConfig{},
					Supervisor:  jtypes.SupervisorConfig{Strategy: jtypes.StrategyAllForOne},
				},
			},
			expectError:   true,
			errorContains: "changing supervisor strategy is not supported",
		},
		{
			name: "valid - adding new allocations and nodes",
			currentConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						"node1": {Peer: "peer1"},
					},
					Allocations: map[string]jtypes.AllocationConfig{
						"alloc1": {DNSName: "alloc1"},
					},
				},
			},
			modifiedConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						"node1": {Peer: "peer1"},
						"node2": {Peer: "peer2"},
					},
					Allocations: map[string]jtypes.AllocationConfig{
						"alloc1": {DNSName: "alloc1"},
						"alloc2": {DNSName: "alloc2"},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid - ports for new allocations on existing nodes",
			currentConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						"node1": {
							Peer:        "peer1",
							Allocations: []string{"webapp"},
							Ports: []jtypes.PortConfig{
								{Public: 8080, Private: 80, Allocation: "webapp"},
							},
						},
					},
					Allocations: map[string]jtypes.AllocationConfig{
						"webapp": {DNSName: "webapp"},
					},
				},
			},
			modifiedConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						"node1": {
							Peer:        "peer1",
							Allocations: []string{"webapp", "api"},
							Ports: []jtypes.PortConfig{
								{Public: 8080, Private: 80, Allocation: "webapp"},
								{Public: 3000, Private: 3000, Allocation: "api"},
							},
						},
					},
					Allocations: map[string]jtypes.AllocationConfig{
						"webapp": {DNSName: "webapp"},
						"api":    {DNSName: "api"},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid - removing ports for removed allocations",
			currentConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						"node1": {
							Peer:        "peer1",
							Allocations: []string{"webapp", "db"},
							Ports: []jtypes.PortConfig{
								{Public: 8080, Private: 80, Allocation: "webapp"},
							},
						},
					},
					Allocations: map[string]jtypes.AllocationConfig{
						"webapp": {DNSName: "webapp"},
						"db":     {DNSName: "db"},
					},
				},
			},
			modifiedConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						"node1": {
							Peer:        "peer1",
							Allocations: []string{"db"},
							Ports:       []jtypes.PortConfig{},
						},
					},
					Allocations: map[string]jtypes.AllocationConfig{
						"db": {DNSName: "db"},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid - edge constraints with new nodes",
			currentConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						"node1": {Peer: "peer1"},
					},
					Allocations: map[string]jtypes.AllocationConfig{},
					Edges:       []jtypes.EdgeConstraint{},
				},
			},
			modifiedConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						"node1": {Peer: "peer1"},
						"node2": {Peer: "peer2"},
					},
					Allocations: map[string]jtypes.AllocationConfig{},
					Edges: []jtypes.EdgeConstraint{
						{S: "node1", T: "node2", RTT: 100},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid - removing allocations and nodes",
			currentConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						"node1": {Peer: "peer1", Allocations: []string{"webapp"}},
						"node2": {Peer: "peer2", Allocations: []string{"db"}},
					},
					Allocations: map[string]jtypes.AllocationConfig{
						"webapp": {DNSName: "webapp"},
						"db":     {DNSName: "database"},
					},
				},
			},
			modifiedConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						"node1": {Peer: "peer1", Allocations: []string{"webapp"}},
					},
					Allocations: map[string]jtypes.AllocationConfig{
						"webapp": {DNSName: "webapp"},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid - keeping existing edge constraints",
			currentConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						"node1": {Peer: "peer1"},
						"node2": {Peer: "peer2"},
					},
					Allocations: map[string]jtypes.AllocationConfig{},
					Edges: []jtypes.EdgeConstraint{
						{S: "node1", T: "node2", RTT: 100},
					},
				},
			},
			modifiedConfig: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						"node1": {Peer: "peer1"},
						"node2": {Peer: "peer2"},
						"node3": {Peer: "peer3"},
					},
					Allocations: map[string]jtypes.AllocationConfig{},
					Edges: []jtypes.EdgeConstraint{
						{S: "node1", T: "node2", RTT: 100},
						{S: "node1", T: "node3", RTT: 50},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEnsembleUpdate(tt.currentConfig, tt.modifiedConfig)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateLocationConstraintsUpdate(t *testing.T) {
	locEU := jtypes.Location{Continent: "EU"}
	locDE := jtypes.Location{Continent: "EU", Country: "DE"}
	locFR := jtypes.Location{Continent: "EU", Country: "FR"}

	tests := []struct {
		name    string
		current jtypes.LocationConstraints
		new     jtypes.LocationConstraints
		want    bool
	}{
		{
			name:    "valid: no changes (empty)",
			current: jtypes.LocationConstraints{Accept: []jtypes.Location{}},
			new:     jtypes.LocationConstraints{Accept: []jtypes.Location{}},
			want:    true,
		},
		{
			name:    "valid: no changes (same accept list)",
			current: jtypes.LocationConstraints{Accept: []jtypes.Location{locDE}},
			new:     jtypes.LocationConstraints{Accept: []jtypes.Location{locDE}},
			want:    true,
		},
		{
			name:    "valid: broaden accept list (add new constraint)",
			current: jtypes.LocationConstraints{Accept: []jtypes.Location{locDE}},
			new:     jtypes.LocationConstraints{Accept: []jtypes.Location{locDE, locFR}},
			want:    true,
		},
		{
			name:    "valid: broaden accept list (replace with broader constraint)",
			current: jtypes.LocationConstraints{Accept: []jtypes.Location{locDE, locFR}},
			new:     jtypes.LocationConstraints{Accept: []jtypes.Location{locEU}},
			want:    true,
		},
		{
			name:    "valid: narrow reject list (remove constraint)",
			current: jtypes.LocationConstraints{Reject: []jtypes.Location{locDE, locFR}},
			new:     jtypes.LocationConstraints{Reject: []jtypes.Location{locFR}},
			want:    true,
		},
		{
			name:    "valid: narrow reject list (replace with narrower constraint)",
			current: jtypes.LocationConstraints{Reject: []jtypes.Location{locEU}},
			new:     jtypes.LocationConstraints{Reject: []jtypes.Location{locDE, locFR}},
			want:    true,
		},
		{
			name:    "valid: clear reject list",
			current: jtypes.LocationConstraints{Accept: []jtypes.Location{locDE}, Reject: []jtypes.Location{locFR}},
			new:     jtypes.LocationConstraints{Accept: []jtypes.Location{locDE}, Reject: []jtypes.Location{}},
			want:    true,
		},
		{
			name:    "valid: clear both accept and reject list",
			current: jtypes.LocationConstraints{Accept: []jtypes.Location{locDE}, Reject: []jtypes.Location{locFR}},
			new:     jtypes.LocationConstraints{Accept: []jtypes.Location{}, Reject: []jtypes.Location{}},
			want:    true,
		},
		{
			name:    "invalid: narrow accept list (remove constraint)",
			current: jtypes.LocationConstraints{Accept: []jtypes.Location{locDE, locFR}},
			new:     jtypes.LocationConstraints{Accept: []jtypes.Location{locDE}},
			want:    false,
		},
		{
			name:    "invalid: narrow accept list (replace with narrower constraint)",
			current: jtypes.LocationConstraints{Accept: []jtypes.Location{locEU}},
			new:     jtypes.LocationConstraints{Accept: []jtypes.Location{locDE}},
			want:    false,
		},
		{
			name:    "invalid: broaden reject list (add constraint)",
			current: jtypes.LocationConstraints{Reject: []jtypes.Location{locDE}},
			new:     jtypes.LocationConstraints{Reject: []jtypes.Location{locDE, locFR}},
			want:    false,
		},
		{
			name:    "invalid: broaden reject list (replace with broader constraint)",
			current: jtypes.LocationConstraints{Reject: []jtypes.Location{locDE}},
			new:     jtypes.LocationConstraints{Reject: []jtypes.Location{locEU}},
			want:    false,
		},
		{
			name:    "invalid: switch from accept list to reject list",
			current: jtypes.LocationConstraints{Accept: []jtypes.Location{locDE}, Reject: []jtypes.Location{locFR}},
			new:     jtypes.LocationConstraints{Accept: []jtypes.Location{}, Reject: []jtypes.Location{locDE}},
			want:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ok := validateLocationConstraintsUpdate(tc.current, tc.new)
			require.Equal(t, tc.want, ok)
		})
	}
}
