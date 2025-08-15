// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package orchestrator

import (
	"fmt"
	"slices"

	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/utils"
)

func newConfigForDeploymentUpdate(
	oldCfg, modifiedCfg jtypes.EnsembleConfig,
	existingNodes map[string]string,
) (jtypes.EnsembleConfig, error) {
	nodes := make(map[string]jtypes.NodeConfig)
	allocations := make(map[string]jtypes.AllocationConfig)
	excludePeers := modifiedCfg.V1.ExcludePeers

	for n, ncfg := range modifiedCfg.Nodes() {
		newAllocations, err := identifyNewAllocations(oldCfg, modifiedCfg, n)
		if err != nil {
			return jtypes.EnsembleConfig{}, err
		}

		if peer, ok := existingNodes[n]; ok {
			ncfg.Peer = peer
			if len(newAllocations) == 0 {
				excludePeers = append(excludePeers, peer)
			}
		}

		for alloc, acfg := range newAllocations {
			allocations[alloc] = acfg
		}
		ncfg.Allocations = utils.MapKeysToSlice(newAllocations)

		var ports []jtypes.PortConfig
		for _, port := range ncfg.Ports {
			if slices.Contains(ncfg.Allocations, port.Allocation) {
				ports = append(ports, port)
			}
		}
		ncfg.Ports = ports

		nodes[n] = ncfg
	}

	return jtypes.EnsembleConfig{
		V1: &jtypes.EnsembleConfigV1{
			EscalationStrategy: modifiedCfg.V1.EscalationStrategy,
			Allocations:        allocations,
			Nodes:              nodes,
			Scripts:            modifiedCfg.V1.Scripts,
			Keys:               modifiedCfg.V1.Keys,
			Edges:              modifiedCfg.EdgeConstraints(),
			Supervisor:         modifiedCfg.V1.Supervisor,
			ExcludePeers:       excludePeers,
			Subnet:             modifiedCfg.Subnet(),
			Metadata:           modifiedCfg.V1.Metadata,
		},
	}, nil
}

func identifyNewAllocations(
	currentConfig, modifiedConfig jtypes.EnsembleConfig,
	node string,
) (map[string]jtypes.AllocationConfig, error) {
	allocations := make(map[string]jtypes.AllocationConfig)
	ncfg, ok := modifiedConfig.Node(node)
	if !ok {
		return allocations, nil
	}
	var currentAllocations []string
	if cfg, ok := currentConfig.Node(node); ok {
		currentAllocations = cfg.Allocations
	}
	for _, alloc := range ncfg.Allocations {
		if slices.Contains(currentAllocations, alloc) {
			continue
		}
		if allocConfig, ok := modifiedConfig.Allocation(alloc); ok {
			allocations[alloc] = allocConfig
		} else {
			return nil, fmt.Errorf("allocation %s not found", alloc)
		}
	}
	return allocations, nil
}

func identifyRemovedAllocations(
	currentConfig, modifiedConfig jtypes.EnsembleConfig,
	node string,
) map[string]jtypes.AllocationConfig {
	allocations := make(map[string]jtypes.AllocationConfig)
	ncfg, ok := modifiedConfig.Node(node)
	if !ok {
		return allocations
	}
	if cfg, ok := currentConfig.Node(node); ok {
		for _, alloc := range cfg.Allocations {
			if !slices.Contains(ncfg.Allocations, alloc) {
				if allocConfig, ok := currentConfig.Allocation(alloc); ok {
					allocations[alloc] = allocConfig
				}
			}
		}
	}
	return allocations
}

// newConfigForRemovedNodes builds a new ensemble config based on
// a base ensemble config + removed nodes
func newConfigForRemovedNodes(
	oldCfg, modifieCfg jtypes.EnsembleConfig,
) (jtypes.EnsembleConfig, error) {
	removedNodes := identifyRemovedNodes(oldCfg, modifieCfg)

	nodesCfg := jtypes.EnsembleConfig{
		V1: &jtypes.EnsembleConfigV1{
			Allocations: make(map[string]jtypes.AllocationConfig),
			Nodes:       removedNodes,
			Scripts:     modifieCfg.V1.Scripts,
			Keys:        modifieCfg.V1.Keys,
			Edges:       modifieCfg.EdgeConstraints(),
			Supervisor:  modifieCfg.V1.Supervisor,
		},
	}

	// add their allocations
	allocationsNames := make([]string, 0)
	for _, node := range nodesCfg.Nodes() {
		allocationsNames = append(allocationsNames, node.Allocations...)
	}

	allocations := make(map[string]jtypes.AllocationConfig)
	for _, name := range allocationsNames {
		alloc, ok := oldCfg.Allocation(name)
		if !ok {
			return jtypes.EnsembleConfig{},
				fmt.Errorf("new config for nodes: allocation %s not found", name)
		}
		allocations[name] = alloc
	}

	nodesCfg.V1.Allocations = allocations

	return nodesCfg, nil
}

