// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build darwin
// +build darwin

package localfs

import (
	"fmt"
	"os/exec"

	"gitlab.com/nunet/device-management-service/types"
)

type LocalFS struct {
	path string
}

var _ types.Mounter = (*LocalFS)(nil)

// New creates a new LocalFS storage instance using the provided path.
func New(path string) (*LocalFS, error) {
	if path == "" {
		return nil, fmt.Errorf("local filesystem path cannot be empty")
	}
	return &LocalFS{path: path}, nil
}

// Mount for LocalFS might perform a bind mount or simply check that the path exists.
func (l *LocalFS) Mount(targetPath string, _ map[string]string) error {
	if targetPath == "" {
		return fmt.Errorf("target path cannot be empty")
	}

	// perform a bind mount: mount --bind <l.path> <targetPath>
	cmd := exec.Command("mount", "--bind", l.path, targetPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to mount local filesystem: %w, output: %s", err, output)
	}
	return nil
}

func (l *LocalFS) Unmount(targetPath string) error {
	if targetPath == "" {
		return fmt.Errorf("target path cannot be empty")
	}
	cmd := exec.Command("umount", targetPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to unmount local filesystem: %w, output: %s", err, output)
	}
	return nil
}
