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
)

// TestNodeIDGenerator for testing with predictable IDs
type TestNodeIDGenerator struct {
	counter int
}

// NewTestNodeIDGenerator creates a new TestNodeIDGenerator
func NewTestNodeIDGenerator() *TestNodeIDGenerator {
	return &TestNodeIDGenerator{counter: 0}
}

// GenerateNodeID generates a node ID from a base name
func (g *TestNodeIDGenerator) GenerateNodeID(baseName string) (string, error) {
	g.counter++
	return fmt.Sprintf("%s-test-%d", baseName, g.counter), nil
}

// GenerateStandbyNodeID generates a standby node ID from a primary node ID and standby index
func (g *TestNodeIDGenerator) GenerateStandbyNodeID(primaryNodeID string, standbyIndex int) (string, error) {
	if primaryNodeID == "" {
		return "", fmt.Errorf("primary node ID cannot be empty")
	}
	if standbyIndex < 1 {
		return "", fmt.Errorf("standby index must be >= 1, got %d", standbyIndex)
	}

	return fmt.Sprintf("%s-standby-%d", primaryNodeID, standbyIndex), nil
}

// ValidateNodeID validates that a node ID is properly formatted
func (g *TestNodeIDGenerator) ValidateNodeID(nodeID string) error {
	// Simple validation for tests
	if nodeID == "" {
		return fmt.Errorf("node ID cannot be empty")
	}
	return nil
}

// ParseNodeID extracts components from a node ID (if needed)
func (g *TestNodeIDGenerator) ParseNodeID(nodeID string) (bool, string, int, error) {
	isStandby, primaryNodeID, standbyIndex := ParseNodeName(nodeID)
	return isStandby, primaryNodeID, standbyIndex, nil
}

// TestAllocationIDGenerator for testing
type TestAllocationIDGenerator struct{}

// NewTestAllocationIDGenerator creates a new TestAllocationIDGenerator
func NewTestAllocationIDGenerator() *TestAllocationIDGenerator {
	return &TestAllocationIDGenerator{}
}

// GenerateManifestKey generates the key for manifest.Allocations map
func (g *TestAllocationIDGenerator) GenerateManifestKey(nodeID, allocName string) (string, error) {
	return fmt.Sprintf("%s.%s", nodeID, allocName), nil
}

// GenerateFullAllocationID generates the full allocation ID for actor handles
func (g *TestAllocationIDGenerator) GenerateFullAllocationID(ensembleID, nodeID, allocName string) (string, error) {
	return fmt.Sprintf("%s_%s.%s", ensembleID, nodeID, allocName), nil
}

// ValidateManifestKey validates a manifest key format
func (g *TestAllocationIDGenerator) ValidateManifestKey(manifestKey string) error {
	if manifestKey == "" {
		return fmt.Errorf("manifest key cannot be empty")
	}
	return nil
}

// ValidateFullAllocationID validates a full allocation ID format
func (g *TestAllocationIDGenerator) ValidateFullAllocationID(allocID string) error {
	if allocID == "" {
		return fmt.Errorf("allocation ID cannot be empty")
	}
	return nil
}

// FailingAllocationIDGenerator for testing error cases
type FailingAllocationIDGenerator struct{}

// NewFailingAllocationIDGenerator creates a new FailingAllocationIDGenerator
func NewFailingAllocationIDGenerator() *FailingAllocationIDGenerator {
	return &FailingAllocationIDGenerator{}
}

// GenerateManifestKey generates the key for manifest.Allocations map
func (g *FailingAllocationIDGenerator) GenerateManifestKey(_, _ string) (string, error) {
	// This generator always returns the same key to test conflicts
	return "conflict.key", nil
}

// GenerateFullAllocationID generates the full allocation ID for actor handles
func (g *FailingAllocationIDGenerator) GenerateFullAllocationID(_, _, _ string) (string, error) {
	// This generator always returns the same ID to test conflicts
	return "ensemble_conflict.key", nil
}

// ValidateManifestKey validates a manifest key format
func (g *FailingAllocationIDGenerator) ValidateManifestKey(manifestKey string) error {
	if manifestKey == "" {
		return fmt.Errorf("manifest key cannot be empty")
	}
	return nil
}

// ValidateFullAllocationID validates a full allocation ID format
func (g *FailingAllocationIDGenerator) ValidateFullAllocationID(allocID string) error {
	if allocID == "" {
		return fmt.Errorf("allocation ID cannot be empty")
	}
	return nil
}

// TestGeneratorValidator for testing validator behavior
type TestGeneratorValidator struct{}

// NewTestGeneratorValidator creates a new TestGeneratorValidator
func NewTestGeneratorValidator() *TestGeneratorValidator {
	return &TestGeneratorValidator{}
}

// ValidateAllocationIDGenerator validates that the generator won't cause conflicts
func (v *TestGeneratorValidator) ValidateAllocationIDGenerator(generator AllocationIDGenerator) error {
	// Simple test - just check if generator can generate different keys for primary and standby
	primaryKey, err := generator.GenerateManifestKey("node1", "alloc1")
	if err != nil {
		return fmt.Errorf("failed to generate primary key: %w", err)
	}

	standbyKey, err := generator.GenerateManifestKey("node1-standby-1", "alloc1")
	if err != nil {
		return fmt.Errorf("failed to generate standby key: %w", err)
	}

	if primaryKey == standbyKey {
		return fmt.Errorf("conflict detected: primary key '%s' == standby key '%s'", primaryKey, standbyKey)
	}

	return nil
}

// ValidateNodeIDGenerator validates that the generator won't cause conflicts
func (v *TestGeneratorValidator) ValidateNodeIDGenerator(generator NodeIDGenerator) error {
	// Simple test - just validate that generator can handle basic node IDs
	if err := generator.ValidateNodeID("node1"); err != nil {
		return fmt.Errorf("failed to validate primary node ID: %w", err)
	}

	if err := generator.ValidateNodeID("node1-standby-1"); err != nil {
		return fmt.Errorf("failed to validate standby node ID: %w", err)
	}

	return nil
}
