// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package e2e

import (
	"encoding/json"
	"fmt"

	"gitlab.com/nunet/device-management-service/actor"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"
)

type executionInfo struct {
	executor string
	id       string
}

// assertManifestAfterDeployment asserts the manifest after a deployment.
//
// It identifies which compute providers were assigned to each ensemble's node,
// that is one of the basis for the assertions.
//
// Most fields are straight forward and they derive from the ensemble config
// (e.g.: SubnetConfig.Join, Metadata, Allocation.Type and etc)
//
// Allocation.Status: we expect to be running if it's a service but we can't
// assert if it's a task as we it might get completed or not.
//
// (TODO-Maybe) PrivAddr: it's the IP binded to the container
//
// TODO: Handle.ID is buggy right now. I couldn't figure out a way of making
// the assertion work. It maybe how we're deriving IDs from DIDs
//
// Reminder: Handle.DID and Handle.Address.HostID are derived from the root
// cap context which started DMS rather than from the child actors like
// allocation and orchestrator.
func (s *TestSuite) assertManifestAfterDeployment(
	deployer *mockNode,
	providers []*mockNode,
	ensemble jobtypes.EnsembleConfig,
	ensembleID string,
) {
	actualManifest, err := deployer.client.deploymentManifest(
		deployer.dmsContext,
		deployer.password, ensembleID)
	s.Require().NoError(err)

	actualManifestNodes := utils.MapKeysToSlice(actualManifest.Nodes)
	s.Require().True(len(ensemble.Nodes()) == len(actualManifestNodes),
		"Number of nodes in manifest doesn't match number of nodes in config")

	// Map providers by their peer IDs for easy lookup
	providersByPeer := make(map[string]*mockNode)
	for _, provider := range providers {
		providersByPeer[provider.peerID] = provider
	}

	// Build expected manifest
	anchor, err := deployer.capCtx.Trust().GetAnchor(
		deployer.capCtx.DID())
	s.Require().NoError(err)
	deployerHandleID, err := crypto.IDFromPublicKey(anchor.PublicKey())
	s.Require().NoError(err)

	expectedManifest := jobtypes.EnsembleManifest{
		ID:       ensembleID,
		Metadata: ensemble.V1.Metadata,
		Subnet:   ensemble.V1.Subnet,
		Orchestrator: actor.Handle{
			ID:  deployerHandleID,
			DID: deployer.capCtx.DID(),
			Address: actor.Address{
				HostID:       deployer.peerID,
				InboxAddress: ensembleID,
			},
		},
		Allocations: make(map[string]jobtypes.AllocationManifest),
		Nodes:       make(map[string]jobtypes.NodeManifest),
	}

	// Map node name to the provider that is hosting it
	nodeToProvider := make(map[string]*mockNode)

	// First pass: build the nodes
	for nodeName, nodeConfig := range ensemble.V1.Nodes {
		// Find the provider for this node by matching peer ID
		s.Require().NotEmpty(nodeName, "Node name is empty")
		manifestNode, exists := actualManifest.Nodes[nodeName]
		s.Require().True(exists, "Could not find actual node %s in actual manifest", nodeName)

		matchedProvider, ok := providersByPeer[manifestNode.Peer]
		s.Require().True(ok, "Could not find provider for node %s", nodeName)

		s.Require().NotNil(matchedProvider, "Could not find provider for node %s", nodeName)
		nodeToProvider[nodeName] = matchedProvider

		// Parse provider DID
		pubk, err := did.PublicKeyFromDID(matchedProvider.capCtx.DID())
		s.Require().NoError(err)
		providerHandleID, err := crypto.IDFromPublicKey(pubk)
		s.Require().NoError(err)

		// Add this node to the expected manifest
		allocKeys := make([]string, len(nodeConfig.Allocations))
		for i, allocName := range nodeConfig.Allocations {
			allocKeys[i] = fmt.Sprintf("%s.%s", nodeName, allocName)
		}
		expectedManifest.Nodes[nodeName] = jobtypes.NodeManifest{
			ID:          nodeName,
			Peer:        matchedProvider.peerID,
			PubAddrss:   []string{}, // We don't validate this
			Allocations: allocKeys,
			Handle: actor.Handle{
				ID:  providerHandleID,
				DID: matchedProvider.capCtx.DID(),
				Address: actor.Address{
					HostID:       matchedProvider.peerID,
					InboxAddress: "root",
				},
			},
			// Location is not validated
		}
	}

	// Second pass: build the allocations
	for allocName, allocConfig := range ensemble.V1.Allocations {
		// Find which node this allocation is assigned to
		var nodeName string
		ports := make(map[int]int)
		for name, nodeConfig := range ensemble.V1.Nodes {
			if utils.SliceContains(nodeConfig.Allocations, allocName) {
				nodeName = name
				for _, portCfg := range nodeConfig.Ports {
					if portCfg.Allocation == allocName {
						ports[portCfg.Public] = portCfg.Private
					}
				}
			}
		}
		s.Require().NotEmpty(nodeName, "Could not find node for allocation %s", allocName)

		provider := nodeToProvider[nodeName]
		allocID := types.ConstructAllocationID(ensembleID, fmt.Sprintf("%s.%s", nodeName, allocName))

		// Determine status based on allocation type
		var expectedStatus jobtypes.AllocationStatus
		if allocConfig.Type == jobtypes.AllocationTypeService {
			expectedStatus = jobtypes.AllocationRunning
		}

		allocProviderDID, err := did.FromString(provider.dmsDID)
		s.Require().NoError(err)
		pubk, err := did.PublicKeyFromDID(allocProviderDID)
		s.Require().NoError(err)
		allocProviderHandleID, err := crypto.IDFromPublicKey(pubk)
		s.Require().NoError(err)

		expectedManifest.Allocations[fmt.Sprintf("%s.%s", nodeName, allocName)] = jobtypes.AllocationManifest{
			ID:          allocID,
			Type:        allocConfig.Type,
			NodeID:      nodeName,
			DNSName:     allocConfig.DNSName + ".internal",
			Ports:       ports,
			PrivAddr:    "TODO", // TODO: determine how to get private address
			Status:      expectedStatus,
			Healthcheck: allocConfig.HealthCheck,
			Handle: actor.Handle{
				ID:  allocProviderHandleID,
				DID: allocProviderDID,
				Address: actor.Address{
					HostID:       provider.peerID,
					InboxAddress: allocID,
				},
			},
		}
	}

	// Compare the expected and actual manifests
	s.assertManifestsEqual(expectedManifest, actualManifest)
}

