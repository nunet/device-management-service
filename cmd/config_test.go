package cmd

import (
	"bytes"
	"encoding/json"
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

func Test_ConfigGetCmd(t *testing.T) {
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
			cmd := newConfigGetCmd()
			out, err := ExecuteCommand(cmd, tt.args...)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if len(tt.args) == 0 {
					assert.NoError(t, json.Unmarshal([]byte(out), &config.Config{}))
				}
			}
		})
	}
}

func Test_ConfigSetCmd(t *testing.T) {
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
