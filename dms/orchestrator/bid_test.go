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

	"github.com/stretchr/testify/assert"

	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
)

const (
	testNodeID        = "node1"
	testNonExistentID = "non-existent"
	testPeerID        = "peer1"
	testPeerID2       = "peer2"
	testCountryUS     = "US"
	testCountryCA     = "CA"
	testContinentNA   = "NA"
)

func TestAcceptPeerLocation(t *testing.T) {
	tests := []struct {
		name     string
		cfg      jtypes.EnsembleConfig
		nodeID   string
		peerID   string
		location jtypes.Location
		expected bool
	}{
		{
			name: "node not found",
			cfg: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{},
				},
			},
			nodeID:   testNonExistentID,
			peerID:   testPeerID,
			location: jtypes.Location{},
			expected: false,
		},
		{
			name: "explicit peer match",
			cfg: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						testNodeID: {
							Peer: testPeerID,
						},
					},
				},
			},
			nodeID:   testNodeID,
			peerID:   testPeerID,
			location: jtypes.Location{},
			expected: true,
		},
		{
			name: "explicit peer mismatch",
			cfg: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						testNodeID: {
							Peer: testPeerID,
						},
					},
				},
			},
			nodeID:   testNodeID,
			peerID:   testPeerID2,
			location: jtypes.Location{},
			expected: false,
		},
		{
			name: "accept location match",
			cfg: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						testNodeID: {
							Location: jtypes.LocationConstraints{
								Accept: []jtypes.Location{
									{Continent: testContinentNA, Country: testCountryUS},
								},
							},
						},
					},
				},
			},
			nodeID:   testNodeID,
			peerID:   testPeerID,
			location: jtypes.Location{Continent: testContinentNA, Country: testCountryUS},
			expected: true,
		},
		{
			name: "do not accept when location not on list of acceptable locations",
			cfg: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						testNodeID: {
							Location: jtypes.LocationConstraints{
								Accept: []jtypes.Location{
									{Continent: testContinentNA, Country: testCountryUS},
								},
							},
						},
					},
				},
			},
			nodeID:   testNodeID,
			peerID:   testPeerID,
			location: jtypes.Location{Continent: testContinentNA, Country: testCountryCA},
			expected: false,
		},
		{
			name: "do not accept when location on list of reject locations",
			cfg: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						testNodeID: {
							Location: jtypes.LocationConstraints{
								Reject: []jtypes.Location{
									{Continent: testContinentNA, Country: testCountryUS},
								},
							},
						},
					},
				},
			},
			nodeID:   testNodeID,
			peerID:   testPeerID,
			location: jtypes.Location{Continent: testContinentNA, Country: testCountryUS},
			expected: false,
		},
		{
			name: "accept location not included on the list of reject locations",
			cfg: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						testNodeID: {
							Location: jtypes.LocationConstraints{
								Reject: []jtypes.Location{
									{Continent: testContinentNA, Country: testCountryUS},
								},
							},
						},
					},
				},
			},
			nodeID:   testNodeID,
			peerID:   testPeerID,
			location: jtypes.Location{Continent: testContinentNA, Country: testCountryCA},
			expected: true,
		},
		{
			name: "no constraints",
			cfg: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						testNodeID: {},
					},
				},
			},
			nodeID:   testNodeID,
			peerID:   testPeerID,
			location: jtypes.Location{},
			expected: true,
		},
		{
			name: "multiple accept locations with match",
			cfg: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						testNodeID: {
							Location: jtypes.LocationConstraints{
								Accept: []jtypes.Location{
									{Continent: testContinentNA, Country: testCountryUS},
									{Continent: testContinentNA, Country: testCountryCA},
								},
							},
						},
					},
				},
			},
			nodeID:   testNodeID,
			peerID:   testPeerID,
			location: jtypes.Location{Continent: testContinentNA, Country: testCountryCA},
			expected: true,
		},
		{
			name: "multiple reject locations with match",
			cfg: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						testNodeID: {
							Location: jtypes.LocationConstraints{
								Reject: []jtypes.Location{
									{Continent: testContinentNA, Country: testCountryUS},
									{Continent: testContinentNA, Country: testCountryCA},
								},
							},
						},
					},
				},
			},
			nodeID:   testNodeID,
			peerID:   testPeerID,
			location: jtypes.Location{Continent: testContinentNA, Country: testCountryCA},
			expected: false,
		},
		{
			name: "accept has precedence over reject",
			cfg: jtypes.EnsembleConfig{
				V1: &jtypes.EnsembleConfigV1{
					Nodes: map[string]jtypes.NodeConfig{
						testNodeID: {
							Location: jtypes.LocationConstraints{
								Accept: []jtypes.Location{
									{Continent: testContinentNA, Country: testCountryUS},
								},
								Reject: []jtypes.Location{
									{Continent: testContinentNA, Country: testCountryUS},
								},
							},
						},
					},
				},
			},
			nodeID:   testNodeID,
			peerID:   testPeerID,
			location: jtypes.Location{Continent: testContinentNA, Country: testCountryUS},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := acceptPeerLocation(tt.cfg, tt.nodeID, tt.peerID, tt.location)
			assert.Equal(t, tt.expected, result)
		})
	}
}