// manifestOnlyForNodes returns a manifest that only contains the given nodes
// and their allocations
func manifestOnlyForNodes(mf jtypes.EnsembleManifest, nodes []string,
) (jtypes.EnsembleManifest, error) {
	newMf := jtypes.EnsembleManifest{
		ID:           mf.ID,
		Orchestrator: mf.Orchestrator,
		Nodes:        make(map[string]jtypes.NodeManifest),
		Allocations:  make(map[string]jtypes.AllocationManifest),
		Subnet:       mf.Subnet,
	}

	for _, name := range nodes {
		nmf, ok := mf.Node(name)
		if !ok {
			return jtypes.EnsembleManifest{},
				fmt.Errorf("manifestOnlyForNodes: node %s not found", name)
		}

		newMf.Nodes[name] = nmf

		for _, allocName := range nmf.Allocations {
			alloc, ok := mf.Allocation(allocName)
			if !ok {
				return jtypes.EnsembleManifest{},
					fmt.Errorf("manifestOnlyForNodes: allocation %s not found", allocName)
			}

			newMf.Allocations[allocName] = alloc
		}
	}

	return newMf, nil
}

func identifyRemovedNodes(
	currentConfig, modifiedConfig jtypes.EnsembleConfig,
) map[string]jtypes.NodeConfig {
	nodes := make(map[string]jtypes.NodeConfig)

	for name, node := range currentConfig.Nodes() {
		if _, exists := modifiedConfig.Node(name); !exists {
			nodes[name] = node
		}
	}

	return nodes
}

// validateEnsembleUpdate checks if a given ensemble modification is valid
//
// Invalid modifications:
// - Removing supervisor
// - Changing node location
// - Changing node's peer
//
// Unsupported modifications:
// - Changing node's ports (excepting when adding for new node's allocations)
// - Adding edge constraints for already deployed nodes
// - Changing supervisor strategy
func validateEnsembleUpdate(currentConfig, modifiedConfig jtypes.EnsembleConfig) error {
	// 1. Supervisor must not be removed
	if len(currentConfig.V1.Supervisor.Allocations) > 0 &&
		len(modifiedConfig.V1.Supervisor.Allocations) == 0 {
		return fmt.Errorf("invalid modification: removing supervisor is not allowed")
	}

	// 2. Validate existing nodes
	for name, currNode := range currentConfig.Nodes() {
		modNode, ok := modifiedConfig.Node(name)
		if !ok {
			continue
		}

		if !validateLocationConstraintsUpdate(currNode.Location, modNode.Location) {
			return fmt.Errorf("invalid modification: changing node location for node '%s' is not allowed", name)
		}

		if currNode.Peer != modNode.Peer {
			return fmt.Errorf("invalid modification: changing node's peer for node '%s' is not allowed", name)
		}

		// Track new allocations
		newAllocs := map[string]bool{}
		for _, alloc := range modNode.Allocations {
			if !slices.Contains(currNode.Allocations, alloc) {
				newAllocs[alloc] = true
			}
		}

		// Validate existing port configurations
		for _, currPort := range currNode.Ports {
			if _, isNew := newAllocs[currPort.Allocation]; isNew {
				continue
			}
			found := false
			for _, modPort := range modNode.Ports {
				if modPort.Allocation == currPort.Allocation &&
					modPort.Public == currPort.Public &&
					modPort.Private == currPort.Private {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("unsupported modification: changing node's ports for existing allocations on node '%s' is not supported", name)
			}
		}
	}

	// 3. Edge constraints: no new edges between already deployed nodes
	existing := currentConfig.Nodes()
	for _, edge := range modifiedConfig.V1.Edges {
		if _, sExists := existing[edge.S]; !sExists {
			continue
		}
		if _, tExists := existing[edge.T]; !tExists {
			continue
		}

		// Skip if edge already exists
		alreadyExists := false
		for _, currEdge := range currentConfig.V1.Edges {
			if (currEdge.S == edge.S && currEdge.T == edge.T) ||
				(currEdge.S == edge.T && currEdge.T == edge.S) {
				alreadyExists = true
				break
			}
		}
		if !alreadyExists {
			return fmt.Errorf("unsupported modification: adding edge constraints for already deployed nodes '%s' and '%s' is not supported", edge.S, edge.T)
		}
	}

	// 4. Supervisor strategy must remain unchanged
	if currentConfig.V1.Supervisor.Strategy != modifiedConfig.V1.Supervisor.Strategy {
		return fmt.Errorf("unsupported modification: changing supervisor strategy is not supported")
	}

	return nil
}

// Helper function to check if location constraints updates are valid
func validateLocationConstraintsUpdate(current, newLoc jtypes.LocationConstraints) bool {
	covers := func(super, sub []jtypes.Location) bool {
		for _, loc := range sub {
			if !slices.ContainsFunc(super, func(candidate jtypes.Location) bool {
				return candidate.Equal(loc)
			}) {
				return false
			}
		}
		return true
	}

	hasNewAccept := len(newLoc.Accept) > 0
	hasCurrentAccept := len(current.Accept) > 0
	hasNewReject := len(newLoc.Reject) > 0
	hasCurrentReject := len(current.Reject) > 0

	if hasNewAccept {
		// Accept list can only broaden: newLoc must include all current accepted locations
		if !hasCurrentAccept || !covers(newLoc.Accept, current.Accept) {
			return false
		}
	} else if hasCurrentAccept && hasNewReject {
		// Cannot switch from accept list mode to reject list mode
		return false
	}

	if hasNewReject {
		// Reject list can only shrink: newLoc must be a subset of current reject list
		if !hasCurrentReject || !covers(current.Reject, newLoc.Reject) {
			return false
		}
	}

	return true
}
