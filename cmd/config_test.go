// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/internal/config"
)

func ExecuteCommand(root *cobra.Command, args ...string) (output string, err error) {
	_, output, err = ExecuteCommandC(root, args...)
	return output, err
}

func ExecuteCommandC(root *cobra.Command, args ...string) (c *cobra.Command, output string, err error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	c, err = root.ExecuteC()

	return c, buf.String(), err
}

func TestConfigGetCmd(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		configExists bool
		wantErr      bool
	}{
		{
			name:    "no args",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "invalid arg",
			args:    []string{"test"},
			wantErr: true,
		},
		{
			name:    "more than one arg",
			args:    []string{"test", "test"},
			wantErr: true,
		},
		{
			name:    "valid arg",
			args:    []string{"general.work_dir"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newConfigGetCmd(afero.NewMemMapFs())
			out, err := ExecuteCommand(cmd, tt.args...)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if len(tt.args) == 0 {
					// The first line of the output should be the path to the config file
					// The json, which is the second line, should be unmarshalled into a Config struct
					lines := strings.SplitN(out, "\n", 2)
					if len(lines) > 1 {
						jsonOutput := lines[1]
						assert.NoError(t, json.Unmarshal([]byte(jsonOutput), &config.Config{}))
					} else {
						assert.Fail(t, "output is empty")
					}
				}
			}
		})
	}
}

func TestConfigSetCmd(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		configExists bool
		wantErr      bool
	}{
		{
			name:    "no args",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "invalid arg",
			args:    []string{"testing"},
			wantErr: true,
		},
		{
			name:    "more than allowed args",
			args:    []string{"test", "1234", "extra"},
			wantErr: true,
		},
		{
			name:    "nonexistent key",
			args:    []string{"test", "1234"},
			wantErr: true,
		},
		{
			name:    "existent key upper case",
			args:    []string{"GENERAL.WORK_dir", "/usr/test/"},
			wantErr: false,
		},
		{
			name:    "existent key string value",
			args:    []string{"general.work_dir", "~/.config/dms"},
			wantErr: false,
		},
		{
			name:    "existent key numeric value",
			args:    []string{"rest.port", "8766"},
			wantErr: false,
		},
		{
			name:    "existent key wrong type",
			args:    []string{"rest.port", "test"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newConfigSetCmd(afero.NewMemMapFs())
			_, err := ExecuteCommand(cmd, tt.args...)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigEditCmd(t *testing.T) {
	tests := []struct {
		name    string
		editor  string
		wantErr bool
	}{
		{
			name:    "editor not set",
			editor:  "",
			wantErr: true,
		},
		{
			name:    "editor set",
			editor:  "true", // maybe this is not the best way to test this
			wantErr: false,
		},
	}
	for _, tt := range tests {
		if path, err := exec.LookPath(tt.editor); tt.editor != "" && (err != nil || path == "") {
			t.Skipf("%s bin not found to simulate editor:", tt.editor)
		}

		t.Run(tt.name, func(t *testing.T) {
			// Set the EDITOR environment variable for the test
			if tt.editor != "" {
				t.Setenv("EDITOR", tt.editor)
			} else {
				os.Unsetenv("EDITOR")
			}

			cmd := newConfigEditCmd(afero.NewMemMapFs())
			_, err := ExecuteCommand(cmd)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
