// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package containerd

import (
	"fmt"
	"os"
	"path/filepath"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

const (
	initScriptsBaseDir = "/tmp/nunet/init-scripts-"
	provisionMountPath = "/run/nunet/provision"
)

func prepInitScripts(scripts map[string][]byte, id string) (string, error) {
	if len(scripts) == 0 {
		return "", nil
	}

	tempDir := initScriptsBaseDir + id
	if err := os.MkdirAll(tempDir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create init scripts base directory: %w", err)
	}

	scriptNames := make([]string, 0, len(scripts))
	for name, content := range scripts {
		filename := filepath.Join(tempDir, name)
		if err := os.WriteFile(filename, content, 0o700); err != nil {
			return "", fmt.Errorf("failed to write init script %s: %w", name, err)
		}
		scriptNames = append(scriptNames, name)
	}

	wrapperContent := "#!/bin/sh\n\n"
	for _, script := range scriptNames {
		wrapperContent += fmt.Sprintf("echo 'Executing %s'\n", script)
		wrapperContent += fmt.Sprintf("%s/%s\n", provisionMountPath, script)
	}

	wrapperPath := filepath.Join(tempDir, "run_provision_scripts.sh")
	if err := os.WriteFile(wrapperPath, []byte(wrapperContent), 0o700); err != nil {
		return "", fmt.Errorf("failed to write wrapper script: %w", err)
	}

	return tempDir, nil
}

func provisionBindMount(hostDir string) specs.Mount {
	return specs.Mount{
		Type:        "bind",
		Source:      hostDir,
		Destination: provisionMountPath,
		Options:     []string{"bind", "ro", "nosuid", "nodev"},
	}
}

func removeInitScriptsDir(id string) error {
	return os.RemoveAll(initScriptsBaseDir + id)
}

func removeAllInitScriptsDirs() {
	matches, err := filepath.Glob(initScriptsBaseDir + "*")
	if err != nil {
		log.Warnw("failed to find init script directories", "error", err)
		return
	}

	for _, dir := range matches {
		if err := os.RemoveAll(dir); err != nil {
			log.Warnw("failed to remove init script directory", "dir", dir, "error", err)
		}
	}
}
