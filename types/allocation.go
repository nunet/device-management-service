// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package types

import (
	"fmt"
	"strconv"
	"strings"
)

func ConstructAllocationID(ensembleID, allocName string) string {
	return ensembleID + "_" + allocName
}

func AllocationNameFromID(id string) string {
	return id[strings.LastIndex(id, "_")+1:]
}

func EnsembleIDFromAllocationID(id string) string {
	if strings.Count(id, "_") == 0 {
		return id
	}
	return id[:strings.LastIndex(id, "_")]
}

// AllocationIdentifier represents a structured allocation identifier
// that can be used consistently across the orchestrator
type AllocationIdentifier struct {
	EnsembleID     string
	NodeID         string
	AllocationName string
	IsStandby      bool
	StandbyIndex   int
}

// String returns the full allocation ID for actor handles and cross-ensemble references
// Format: ensembleID_nodeID.allocName or ensembleID_nodeID-standby-N.allocName
func (aid AllocationIdentifier) String() string {
	if aid.IsStandby {
		// Check if NodeID already contains standby suffix
		if strings.Contains(aid.NodeID, "-standby-") {
			// NodeID already has standby suffix, use it as is
			return fmt.Sprintf("%s_%s.%s", aid.EnsembleID, aid.NodeID, aid.AllocationName)
		}
		// NodeID is primary, add standby suffix
		return fmt.Sprintf("%s_%s-standby-%d.%s",
			aid.EnsembleID, aid.NodeID, aid.StandbyIndex, aid.AllocationName)
	}
	return fmt.Sprintf("%s_%s.%s", aid.EnsembleID, aid.NodeID, aid.AllocationName)
}

// ManifestKey returns the key used in manifest.Allocations map
// Format: nodeID.allocName or nodeID-standby-N.allocName
func (aid AllocationIdentifier) ManifestKey() string {
	if aid.IsStandby {
		// Check if NodeID already contains standby suffix
		if strings.Contains(aid.NodeID, "-standby-") {
			// NodeID already has standby suffix, use it as is
			return fmt.Sprintf("%s.%s", aid.NodeID, aid.AllocationName)
		}
		// NodeID is primary, add standby suffix
		return fmt.Sprintf("%s-standby-%d.%s", aid.NodeID, aid.StandbyIndex, aid.AllocationName)
	}
	return fmt.Sprintf("%s.%s", aid.NodeID, aid.AllocationName)
}

// ConfigName returns the base allocation name from configuration
// Format: allocName (same for primary and standby)
func (aid AllocationIdentifier) ConfigName() string {
	return aid.AllocationName
}

// PrimaryNodeID returns the primary node ID (removes standby suffix if present)
func (aid AllocationIdentifier) PrimaryNodeID() string {
	if aid.IsStandby {
		return aid.NodeID
	}
	return aid.NodeID
}

// NewAllocationID creates a new AllocationIdentifier for a primary allocation
func NewAllocationID(ensembleID, nodeID, allocName string) AllocationIdentifier {
	return AllocationIdentifier{
		EnsembleID:     ensembleID,
		NodeID:         nodeID,
		AllocationName: allocName,
		IsStandby:      false,
		StandbyIndex:   0,
	}
}

// NewStandbyAllocationID creates a new AllocationIdentifier for a standby allocation
func NewStandbyAllocationID(ensembleID, primaryNodeID, allocName string, standbyIndex int) AllocationIdentifier {
	return AllocationIdentifier{
		EnsembleID:     ensembleID,
		NodeID:         primaryNodeID, // Keep the primary node ID, we'll handle standby formatting in String() and ManifestKey()
		AllocationName: allocName,
		IsStandby:      true,
		StandbyIndex:   standbyIndex,
	}
}

