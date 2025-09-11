// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultAllocationIDGenerator_GenerateManifestKey(t *testing.T) {
	generator := NewDefaultAllocationIDGenerator()

	t.Run("successful generation", func(t *testing.T) {
		key, err := generator.GenerateManifestKey("node1", "alloc1")
		require.NoError(t, err)
		assert.Equal(t, "node1.alloc1", key)
	})

	t.Run("successful generation with standby node", func(t *testing.T) {
		key, err := generator.GenerateManifestKey("node1-standby-1", "alloc1")
		require.NoError(t, err)
		assert.Equal(t, "node1-standby-1.alloc1", key)
	})

	t.Run("empty nodeID", func(t *testing.T) {
		_, err := generator.GenerateManifestKey("", "alloc1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nodeID and allocName cannot be empty")
	})

	t.Run("empty allocName", func(t *testing.T) {
		_, err := generator.GenerateManifestKey("node1", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nodeID and allocName cannot be empty")
	})

	t.Run("both empty", func(t *testing.T) {
		_, err := generator.GenerateManifestKey("", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nodeID and allocName cannot be empty")
	})
}

func TestDefaultAllocationIDGenerator_GenerateFullAllocationID(t *testing.T) {
	generator := NewDefaultAllocationIDGenerator()

	t.Run("successful generation", func(t *testing.T) {
		id, err := generator.GenerateFullAllocationID("ensemble1", "node1", "alloc1")
		require.NoError(t, err)
		assert.Equal(t, "ensemble1_node1.alloc1", id)
	})

	t.Run("successful generation with standby node", func(t *testing.T) {
		id, err := generator.GenerateFullAllocationID("ensemble1", "node1-standby-1", "alloc1")
		require.NoError(t, err)
		assert.Equal(t, "ensemble1_node1-standby-1.alloc1", id)
	})

	t.Run("empty ensembleID", func(t *testing.T) {
		_, err := generator.GenerateFullAllocationID("", "node1", "alloc1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ensembleID, nodeID, and allocName cannot be empty")
	})

	t.Run("empty nodeID", func(t *testing.T) {
		_, err := generator.GenerateFullAllocationID("ensemble1", "", "alloc1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ensembleID, nodeID, and allocName cannot be empty")
	})

	t.Run("empty allocName", func(t *testing.T) {
		_, err := generator.GenerateFullAllocationID("ensemble1", "node1", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ensembleID, nodeID, and allocName cannot be empty")
	})
}

func TestDefaultAllocationIDGenerator_ValidateManifestKey(t *testing.T) {
	generator := NewDefaultAllocationIDGenerator()

	t.Run("valid manifest key", func(t *testing.T) {
		err := generator.ValidateManifestKey("node1.alloc1")
		assert.NoError(t, err)
	})

	t.Run("valid manifest key with standby", func(t *testing.T) {
		err := generator.ValidateManifestKey("node1-standby-1.alloc1")
		assert.NoError(t, err)
	})

	t.Run("empty manifest key", func(t *testing.T) {
		err := generator.ValidateManifestKey("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "manifest key cannot be empty")
	})

	t.Run("invalid format - no dot", func(t *testing.T) {
		err := generator.ValidateManifestKey("node1alloc1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid manifest key format")
	})

	t.Run("invalid format - multiple dots", func(t *testing.T) {
		err := generator.ValidateManifestKey("node1.alloc1.sub")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid manifest key format")
	})

	t.Run("invalid format - empty nodeID", func(t *testing.T) {
		err := generator.ValidateManifestKey(".alloc1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nodeID and allocName cannot be empty")
	})

	t.Run("invalid format - empty allocName", func(t *testing.T) {
		err := generator.ValidateManifestKey("node1.")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nodeID and allocName cannot be empty")
	})
}

func TestDefaultAllocationIDGenerator_ValidateFullAllocationID(t *testing.T) {
	generator := NewDefaultAllocationIDGenerator()

	t.Run("valid full allocation ID", func(t *testing.T) {
		err := generator.ValidateFullAllocationID("ensemble1_node1.alloc1")
		assert.NoError(t, err)
	})

	t.Run("valid full allocation ID with standby", func(t *testing.T) {
		err := generator.ValidateFullAllocationID("ensemble1_node1-standby-1.alloc1")
		assert.NoError(t, err)
	})

	t.Run("empty allocation ID", func(t *testing.T) {
		err := generator.ValidateFullAllocationID("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "allocation ID cannot be empty")
	})

	t.Run("invalid format - no underscore", func(t *testing.T) {
		err := generator.ValidateFullAllocationID("ensemble1node1.alloc1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid allocation ID format")
	})

	t.Run("invalid format - empty ensembleID", func(t *testing.T) {
		err := generator.ValidateFullAllocationID("_node1.alloc1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ensemble ID cannot be empty")
	})

	t.Run("invalid format - invalid manifest key part", func(t *testing.T) {
		err := generator.ValidateFullAllocationID("ensemble1_node1alloc1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid allocation ID format")
	})
}

func TestDefaultNodeIDGenerator_GenerateNodeID(t *testing.T) {
	generator := NewDefaultNodeIDGenerator()

	t.Run("successful generation", func(t *testing.T) {
		id, err := generator.GenerateNodeID("node1")
		require.NoError(t, err)
		assert.Equal(t, "node1", id)
	})

	t.Run("empty base name", func(t *testing.T) {
		_, err := generator.GenerateNodeID("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "base name cannot be empty")
	})
}

func TestDefaultNodeIDGenerator_ValidateNodeID(t *testing.T) {
	generator := NewDefaultNodeIDGenerator()

	t.Run("valid node ID", func(t *testing.T) {
		err := generator.ValidateNodeID("node1")
		assert.NoError(t, err)
	})

	t.Run("valid standby node ID", func(t *testing.T) {
		err := generator.ValidateNodeID("node1-standby-1")
		assert.NoError(t, err)
	})

	t.Run("empty node ID", func(t *testing.T) {
		err := generator.ValidateNodeID("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "node ID cannot be empty")
	})
}

func TestDefaultNodeIDGenerator_ParseNodeID(t *testing.T) {
	generator := NewDefaultNodeIDGenerator()

	t.Run("parse primary node", func(t *testing.T) {
		isStandby, primaryNodeID, standbyIndex, err := generator.ParseNodeID("node1")
		require.NoError(t, err)
		assert.False(t, isStandby)
		assert.Equal(t, "node1", primaryNodeID)
		assert.Equal(t, 0, standbyIndex)
	})

	t.Run("parse standby node", func(t *testing.T) {
		isStandby, primaryNodeID, standbyIndex, err := generator.ParseNodeID("node1-standby-1")
		require.NoError(t, err)
		assert.True(t, isStandby)
		assert.Equal(t, "node1", primaryNodeID)
		assert.Equal(t, 1, standbyIndex)
	})

	t.Run("parse standby node with higher index", func(t *testing.T) {
		isStandby, primaryNodeID, standbyIndex, err := generator.ParseNodeID("node1-standby-5")
		require.NoError(t, err)
		assert.True(t, isStandby)
		assert.Equal(t, "node1", primaryNodeID)
		assert.Equal(t, 5, standbyIndex)
	})
}

func TestDefaultGeneratorValidator_ValidateAllocationIDGenerator(t *testing.T) {
	validator := NewDefaultGeneratorValidator()

	t.Run("valid generator", func(t *testing.T) {
		generator := NewDefaultAllocationIDGenerator()
		err := validator.ValidateAllocationIDGenerator(generator)
		assert.NoError(t, err)
	})

	t.Run("failing generator with conflicts", func(t *testing.T) {
		generator := NewFailingAllocationIDGenerator()
		err := validator.ValidateAllocationIDGenerator(generator)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "manifest key conflict")
	})
}

func TestDefaultGeneratorValidator_ValidateNodeIDGenerator(t *testing.T) {
	validator := NewDefaultGeneratorValidator()

	t.Run("valid generator", func(t *testing.T) {
		generator := NewDefaultNodeIDGenerator()
		err := validator.ValidateNodeIDGenerator(generator)
		assert.NoError(t, err)
	})
}

func TestTestAllocationIDGenerator(t *testing.T) {
	generator := NewTestAllocationIDGenerator()

	t.Run("generate manifest key", func(t *testing.T) {
		key, err := generator.GenerateManifestKey("node1", "alloc1")
		require.NoError(t, err)
		assert.Equal(t, "node1.alloc1", key)
	})

	t.Run("generate full allocation ID", func(t *testing.T) {
		id, err := generator.GenerateFullAllocationID("ensemble1", "node1", "alloc1")
		require.NoError(t, err)
		assert.Equal(t, "ensemble1_node1.alloc1", id)
	})

	t.Run("validate manifest key", func(t *testing.T) {
		err := generator.ValidateManifestKey("node1.alloc1")
		assert.NoError(t, err)
	})

	t.Run("validate full allocation ID", func(t *testing.T) {
		err := generator.ValidateFullAllocationID("ensemble1_node1.alloc1")
		assert.NoError(t, err)
	})
}

func TestTestNodeIDGenerator(t *testing.T) {
	generator := NewTestNodeIDGenerator()

	t.Run("generate node ID", func(t *testing.T) {
		id1, err := generator.GenerateNodeID("node1")
		require.NoError(t, err)
		assert.Equal(t, "node1-test-1", id1)

		id2, err := generator.GenerateNodeID("node2")
		require.NoError(t, err)
		assert.Equal(t, "node2-test-2", id2)
	})

	t.Run("validate node ID", func(t *testing.T) {
		err := generator.ValidateNodeID("node1")
		assert.NoError(t, err)
	})

	t.Run("parse node ID", func(t *testing.T) {
		isStandby, primaryNodeID, standbyIndex, err := generator.ParseNodeID("node1-standby-1")
		require.NoError(t, err)
		assert.True(t, isStandby)
		assert.Equal(t, "node1", primaryNodeID)
		assert.Equal(t, 1, standbyIndex)
	})
}

func TestTestGeneratorValidator(t *testing.T) {
	validator := NewTestGeneratorValidator()

	t.Run("validate allocation ID generator", func(t *testing.T) {
		generator := NewTestAllocationIDGenerator()
		err := validator.ValidateAllocationIDGenerator(generator)
		assert.NoError(t, err)
	})

	t.Run("validate node ID generator", func(t *testing.T) {
		generator := NewTestNodeIDGenerator()
		err := validator.ValidateNodeIDGenerator(generator)
		assert.NoError(t, err)
	})
}

func TestFailingAllocationIDGenerator(t *testing.T) {
	generator := NewFailingAllocationIDGenerator()

	t.Run("always returns same manifest key", func(t *testing.T) {
		key1, err := generator.GenerateManifestKey("node1", "alloc1")
		require.NoError(t, err)
		assert.Equal(t, "conflict.key", key1)

		key2, err := generator.GenerateManifestKey("node2", "alloc2")
		require.NoError(t, err)
		assert.Equal(t, "conflict.key", key2)
	})

	t.Run("always returns same full allocation ID", func(t *testing.T) {
		id1, err := generator.GenerateFullAllocationID("ensemble1", "node1", "alloc1")
		require.NoError(t, err)
		assert.Equal(t, "ensemble_conflict.key", id1)

		id2, err := generator.GenerateFullAllocationID("ensemble2", "node2", "alloc2")
		require.NoError(t, err)
		assert.Equal(t, "ensemble_conflict.key", id2)
	})
}
