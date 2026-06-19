// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package volume

import (
	"path/filepath"
	"strings"

	"gitlab.com/nunet/device-management-service/types"
)

// DirName returns the subdirectory name for a volume under <work_dir>/volumes/<ownerID>/
// Src is used when set. Name is fallback (e.g. glusterfs volumes without src)
func DirName(v types.VolumeConfig) string {
	if v.Src != "" {
		if !strings.Contains(v.Src, "/") {
			return v.Src
		}

		path := filepath.Clean(v.Src)
		path = strings.TrimPrefix(path, string(filepath.Separator))
		return strings.ReplaceAll(path, string(filepath.Separator), "_")
	}

	if v.Name != "" {
		return v.Name
	}

	return ""
}

// HostPath returns the host path where a volume is mounted before the executor bind-mounts it
func HostPath(workDir, ownerID string, v types.VolumeConfig) string {
	return filepath.Join(workDir, "volumes", ownerID, DirName(v))
}
