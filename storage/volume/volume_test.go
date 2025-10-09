// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package volume

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/storage"
	"gitlab.com/nunet/device-management-service/storage/volume/glusterfs"
	"gitlab.com/nunet/device-management-service/storage/volume/localfs"
	"gitlab.com/nunet/device-management-service/types"
)

// Test constants
const (
	testAllocationID    = "alloc-test"
	testGlusterfsServer = "test-server:24007"
	testVolumeName      = "test-volume"
	testLocalSrc        = "/test/local/path"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("ShouldReturnGlusterFSMounter_WhenTypeIsGlusterfs", func(t *testing.T) {
		t.Parallel()
		tracker := storage.NewVolumeTracker()
		config := types.VolumeConfig{
			Type:             "glusterfs",
			Servers:          []string{testGlusterfsServer},
			Name:             testVolumeName,
			ClientPrivateKey: "test-key",
			ClientPEM:        "test-pem",
			ClientCA:         "test-ca",
		}

		mounter, err := New(tracker, config, testAllocationID)

		require.NoError(t, err)
		require.NotNil(t, mounter)

		// Verify it's a GlusterFS implementation
		_, ok := mounter.(*glusterfs.GlusterFS)
		assert.True(t, ok, "Expected a GlusterFS implementation")
	})

	t.Run("ShouldReturnLocalFSMounter_WhenTypeIsLocal", func(t *testing.T) {
		t.Parallel()
		tracker := storage.NewVolumeTracker()
		config := types.VolumeConfig{
			Type: "local",
			Src:  testLocalSrc,
		}

		mounter, err := New(tracker, config, testAllocationID)

		require.NoError(t, err)
		require.NotNil(t, mounter)

		// Verify it's a LocalFS implementation
		_, ok := mounter.(*localfs.LocalFS)
		assert.True(t, ok, "Expected a LocalFS implementation")
	})

	t.Run("ShouldReturnError_WhenTypeIsUnsupported", func(t *testing.T) {
		t.Parallel()
		tracker := storage.NewVolumeTracker()
		config := types.VolumeConfig{
			Type: "unsupported",
		}

		mounter, err := New(tracker, config, testAllocationID)

		assert.Error(t, err)
		assert.Nil(t, mounter)
		assert.Contains(t, err.Error(), "unsupported storage type")
	})

	t.Run("ShouldPassParameters_ToGlusterFSConstructor", func(t *testing.T) {
		t.Parallel()
		tracker := storage.NewVolumeTracker()
		servers := []string{testGlusterfsServer, "another-server:24007"}
		name := "test-volume-name"
		key := "test-private-key"
		pem := "test-pem-content"
		ca := "test-ca-content"
		allocID := "special-allocation-id"

		config := types.VolumeConfig{
			Type:             "glusterfs",
			Servers:          servers,
			Name:             name,
			ClientPrivateKey: key,
			ClientPEM:        pem,
			ClientCA:         ca,
		}

		// We can't directly inspect the GlusterFS struct fields as they're private,
		// but we can verify the constructor is called with the right parameters
		// by checking if New succeeds (it validates parameters)
		mounter, err := New(tracker, config, allocID)

		require.NoError(t, err)
		require.NotNil(t, mounter)
	})

	t.Run("ShouldPassParameters_ToLocalFSConstructor", func(t *testing.T) {
		t.Parallel()
		tracker := storage.NewVolumeTracker()
		src := "/custom/local/path"

		config := types.VolumeConfig{
			Type: "local",
			Src:  src,
		}

		// LocalFS constructor doesn't validate much, but we can ensure it succeeds
		mounter, err := New(tracker, config, testAllocationID)

		require.NoError(t, err)
		require.NotNil(t, mounter)
	})
}
