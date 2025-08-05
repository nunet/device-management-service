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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	allocName       = "a"
	origDNSName     = "orig"
	modifiedDNSName = "mutated"
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

		contract := ContractManifest{
			ID:   "contract1",
			DID:  "did:example:1",
			Host: "did:host:1",
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
			Contracts: map[string]ContractManifest{"contract1": contract},
		}

		clone := mf.Clone()

		require.Equal(t, mf.ID, clone.ID)
		require.Equal(t, mf.Orchestrator, clone.Orchestrator)
		require.Equal(t, mf.Allocations, clone.Allocations)
		require.Equal(t, mf.Nodes, clone.Nodes)
		require.Equal(t, mf.Contracts, clone.Contracts)

		// Ensure contracts are cloned
		for key, contract := range mf.Contracts {
			cloneContract, ok := clone.Contracts[key]
			require.True(t, ok)

			require.Equal(t, contract.ID, cloneContract.ID)
			require.Equal(t, contract.DID, cloneContract.DID)
			require.Equal(t, contract.Host, cloneContract.Host)
		}

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

	t.Run("must be able to check terminated task", func(t *testing.T) {
		t.Parallel()

		mf := EnsembleManifest{
			Allocations: map[string]AllocationManifest{
				"alloc1": {
					Type:   AllocationTypeTask,
					Status: AllocationRunning,
				},
				"alloc2": {
					Type:   AllocationTypeTask,
					Status: AllocationStopped,
				},
				"alloc3": {
					Type:   AllocationTypeService,
					Status: AllocationRunning,
				},
			},
		}

		// non-existent allocation should return false
		require.False(t, mf.IsTerminatedTask("non-existent"))

		// running allocation should return false
		require.False(t, mf.IsTerminatedTask("alloc1"))

		// stopped allocation should return true
		require.True(t, mf.IsTerminatedTask("alloc2"))

		// allocation not of type task should return false
		require.False(t, mf.IsTerminatedTask("alloc3"))
	})

	t.Run("must be able to get allocation", func(t *testing.T) {
		t.Parallel()
		mf := EnsembleManifest{
			Allocations: map[string]AllocationManifest{
				allocName: {
					Type:   AllocationTypeTask,
					Status: AllocationRunning,
				},
			},
		}

		// non-existent allocation should return false
		alloc, ok := mf.Allocation("non-existent")
		require.False(t, ok)
		require.Equal(t, AllocationManifest{}, alloc)

		// existing allocation should return true
		alloc, ok = mf.Allocation(allocName)
		require.True(t, ok)
		require.Equal(t, AllocationTypeTask, alloc.Type)
		require.Equal(t, AllocationRunning, alloc.Status)
	})

	t.Run("must be able to get node", func(t *testing.T) {
		t.Parallel()

		mf := EnsembleManifest{
			Nodes: map[string]NodeManifest{
				"node1": {
					ID:   "node1",
					Peer: "peer1",
				},
			},
		}

		// non-existent node should return false
		node, ok := mf.Node("non-existent")
		require.False(t, ok)
		require.Equal(t, NodeManifest{}, node)

		// existing node should return true
		node, ok = mf.Node("node1")
		require.True(t, ok)
		require.Equal(t, "node1", node.ID)
		require.Equal(t, "peer1", node.Peer)
	})
}

func TestUpdateAllocation(t *testing.T) {
	t.Parallel()

	t.Run("update existing allocation", func(t *testing.T) {
		t.Parallel()
		manifest := EnsembleManifest{
			Allocations: map[string]AllocationManifest{
				allocName: {
					DNSName: origDNSName,
					Status:  AllocationRunning,
				},
			},
		}

		err := manifest.UpdateAllocation(allocName, func(alloc *AllocationManifest) {
			alloc.DNSName = modifiedDNSName
			alloc.Status = AllocationStopped
		})

		assert.NoError(t, err)
		assert.Equal(t, modifiedDNSName, manifest.Allocations[allocName].DNSName)
		assert.Equal(t, AllocationStopped, manifest.Allocations[allocName].Status)
	})

	t.Run("update non-existent allocation", func(t *testing.T) {
		t.Parallel()
		manifest := EnsembleManifest{
			Allocations: map[string]AllocationManifest{},
		}

		err := manifest.UpdateAllocation("non-existent", func(alloc *AllocationManifest) {
			alloc.DNSName = modifiedDNSName
		})

		assert.Error(t, err)
		assert.Equal(t, ErrAllocationNotFound, err)
	})

	t.Run("update does not affect other allocations", func(t *testing.T) {
		t.Parallel()
		const otherAllocName = "other"
		const otherDNSName = "other-dns"

		manifest := EnsembleManifest{
			Allocations: map[string]AllocationManifest{
				allocName:      {DNSName: origDNSName},
				otherAllocName: {DNSName: otherDNSName},
			},
		}

		err := manifest.UpdateAllocation(allocName, func(alloc *AllocationManifest) {
			alloc.DNSName = modifiedDNSName
		})

		assert.NoError(t, err)
		assert.Equal(t, modifiedDNSName, manifest.Allocations[allocName].DNSName)
		assert.Equal(t, otherDNSName, manifest.Allocations[otherAllocName].DNSName)
	})
}
