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

	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/types"
)

func TestDirName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		v    types.VolumeConfig
		want string
	}{
		{
			name: "glusterfs without src falls back to name",
			v: types.VolumeConfig{
				Type: "glusterfs",
				Name: "my-gluster-vol",
			},
			want: "my-gluster-vol",
		},
		{
			name: "local named volume uses src",
			v: types.VolumeConfig{
				Type: "local",
				Src:  "my-data",
			},
			want: "my-data",
		},
		{
			name: "local absolute path is sanitized",
			v: types.VolumeConfig{
				Type: "local",
				Src:  "/data/volume1",
			},
			want: "data_volume1",
		},
		{
			name: "src takes precedence over name",
			v: types.VolumeConfig{
				Type: "local",
				Name: "named",
				Src:  "/other/path",
			},
			want: "other_path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, DirName(tt.v))
		})
	}
}

func TestHostPath(t *testing.T) {
	t.Parallel()

	v := types.VolumeConfig{
		Type: "local",
		Src:  "my-data",
	}

	got := HostPath("/work", "alloc-1", v)
	require.Equal(t, "/work/volumes/alloc-1/my-data", got)
}
