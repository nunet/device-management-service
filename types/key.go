// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package types

type AllocationKeyType string

const (
	KeySSH AllocationKeyType = "ssh"
	KeyGPG AllocationKeyType = "gpg"
)

// AllocationKey is a key specification to be uploaded on the allocation, e.g. ssh, gpg
type AllocationKey struct {
	Type AllocationKeyType `json:"type"`
	File string            `json:"file"` // source path to file
	Dest string            `json:"dest"` // destination path
}

func (t AllocationKeyType) Equal(other string) bool {
	return string(t) == other
}

func (t AllocationKeyType) String() string {
	return string(t)
}
