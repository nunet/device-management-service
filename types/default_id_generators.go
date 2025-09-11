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
	"strings"
)

// DefaultAllocationIDGenerator is the default implementation of AllocationIDGenerator
type DefaultAllocationIDGenerator struct{}

// NewDefaultAllocationIDGenerator creates a new DefaultAllocationIDGenerator
func NewDefaultAllocationIDGenerator() *DefaultAllocationIDGenerator {
	return &DefaultAllocationIDGenerator{}
}

// GenerateManifestKey generates the key for manifest.Allocations map
// Format: nodeID.allocName (generator doesn't care if nodeID is standby or not)
func (g *DefaultAllocationIDGenerator) GenerateManifestKey(nodeID, allocName string) (string, error) {
	if nodeID == "" || allocName == "" {
		return "", fmt.Errorf("nodeID and allocName cannot be empty")
	}

	// Simple format: nodeID.allocName (generator doesn't care if nodeID is standby or not)
	return fmt.Sprintf("%s.%s", nodeID, allocName), nil
}

// GenerateFullAllocationID generates the full allocation ID for actor handles
// Format: ensembleID_nodeID.allocName (generator doesn't care if nodeID is standby or not)
func (g *DefaultAllocationIDGenerator) GenerateFullAllocationID(ensembleID, nodeID, allocName string) (string, error) {
	if ensembleID == "" || nodeID == "" || allocName == "" {
		return "", fmt.Errorf("ensembleID, nodeID, and allocName cannot be empty")
	}

	// Simple format: ensembleID_nodeID.allocName (generator doesn't care if nodeID is standby or not)
	return fmt.Sprintf("%s_%s.%s", ensembleID, nodeID, allocName), nil
}

// ValidateManifestKey validates a manifest key format
func (g *DefaultAllocationIDGenerator) ValidateManifestKey(manifestKey string) error {
	if manifestKey == "" {
		return fmt.Errorf("manifest key cannot be empty")
	}

	// Validate format: nodeID.allocName
	parts := strings.Split(manifestKey, ".")
	if len(parts) != 2 {
		return fmt.Errorf("invalid manifest key format: %s (expected nodeID.allocName)", manifestKey)
	}

	nodeID, allocName := parts[0], parts[1]
	if nodeID == "" || allocName == "" {
		return fmt.Errorf("invalid manifest key format: %s (nodeID and allocName cannot be empty)", manifestKey)
	}

	return nil
}

// ValidateFullAllocationID validates a full allocation ID format
func (g *DefaultAllocationIDGenerator) ValidateFullAllocationID(allocID string) error {
	if allocID == "" {
		return fmt.Errorf("allocation ID cannot be empty")
	}

	// Validate format: ensembleID_nodeID.allocName
	parts := strings.SplitN(allocID, "_", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid allocation ID format: %s (expected ensembleID_nodeID.allocName)", allocID)
	}

	ensembleID, rest := parts[0], parts[1]
	if ensembleID == "" {
		return fmt.Errorf("invalid allocation ID format: %s (ensemble ID cannot be empty)", allocID)
	}

	// Validate the rest follows manifest key format
	if err := g.ValidateManifestKey(rest); err != nil {
		return fmt.Errorf("invalid allocation ID format: %s (%v)", allocID, err)
	}

	return nil
}

// DefaultNodeIDGenerator is the default implementation of NodeIDGenerator
type DefaultNodeIDGenerator struct{}

// NewDefaultNodeIDGenerator creates a new DefaultNodeIDGenerator
func NewDefaultNodeIDGenerator() *DefaultNodeIDGenerator {
	return &DefaultNodeIDGenerator{}
}

// GenerateNodeID generates a node ID from a base name
func (g *DefaultNodeIDGenerator) GenerateNodeID(baseName string) (string, error) {
	if baseName == "" {
		return "", fmt.Errorf("base name cannot be empty")
	}

	return baseName, nil
}

// GenerateStandbyNodeID generates a standby node ID from a primary node ID and standby index
func (g *DefaultNodeIDGenerator) GenerateStandbyNodeID(primaryNodeID string, standbyIndex int) (string, error) {
	if primaryNodeID == "" {
		return "", fmt.Errorf("primary node ID cannot be empty")
	}
	if standbyIndex < 1 {
		return "", fmt.Errorf("standby index must be >= 1, got %d", standbyIndex)
	}

	return fmt.Sprintf("%s-standby-%d", primaryNodeID, standbyIndex), nil
}

// ValidateNodeID validates that a node ID is properly formatted
func (g *DefaultNodeIDGenerator) ValidateNodeID(nodeID string) error {
	if nodeID == "" {
		return fmt.Errorf("node ID cannot be empty")
	}

	return nil
}

// ParseNodeID extracts components from a node ID (if needed)
func (g *DefaultNodeIDGenerator) ParseNodeID(nodeID string) (bool, string, int, error) {
	isStandby, primaryNodeID, standbyIndex := ParseNodeName(nodeID)
	return isStandby, primaryNodeID, standbyIndex, nil
}
