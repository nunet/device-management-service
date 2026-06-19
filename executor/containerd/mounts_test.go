// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package containerd

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/types"
)

func TestMakeMounts(t *testing.T) {
	t.Parallel()

	mounts, err := makeMounts(
		[]*types.StorageVolumeExecutor{
			{Type: "bind", Source: "/host/input", Target: "/input", ReadOnly: true},
		},
		[]*types.StorageVolumeExecutor{
			{Type: "bind", Source: "/host/output", Target: "/output"},
		},
		"/host/results",
	)
	require.NoError(t, err)
	require.Len(t, mounts, 2)

	require.Equal(t, "/host/input", mounts[0].Source)
	require.Equal(t, "/input", mounts[0].Destination)
	require.Equal(t, []string{"rbind", "ro"}, mounts[0].Options)

	require.Equal(t, "/host/output", mounts[1].Source)
	require.Equal(t, "/output", mounts[1].Destination)
	require.Equal(t, []string{"rbind"}, mounts[1].Options)
}

func TestMakeMountsBindMountsNamedSource(t *testing.T) {
	t.Parallel()

	mounts, err := makeMounts(
		[]*types.StorageVolumeExecutor{
			{Type: "volume", Source: "my-volume", Target: "/data"},
		},
		nil,
		"/results",
	)
	require.NoError(t, err)
	require.Len(t, mounts, 1)
	require.Equal(t, "bind", mounts[0].Type)
	require.Equal(t, "my-volume", mounts[0].Source)
	require.Equal(t, "/data", mounts[0].Destination)
}

func TestMakeMountsRequiresResultsDirForOutputs(t *testing.T) {
	t.Parallel()

	_, err := makeMounts(
		nil,
		[]*types.StorageVolumeExecutor{
			{Type: "bind", Source: "/host/output", Target: "/output"},
		},
		"",
	)
	require.Error(t, err)
}