// assertNoAllocationsRunning relies on `dms/node/allocations/list`
// and checks if execution is not running too
func (s *TestSuite) assertNoAllocationsRunning(
	node *mockNode,
	executions ...executionInfo,
) {
	allocations, err := node.client.allocationsList(
		node.userContext, node.password)
	s.Require().NoError(err)

	s.Require().Truef(
		len(allocations) == 0, "Expected no allocations to be running but got %d. Allocs: %+vv", len(allocations), allocations)

	// check if executions is running
	s.T().Logf("checking if executions are running %v", executions)
	for _, e := range executions {
		switch e.executor {
		case string(types.ExecutorTypeDocker):
			// IMPROVE: instead, we should be able to filter executions with
			// ensemble ID
			containerRunning, err := isContainerRunning(e.id)
			s.Require().NoError(err)
			s.Require().False(containerRunning, "Expected container to be stopped")
		default:
			s.T().Error("Unknown executor ", e.executor)
		}
	}
}

func (s *TestSuite) assertAllocationsRunning(
	node *mockNode, ensembleID string,
	allocationsNames []string,
) {
	if len(allocationsNames) == 0 {
		return
	}

	allocations, err := node.client.allocationsList(
		node.userContext, node.password)
	s.Require().NoError(err)

	expectedAllocsIDs := make(map[string]bool)
	for _, alloc := range allocationsNames {
		expectedAllocsIDs[alloc] = true
	}
	for _, alloc := range allocations {
		key, err := types.ParseManifestKey(alloc.ID, ensembleID)
		s.Require().NoError(err)
		_, ok := expectedAllocsIDs[key.AllocationName]
		s.Assert().True(ok, "Expected allocation %s to be running", alloc.ID)

		if alloc.Executor == string(types.ExecutorTypeDocker) {
			fmt.Println("alloc.ExecutionID", alloc.ExecutionID)
			containerRunning, err := isContainerRunning(alloc.ExecutionID)
			fmt.Println("containerRunning", containerRunning)
			s.Assert().NoError(err)
			s.Assert().True(containerRunning, "Expected container to be running")
		}
	}
}

