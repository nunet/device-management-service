// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package types

// Mounter is responsible for mounting and unmounting a volume.
type Mounter interface {
	Mount(targetPath string, options map[string]string) error
	Unmount(targetPath string) error
}

type VolumeConfig struct {
	// The type of storage backend, e.g., "glusterfs" or "local".
	Type             string `json:"type" yaml:"type"`
	MountDestination string `json:"mount_destination" yaml:"mount_destination"` // the mount path inside the container
	ReadOnly         bool   `json:"read_only" yaml:"read_only"`

	Name             string   `json:"name,omitempty" yaml:"name,omitempty"`
	Servers          []string `json:"servers,omitempty" yaml:"servers,omitempty"`
	ClientPrivateKey string   `json:"client_private_key,omitempty" yaml:"client_private_key,omitempty"`
	ClientPEM        string   `json:"client_pem,omitempty" yaml:"client_pem,omitempty"`
	ClientCA         string   `json:"client_ca,omitempty" yaml:"client_ca,omitempty"`

	// Local
	Src string `json:"src,omitempty" yaml:"src,omitempty"`
}
