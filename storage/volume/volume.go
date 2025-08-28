// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package volume

import (
	"fmt"

	"gitlab.com/nunet/device-management-service/storage"
	"gitlab.com/nunet/device-management-service/storage/volume/glusterfs"
	"gitlab.com/nunet/device-management-service/storage/volume/localfs"
	"gitlab.com/nunet/device-management-service/types"
)

// New creates a volume implementation based on the provided configuration.
func New(t *storage.VolumeTracker, sc types.VolumeConfig, allocationID string) (types.Mounter, error) {
	switch sc.Type {
	case "glusterfs":
		return glusterfs.New(t, sc.Servers, sc.Name, sc.ClientPrivateKey, sc.ClientPEM, sc.ClientCA, allocationID)
	case "local":
		return localfs.New(sc.Src)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", sc.Type)
	}
}
