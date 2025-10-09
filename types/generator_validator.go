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

// DefaultGeneratorValidator is the default implementation of GeneratorValidator
type DefaultGeneratorValidator struct{}

// NewDefaultGeneratorValidator creates a new DefaultGeneratorValidator
func NewDefaultGeneratorValidator() *DefaultGeneratorValidator {
	return &DefaultGeneratorValidator{}
}

// ValidateAllocationIDGenerator validates that the generator won't cause conflicts
// when given the same allocation name but different node types (primary vs standby)
func (v *DefaultGeneratorValidator) ValidateAllocationIDGenerator(generator AllocationIDGenerator) error {
	// Test cases to validate the generator won't cause conflicts
	testCases := []struct {
		name        string
		ensembleID  string
		primaryNode string
		standbyNode string
		allocName   string
	}{
		{
			name:        "basic primary and standby",
			ensembleID:  "test-ensemble",
			primaryNode: "node1",
			standbyNode: "node1-standby-1",
			allocName:   "alloc1",
		},
		{
			name:        "complex node names",
			ensembleID:  "ensemble-123",
			primaryNode: "web-server-01",
			standbyNode: "web-server-01-standby-2",
			allocName:   "nginx-service",
		},
		{
			name:        "edge case with numbers",
			ensembleID:  "e1",
			primaryNode: "n1",
			standbyNode: "n1-standby-1",
			allocName:   "a1",
		},
	}

	for _, tc := range testCases {
		// Generate manifest keys for primary and standby
		primaryManifestKey, err := generator.GenerateManifestKey(tc.primaryNode, tc.allocName)
		if err != nil {
			return fmt.Errorf("failed to generate primary manifest key for test case '%s': %w", tc.name, err)
		}

		standbyManifestKey, err := generator.GenerateManifestKey(tc.standbyNode, tc.allocName)
		if err != nil {
			return fmt.Errorf("failed to generate standby manifest key for test case '%s': %w", tc.name, err)
		}

		// Check if manifest keys conflict
		if primaryManifestKey == standbyManifestKey {
			return fmt.Errorf("manifest key conflict in test case '%s': primary key '%s' == standby key '%s'",
				tc.name, primaryManifestKey, standbyManifestKey)
		}

		// Generate full allocation IDs for primary and standby
		primaryFullID, err := generator.GenerateFullAllocationID(tc.ensembleID, tc.primaryNode, tc.allocName)
		if err != nil {
			return fmt.Errorf("failed to generate primary full allocation ID for test case '%s': %w", tc.name, err)
		}

		standbyFullID, err := generator.GenerateFullAllocationID(tc.ensembleID, tc.standbyNode, tc.allocName)
		if err != nil {
			return fmt.Errorf("failed to generate standby full allocation ID for test case '%s': %w", tc.name, err)
		}

		// Check if full allocation IDs conflict
		if primaryFullID == standbyFullID {
			return fmt.Errorf("full allocation ID conflict in test case '%s': primary ID '%s' == standby ID '%s'",
				tc.name, primaryFullID, standbyFullID)
		}

		// Validate that the generated keys are properly formatted
		if err := generator.ValidateManifestKey(primaryManifestKey); err != nil {
			return fmt.Errorf("invalid primary manifest key in test case '%s': %w", tc.name, err)
		}

		if err := generator.ValidateManifestKey(standbyManifestKey); err != nil {
			return fmt.Errorf("invalid standby manifest key in test case '%s': %w", tc.name, err)
		}

		if err := generator.ValidateFullAllocationID(primaryFullID); err != nil {
			return fmt.Errorf("invalid primary full allocation ID in test case '%s': %w", tc.name, err)
		}

		if err := generator.ValidateFullAllocationID(standbyFullID); err != nil {
			return fmt.Errorf("invalid standby full allocation ID in test case '%s': %w", tc.name, err)
		}
	}

	return nil
}

// ValidateNodeIDGenerator validates that the generator won't cause conflicts
func (v *DefaultGeneratorValidator) ValidateNodeIDGenerator(generator NodeIDGenerator) error {
	// Test cases to validate the generator won't cause conflicts
	testCases := []struct {
		name        string
		primaryNode string
		standbyNode string
	}{
		{
			name:        "basic primary and standby",
			primaryNode: "node1",
			standbyNode: "node1-standby-1",
		},
		{
			name:        "complex node names",
			primaryNode: "web-server-01",
			standbyNode: "web-server-01-standby-2",
		},
	}

	for _, tc := range testCases {
		// Validate that the generator can handle both primary and standby node IDs
		if err := generator.ValidateNodeID(tc.primaryNode); err != nil {
			return fmt.Errorf("invalid primary node ID in test case '%s': %w", tc.name, err)
		}

		if err := generator.ValidateNodeID(tc.standbyNode); err != nil {
			return fmt.Errorf("invalid standby node ID in test case '%s': %w", tc.name, err)
		}

		// Test parsing functionality
		isStandby, _, _, err := generator.ParseNodeID(tc.primaryNode)
		if err != nil {
			return fmt.Errorf("failed to parse primary node ID in test case '%s': %w", tc.name, err)
		}
		if isStandby {
			return fmt.Errorf("primary node ID incorrectly identified as standby in test case '%s'", tc.name)
		}

		isStandby, parsedPrimaryNodeID, standbyIndex, err := generator.ParseNodeID(tc.standbyNode)
		if err != nil {
			return fmt.Errorf("failed to parse standby node ID in test case '%s': %w", tc.name, err)
		}
		if !isStandby {
			return fmt.Errorf("standby node ID incorrectly identified as primary in test case '%s'", tc.name)
		}
		if parsedPrimaryNodeID != tc.primaryNode {
			return fmt.Errorf("incorrect primary node ID parsed from standby node in test case '%s': expected '%s', got '%s'",
				tc.name, tc.primaryNode, parsedPrimaryNodeID)
		}
		if standbyIndex < 1 {
			return fmt.Errorf("invalid standby index in test case '%s': %d", tc.name, standbyIndex)
		}
	}

	return nil
}