// ParseAllocationID parses a full allocation ID string into an AllocationIdentifier
// Format: ensembleID_nodeID.allocName or ensembleID_nodeID-standby-N.allocName
func ParseAllocationID(id string) (AllocationIdentifier, error) {
	// Split by first underscore to get ensembleID and the rest
	parts := strings.SplitN(id, "_", 2)
	if len(parts) != 2 {
		return AllocationIdentifier{}, fmt.Errorf("invalid allocation ID format: %s", id)
	}

	ensembleID := parts[0]
	rest := parts[1]

	// Check if it's a standby allocation
	if strings.Contains(rest, "-standby-") {
		// Format: nodeID-standby-N.allocName
		standbyParts := strings.Split(rest, "-standby-")
		if len(standbyParts) != 2 {
			return AllocationIdentifier{}, fmt.Errorf("invalid standby allocation ID format: %s", id)
		}

		nodeID := standbyParts[0]
		indexAndAlloc := strings.Split(standbyParts[1], ".")
		if len(indexAndAlloc) != 2 {
			return AllocationIdentifier{}, fmt.Errorf("invalid standby allocation ID format: %s", id)
		}

		// Validate that the index part is a valid integer
		standbyIndex, err := strconv.Atoi(indexAndAlloc[0])
		if err != nil {
			return AllocationIdentifier{}, fmt.Errorf("invalid standby index in allocation ID: %s", id)
		}

		allocName := indexAndAlloc[1]

		return AllocationIdentifier{
			EnsembleID:     ensembleID,
			NodeID:         fmt.Sprintf("%s-standby-%d", nodeID, standbyIndex),
			AllocationName: allocName,
			IsStandby:      true,
			StandbyIndex:   standbyIndex,
		}, nil
	} else if strings.Contains(rest, "standby") {
		return AllocationIdentifier{}, fmt.Errorf("invalid standby allocation ID format: %s", id)
	}

	// Format: nodeID.allocName
	nodeAllocParts := strings.Split(rest, ".")
	if len(nodeAllocParts) != 2 {
		return AllocationIdentifier{}, fmt.Errorf("invalid allocation ID format: %s", id)
	}

	return AllocationIdentifier{
		EnsembleID:     ensembleID,
		NodeID:         nodeAllocParts[0],
		AllocationName: nodeAllocParts[1],
		IsStandby:      false,
		StandbyIndex:   0,
	}, nil
}

// ParseManifestKey parses a manifest key into an AllocationIdentifier
// Format: nodeID.allocName or nodeID-standby-N.allocName
// Note: This requires the ensembleID to be provided separately
func ParseManifestKey(manifestKey, ensembleID string) (AllocationIdentifier, error) {
	// Check if it's a standby allocation
	if strings.Contains(manifestKey, "-standby-") {
		// Format: nodeID-standby-N.allocName
		standbyParts := strings.Split(manifestKey, "-standby-")
		if len(standbyParts) != 2 {
			return AllocationIdentifier{}, fmt.Errorf("invalid standby manifest key format: %s", manifestKey)
		}

		nodeID := standbyParts[0]
		indexAndAlloc := strings.Split(standbyParts[1], ".")
		if len(indexAndAlloc) != 2 {
			return AllocationIdentifier{}, fmt.Errorf("invalid standby manifest key format: %s", manifestKey)
		}

		standbyIndex, err := strconv.Atoi(indexAndAlloc[0])
		if err != nil {
			return AllocationIdentifier{}, fmt.Errorf("invalid standby index in manifest key: %s", manifestKey)
		}

		allocName := indexAndAlloc[1]

		return AllocationIdentifier{
			EnsembleID:     ensembleID,
			NodeID:         fmt.Sprintf("%s-standby-%d", nodeID, standbyIndex),
			AllocationName: allocName,
			IsStandby:      true,
			StandbyIndex:   standbyIndex,
		}, nil
	}

	// Format: nodeID.allocName
	nodeAllocParts := strings.Split(manifestKey, ".")
	if len(nodeAllocParts) != 2 {
		return AllocationIdentifier{}, fmt.Errorf("invalid manifest key format: %s", manifestKey)
	}

	return AllocationIdentifier{
		EnsembleID:     ensembleID,
		NodeID:         nodeAllocParts[0],
		AllocationName: nodeAllocParts[1],
		IsStandby:      false,
		StandbyIndex:   0,
	}, nil
}

// ParseNodeName parses a node name to determine if it's a standby node
// Returns (isStandby, primaryNodeID, standbyIndex)
func ParseNodeName(nodeName string) (bool, string, int) {
	if strings.Contains(nodeName, "-standby-") {
		parts := strings.Split(nodeName, "-standby-")
		if len(parts) == 2 {
			standbyIndex, err := strconv.Atoi(parts[1])
			if err == nil {
				return true, parts[0], standbyIndex
			}
		}
	}
	return false, nodeName, 0
}