func (s *TestSuite) assertResourcesAfterDeployment(
	node *mockNode,
	resourcesAllocated types.Resources,
) {
	// 1. Verify allocated resources match expected resources
	actualAllocated, err := node.client.allocatedResources(s.T(), node.dmsContext, node.password)
	s.Require().NoError(err, "Failed to get allocated resources")
	s.Require().True(actualAllocated.Equal(resourcesAllocated),
		"Allocated resources don't match. Got: %+v, Expected: %+v",
		actualAllocated, resourcesAllocated)

	// 2. Verify free resources = onboarded - allocated
	onboarded, err := node.client.onboardedResources(s.T(), node.dmsContext, node.password)
	s.Require().NoError(err, "Failed to get onboarded resources")

	free, err := node.client.freeResources(s.T(), node.dmsContext, node.password)
	s.Require().NoError(err, "Failed to get free resources")

	expectedFree := onboarded
	err = expectedFree.Subtract(actualAllocated)
	s.Require().NoError(err, "Failed to subtract allocated from onboarded resources")

	s.Require().True(free.Equal(expectedFree),
		"Free resources don't match (onboarded - allocated). Free: %+v, Expected: %+v",
		free, expectedFree)
}

func (s *TestSuite) assertFreeResourcesFull(node *mockNode) {
	// 1. Verify free resources equal onboarded resources
	free, err := node.client.freeResources(s.T(), node.dmsContext, node.password)
	s.Require().NoError(err, "Failed to get free resources")

	onboarded, err := node.client.onboardedResources(s.T(), node.dmsContext, node.password)
	s.Require().NoError(err, "Failed to get onboarded resources")

	s.Require().True(free.Equal(onboarded),
		"Free resources don't match onboarded resources. Free: %+v, Onboarded: %+v",
		free, onboarded)

	// 2. Verify allocated resources are zero
	allocated, err := node.client.allocatedResources(s.T(), node.dmsContext, node.password)
	s.Require().NoError(err, "Failed to get allocated resources")

	// Create empty resources to compare with
	emptyResources := types.Resources{}
	s.Require().True(allocated.Equal(emptyResources),
		"Allocated resources should be zero, but got: %+v", allocated)
}

// assertManifestsEqual compares two ensemble manifests and asserts they are equal
//
// TODO: Handle.ID
//
// Ignored fields:
// - Location
// - Node.PubAddrs
func (s *TestSuite) assertManifestsEqual(
	expected, actual jobtypes.EnsembleManifest,
) {
	// Always log both manifests at the start
	expectedJSON, _ := json.MarshalIndent(expected, "", "  ")
	actualJSON, _ := json.MarshalIndent(actual, "", "  ")
	s.T().Logf("\n==== MANIFESTS BEING COMPARED ====")
	s.T().Logf("\n==== EXPECTED MANIFEST ====\n%s", string(expectedJSON))
	s.T().Logf("\n==== ACTUAL MANIFEST ====\n%s", string(actualJSON))

	// Compare top-level fields
	s.Require().Equal(expected.ID, actual.ID, "Manifest IDs don't match")

	// Metadata
	s.Require().Equal(len(expected.Metadata), len(actual.Metadata), "Metadata map sizes don't match")
	for key, valueA := range expected.Metadata {
		valueB, exists := actual.Metadata[key]
		s.Require().True(exists, "Key %s missing from second manifest metadata", key)
		s.Require().Equal(valueA, valueB, "Metadata values don't match for key %s", key)
	}

	// Orchestrator Handle
	s.Require().True(expected.Orchestrator.DID.Equal(actual.Orchestrator.DID),
		"Orchestrator Handle DID doesn't match. %v != %v",
		expected.Orchestrator.DID.String(), actual.Orchestrator.DID.String())
	s.Require().True(expected.Orchestrator.Address.HostID == actual.Orchestrator.Address.HostID,
		"Orchestrator Handle HostID doesn't match. %v != %v",
		expected.Orchestrator.Address.HostID, actual.Orchestrator.Address.HostID)
	s.Require().True(expected.Orchestrator.Address.InboxAddress == actual.Orchestrator.Address.InboxAddress,
		"Orchestrator Handle InboxAddress doesn't match. %v != %v",
		expected.Orchestrator.Address.InboxAddress, actual.Orchestrator.Address.InboxAddress)

	// Subnet config
	s.Require().Equal(expected.Subnet.Join, actual.Subnet.Join, "Subnet Join flag doesn't match")

	// Use the helper functions to compare nodes and allocations
	s.assertAllocationsManifestsEqual(expected.Allocations, actual.Allocations)
	s.assertNodesManifestsEqual(expected.Nodes, actual.Nodes)
}

