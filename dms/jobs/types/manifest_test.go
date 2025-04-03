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

	"gitlab.com/nunet/device-management-service/lib/crypto"

	"gitlab.com/nunet/device-management-service/lib/did"

	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/types"
)

func getMockActorHandle(t *testing.T) actor.Handle {
	_, pubK, err := crypto.GenerateKeyPair(crypto.Ed25519)
	require.NoError(t, err)
	id, err := pubK.Raw()
	require.NoError(t, err)
	testDID := did.FromPublicKey(pubK)

	return actor.Handle{
		ID: actor.ID{
			PublicKey: id,
		},
		DID: testDID,
		Address: actor.Address{
			HostID:       "hostID",
			InboxAddress: "inboxAddress",
		},
	}
}

func TestManifest(t *testing.T) {
	t.Parallel()

	t.Run("must be able to clone ensemble manifest", func(t *testing.T) {
		t.Parallel()

		alloc := AllocationManifest{
			ID:       "alloc1",
			NodeID:   "node1",
			Handle:   getMockActorHandle(t),
			DNSName:  "dns1",
			PrivAddr: "priv1",
			Ports:    map[int]int{1: 2},
			Healthcheck: types.HealthCheckManifest{
				Type:     "test-type",
				Exec:     []string{"docker"},
				Endpoint: "test/endpoint",
				Response: types.HealthCheckResponse{
					Type:  "test-response-type",
					Value: "test-value",
				},
				Interval: 5,
			},
		}

		mf := EnsembleManifest{
			ID:           "id1",
			Orchestrator: getMockActorHandle(t),
			Allocations:  map[string]AllocationManifest{"alloc1": alloc},
			Nodes: map[string]NodeManifest{"node1": {
				ID:        "node1",
				Peer:      "peer1",
				Handle:    getMockActorHandle(t),
				PubAddrss: []string{"pub1"},
				Location: Location{
					Continent: "test-continent",
					Country:   "test-country",
					City:      "test-city",
					ASN:       10,
					ISP:       "test-isp",
				},
				Allocations: []string{"alloc1"},
			}},
		}

		clone := mf.Clone()

		require.Equal(t, mf.ID, clone.ID)
		require.Equal(t, mf.Orchestrator, clone.Orchestrator)
		require.Equal(t, mf.Allocations, clone.Allocations)
		require.Equal(t, mf.Nodes, clone.Nodes)

		// Ensure allocations are cloned
		for key, alloc := range mf.Allocations {
			cloneAlloc, ok := clone.Allocations[key]
			require.True(t, ok)

			require.Equal(t, alloc.ID, cloneAlloc.ID)
			require.Equal(t, alloc.NodeID, cloneAlloc.NodeID)
			require.Equal(t, alloc.Handle, cloneAlloc.Handle)
			require.Equal(t, alloc.DNSName, cloneAlloc.DNSName)
			require.Equal(t, alloc.PrivAddr, cloneAlloc.PrivAddr)
			require.Equal(t, alloc.Ports, cloneAlloc.Ports)
			require.Equal(t, alloc.Healthcheck, cloneAlloc.Healthcheck)
		}

		// Ensure nodes are cloned
		for key, node := range mf.Nodes {
			cloneNode, ok := clone.Nodes[key]
			require.True(t, ok)

			require.Equal(t, node.ID, cloneNode.ID)
			require.Equal(t, node.Peer, cloneNode.Peer)
			require.Equal(t, node.Handle, cloneNode.Handle)
			require.Equal(t, node.PubAddrss, cloneNode.PubAddrss)
			require.Equal(t, node.Location, cloneNode.Location)
			require.Equal(t, node.Allocations, cloneNode.Allocations)
		}
	})
}
