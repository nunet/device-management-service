// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package types

// NodeIDGenerator generates unique node IDs
type NodeIDGenerator interface {
	// GenerateNodeID generates a node ID from a base name
	GenerateNodeID(baseName string) (string, error)

	// GenerateStandbyNodeID generates a standby node ID from a primary node ID and standby index
	GenerateStandbyNodeID(primaryNodeID string, standbyIndex int) (string, error)

	// ValidateNodeID validates that a node ID is properly formatted
	ValidateNodeID(nodeID string) error

	// ParseNodeID extracts components from a node ID (if needed)
	ParseNodeID(nodeID string) (isStandby bool, primaryNodeID string, standbyIndex int, err error)
}

// AllocationIDGenerator generates unique allocation IDs and keys
type AllocationIDGenerator interface {
	// GenerateManifestKey generates the key for manifest.Allocations map
	// Format: nodeID.allocName (generator doesn't care if nodeID is standby or not)
	GenerateManifestKey(nodeID, allocName string) (string, error)

	// GenerateFullAllocationID generates the full allocation ID for actor handles
	// Format: ensembleID_nodeID.allocName (generator doesn't care if nodeID is standby or not)
	GenerateFullAllocationID(ensembleID, nodeID, allocName string) (string, error)

	// ValidateManifestKey validates a manifest key format
	ValidateManifestKey(manifestKey string) error

	// ValidateFullAllocationID validates a full allocation ID format
	ValidateFullAllocationID(allocID string) error
}

// GeneratorValidator validates that a generator won't cause conflicts
type GeneratorValidator interface {
	// ValidateAllocationIDGenerator validates that the generator won't cause conflicts
	// when given the same allocation name but different node types (primary vs standby)
	ValidateAllocationIDGenerator(generator AllocationIDGenerator) error

	// ValidateNodeIDGenerator validates that the generator won't cause conflicts
	ValidateNodeIDGenerator(generator NodeIDGenerator) error
}
