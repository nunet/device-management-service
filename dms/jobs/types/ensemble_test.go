// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package jobtypes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testAlloc1 = "alloc1"
	testAlloc2 = "alloc2"
	testDNS1   = "dns1"
	testDNS2   = "dns2"
	testNode1  = "node1"
	testNode2  = "node2"
	testPort1  = 8080
	testPort2  = 9090
)

func TestEnsemble(t *testing.T) {
	t.Parallel()

	t.Run("must be able to validate ensemble", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			ensemble *EnsembleConfig
			wantErr  bool
		}{
			{
				name: "valid ensemble",
				ensemble: &EnsembleConfig{
					V1: &EnsembleConfigV1{
						Allocations: map[string]AllocationConfig{
							"alloc1": {},
						},
						Nodes: map[string]NodeConfig{
							"node1": {},
						},
						Edges: []EdgeConstraint{},
					},
				},
			},
			{
				name:     "invalid ensemble",
				ensemble: &EnsembleConfig{},
				wantErr:  true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				if err := tt.ensemble.Validate(); (err != nil) != tt.wantErr {
					t.Errorf("EnsembleConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})

	t.Run("must be able to get allocations", func(t *testing.T) {
		t.Parallel()

		ensemble := &EnsembleConfig{
			V1: &EnsembleConfigV1{
				Allocations: map[string]AllocationConfig{
					"alloc1": {
						DNSName: "dns1",
					},
				},
			},
		}

		a, ok := ensemble.Allocation("alloc1")
		require.True(t, ok)
		require.Equal(t, ensemble.V1.Allocations["alloc1"].DNSName, a.DNSName)

		_, ok = ensemble.Allocation("alloc2")
		require.False(t, ok)

		allocations := ensemble.Allocations()
		require.Len(t, allocations, 1)
		require.Equal(t, ensemble.V1.Allocations, allocations)
	})

	t.Run("must be able to get nodes", func(t *testing.T) {
		t.Parallel()

		ensemble := &EnsembleConfig{
			V1: &EnsembleConfigV1{
				Nodes: map[string]NodeConfig{
					"node1": {
						Peer: "peer1",
					},
				},
			},
		}

		n, ok := ensemble.Node("node1")
		require.True(t, ok)
		require.Equal(t, ensemble.V1.Nodes["node1"].Peer, n.Peer)

		_, ok = ensemble.Node("node2")
		require.False(t, ok)

		nodes := ensemble.Nodes()
		require.Len(t, nodes, 1)
		require.Equal(t, ensemble.V1.Nodes, nodes)
	})

	t.Run("must be able to get edge constraints", func(t *testing.T) {
		t.Parallel()

		ensemble := &EnsembleConfig{
			V1: &EnsembleConfigV1{
				Edges: []EdgeConstraint{
					{
						T: "t1",
					},
				},
			},
		}

		edges := ensemble.EdgeConstraints()
		require.Len(t, edges, 1)
		require.Equal(t, ensemble.V1.Edges, edges)
	})

	t.Run("must be able to get subnet", func(t *testing.T) {
		t.Parallel()

		ensemble := &EnsembleConfig{
			V1: &EnsembleConfigV1{
				Subnet: SubnetConfig{
					Join: true,
				},
			},
		}

		subnet := ensemble.Subnet()
		require.Equal(t, ensemble.V1.Subnet, subnet)
	})

	t.Run("must be able to add node allocations", func(t *testing.T) {
		t.Parallel()

		ensemble := &EnsembleConfig{
			V1: &EnsembleConfigV1{
				Allocations: map[string]AllocationConfig{
					"alloc1": {
						DNSName: "dns1",
					},
				},
				Nodes: map[string]NodeConfig{
					"node1": {
						Allocations: []string{"alloc1"},
					},
				},
			},
		}

		ensemble.AddNodeAndAllocations(
			"node1",
			NodeConfig{
				Allocations: []string{"alloc2"},
			},
			map[string]AllocationConfig{
				"alloc2": {
					DNSName: "dns2",
				},
			},
		)

		require.Len(t, ensemble.V1.Allocations, 2)
		require.Len(t, ensemble.V1.Nodes, 1)
		require.Len(t, ensemble.V1.Nodes["node1"].Allocations, 1)
		require.Equal(t, "alloc2", ensemble.V1.Nodes["node1"].Allocations[0])
		require.Equal(t, "dns2", ensemble.V1.Allocations["alloc2"].DNSName)
	})

	t.Run("must be able to remove node and it's allocations", func(t *testing.T) {
		t.Parallel()

		ensemble := &EnsembleConfig{
			V1: &EnsembleConfigV1{
				Allocations: map[string]AllocationConfig{
					"alloc1": {
						DNSName: "dns1",
					},
				},
				Nodes: map[string]NodeConfig{
					"node1": {
						Allocations: []string{"alloc1"},
					},
				},
			},
		}

		ensemble.RemoveNodeAndAllocations("node1")

		require.Empty(t, ensemble.V1.Allocations)
		require.Empty(t, ensemble.V1.Nodes)
	})

	t.Run("must be able to clone ensemble", func(t *testing.T) {
		t.Parallel()

		ensemble := &EnsembleConfig{
			V1: &EnsembleConfigV1{
				Allocations: map[string]AllocationConfig{
					"alloc1": {
						DNSName: "dns1",
					},
				},
				Nodes: map[string]NodeConfig{
					"node1": {
						Peer: "peer1",
					},
				},
				Edges: []EdgeConstraint{
					{
						T: "t1",
					},
				},
			},
		}

		clone := ensemble.Clone()
		require.Equal(t, ensemble, &clone)

		// Modify the clone and ensure the original is not modified
		clone.V1.Allocations = make(map[string]AllocationConfig)
		clone.V1.Nodes = make(map[string]NodeConfig)
		clone.V1.Edges = []EdgeConstraint{}

		require.NotEqual(t, ensemble, &clone)
	})

	t.Run("must be able to get allocations for node", func(t *testing.T) {
		t.Parallel()

		ensemble := &EnsembleConfig{
			V1: &EnsembleConfigV1{
				Allocations: map[string]AllocationConfig{
					testAlloc1:    {DNSName: testDNS1},
					testAlloc2:    {DNSName: testDNS2},
					"randomAlloc": {DNSName: "randomDNS"},
				},
				Nodes: map[string]NodeConfig{
					testNode1: {
						Allocations: []string{testAlloc1, testAlloc2},
					},
					"randomNode": {
						Allocations: []string{"randomAlloc"},
					},
				},
			},
		}

		// Test getting allocations for existing node
		allocs := ensemble.AllocationsForNode(testNode1)
		require.Len(t, allocs, 2)
		require.Equal(t, testDNS1, allocs[testAlloc1].DNSName)
		require.Equal(t, testDNS2, allocs[testAlloc2].DNSName)

		// Test getting allocations for non-existent node
		require.Empty(t, ensemble.AllocationsForNode("unknown"))
	})

	t.Run("must be able to get ports for allocation", func(t *testing.T) {
		t.Parallel()

		ensemble := &EnsembleConfig{
			V1: &EnsembleConfigV1{
				Nodes: map[string]NodeConfig{
					testNode1: {
						Ports: []PortConfig{
							{
								Public:     testPort1,
								Private:    80,
								Allocation: testAlloc1,
							},
							{
								Public:     40839,
								Private:    90,
								Allocation: "randomAlloc",
							},
						},
					},
					testNode2: {
						Ports: []PortConfig{
							{
								Public:     testPort2,
								Private:    90,
								Allocation: testAlloc1,
							},
						},
					},
				},
			},
		}

		ports := ensemble.PortsForAllocation(testAlloc1)
		require.Len(t, ports, 2)

		// Verify ports are from both nodes
		publicPorts := []int{ports[0].Public, ports[1].Public}
		require.Contains(t, publicPorts, testPort1)
		require.Contains(t, publicPorts, testPort2)

		// Test getting ports for non-existent allocation
		require.Empty(t, ensemble.PortsForAllocation("unknown"))
	})
}

func TestLocation(t *testing.T) {
	t.Parallel()

	t.Run("must be able to check the equality", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			l        Location
			other    Location
			expected bool
		}{
			{
				name: "same location",
				l: Location{
					Continent: "continent1",
					Country:   "country1",
					City:      "city1",
					ASN:       1,
				},
				other: Location{
					Continent: "continent1",
					Country:   "country1",
					City:      "city1",
					ASN:       1,
				},
				expected: true,
			},
			{
				name: "different region",
				l: Location{
					Continent: "continent1",
					Country:   "country1",
					City:      "city1",
					ASN:       1,
				},
				other: Location{
					Continent: "continent2",
					Country:   "country1",
					City:      "city1",
					ASN:       1,
				},
				expected: false,
			},
			{
				name: "different country",
				l: Location{
					Continent: "continent1",
					Country:   "country1",
					City:      "city1",
					ASN:       1,
				},
				other: Location{
					Continent: "continent1",
					Country:   "country2",
					City:      "city1",
					ASN:       1,
				},
				expected: false,
			},
			{
				name: "different city",
				l: Location{
					Continent: "continent1",
					Country:   "country1",
					City:      "city1",
					ASN:       1,
				},
				other: Location{
					Continent: "continent1",
					Country:   "country1",
					City:      "city2",
					ASN:       1,
				},
				expected: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				require.Equal(t, tt.expected, tt.l.Equal(tt.other))
			})
		}
	})
}