// assertNodesManifestsEqual compares node manifests from two ensemble manifests
//
// Ignored fields:
// - Location
// - Node.PubAddrs
func (s *TestSuite) assertNodesManifestsEqual(
	expectedNodes, actualNodes map[string]jobtypes.NodeManifest,
) {
	s.Require().Equal(len(expectedNodes), len(actualNodes), "Number of nodes doesn't match")
	for name, nodeExpected := range expectedNodes {
		nodeActual, exists := actualNodes[name]
		s.Require().True(exists, "Node %s missing from second manifest", name)

		s.Require().Equal(nodeExpected.ID, nodeActual.ID, "Node ID doesn't match for %s", name)
		s.Require().Equal(nodeExpected.Peer, nodeActual.Peer, "Node Peer doesn't match for %s", name)

		// Compare node handle
		// TODO: use handle.Equal when ID equality gets fixed
		s.Require().True(nodeExpected.Handle.DID.Equal(nodeActual.Handle.DID))
		s.Require().True(nodeExpected.Handle.Address.HostID == nodeActual.Handle.Address.HostID)
		s.Require().True(nodeExpected.Handle.Address.InboxAddress == nodeActual.Handle.Address.InboxAddress)

		// Compare allocations list
		s.Require().Equal(len(nodeExpected.Allocations), len(nodeActual.Allocations),
			"Number of allocations doesn't match for node %s", name)
		for i, allocExpected := range nodeExpected.Allocations {
			if i < len(nodeActual.Allocations) {
				s.Require().Equal(allocExpected, nodeActual.Allocations[i],
					"Allocation at index %d doesn't match for node %s", i, name)
			}
		}
	}
}

// assertAllocationsManifestsEqual compares allocation manifests from two ensemble manifests
//
// TODO: PrivAddr (not sure about this one yet)
func (s *TestSuite) assertAllocationsManifestsEqual(
	expectedAllocs, actualAllocs map[string]jobtypes.AllocationManifest,
) {
	s.Require().Equal(len(expectedAllocs), len(actualAllocs), "Number of allocations doesn't match")
	for name, allocExpected := range expectedAllocs {
		allocActual, exists := actualAllocs[name]
		s.Require().True(exists, "Allocation %s missing from second manifest", name)

		s.Require().Equal(allocExpected.ID, allocActual.ID, "Allocation ID doesn't match for %s", name)
		s.Require().Equal(allocExpected.Type, allocActual.Type, "Allocation Type doesn't match for %s", name)
		s.Require().Equal(allocExpected.NodeID, allocActual.NodeID, "Allocation NodeID doesn't match for %s", name)
		s.Require().Equal(allocExpected.DNSName, allocActual.DNSName, "Allocation DNSName doesn't match for %s", name)
		// s.Require().Equal(allocExpected.PrivAddr, allocActual.PrivAddr, "Allocation PrivAddr doesn't match for %s", name)
		s.Require().Equal(allocExpected.Status, allocActual.Status, "Allocation Status doesn't match for %s", name)

		// Compare allocation handle
		s.Require().True(allocExpected.Handle.DID.Equal(allocActual.Handle.DID),
			"Allocation Handle DID doesn't match for allocation %s. %v != %v",
			name, allocExpected.Handle.DID.String(), allocActual.Handle.DID.String(), name)
		s.Require().True(allocExpected.Handle.Address.HostID == allocActual.Handle.Address.HostID,
			"Allocation Handle HostID doesn't match for allocation %s. %v != %v",
			name, allocExpected.Handle.Address.HostID, allocActual.Handle.Address.HostID, name)
		s.Require().True(allocExpected.Handle.Address.InboxAddress == allocActual.Handle.Address.InboxAddress,
			"Allocation Handle InboxAddress doesn't match for allocation %s. %v != %v",
			name, allocExpected.Handle.Address.InboxAddress, allocActual.Handle.Address.InboxAddress, name)

		// Compare ports
		s.Require().Equal(len(allocExpected.Ports), len(allocActual.Ports), "Number of ports doesn't match for allocation %s", name)
		for pubPort, privPortA := range allocExpected.Ports {
			privPortB, exists := allocActual.Ports[pubPort]
			s.Require().True(exists, "Public port %d missing from allocation %s in second manifest", pubPort, name)
			s.Require().Equal(privPortA, privPortB, "Private port mapping doesn't match for public port %d in allocation %s", pubPort, name)
		}

		// Compare healthcheck
		s.Require().Equal(allocExpected.Healthcheck, allocActual.Healthcheck, "Healthcheck doesn't match for allocation %s", name)
	}
}
