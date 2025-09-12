package cmd

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/cmd/cli"
	"gitlab.com/nunet/device-management-service/cmd/utils"
)

func TestTranslateCmd(t *testing.T) {
	inputFile := `
services:
  nginx:
    image: nginxdemos/hello:plain-text
    command: [nginx-debug, '-g', 'daemon off;']
    ports:
      - "8080:80"
    environment:
      - NGINX_PORT=80
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080"]
      interval: 10s
      timeout: 10s
      retries: 3
`
	tests := []struct {
		name       string
		args       []string
		files      map[string]string
		outputFile string
		wantErr    bool
	}{
		{
			name:    "no args",
			args:    []string{},
			wantErr: true,
		},
		{
			name: "invalid format",
			args: []string{"-f", "invalid", "compose.yaml"},
			files: map[string]string{
				"compose.yaml": inputFile,
			},
			wantErr: true,
		},
		{
			name: "valid compose file",
			args: []string{"compose.yaml"},
			files: map[string]string{
				"compose.yaml": inputFile,
			},
			wantErr: false,
		},
		{
			name: "valid compose file with source format",
			args: []string{"-f", "docker-compose", "compose.yaml"},
			files: map[string]string{
				"compose.yaml": inputFile,
			},
			wantErr: false,
		},
		{
			name: "valid compose file with output file",
			args: []string{"-o", "test.yaml", "compose.yaml"},
			files: map[string]string{
				"compose.yaml": inputFile,
			},
			outputFile: "test.yaml",
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.Afero{Fs: afero.NewMemMapFs()}
			if tt.files != nil {
				for filePath, content := range tt.files {
					err := fs.WriteFile(filePath, []byte(content), 0o644)
					require.NoError(t, err, "Failed to write test file")
				}
			}
			dmsCli := utils.NewTestCli(cli.WithFS(fs))
			cmd := newTranslateCmd(dmsCli)
			out, _, err := utils.ExecuteCommand(cmd, tt.args...)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.outputFile != "" {
					exists, err := fs.Exists(tt.outputFile)
					assert.NoError(t, err)
					assert.True(t, exists)
				} else {
					assert.NotEmpty(t, out)
				}
			}
		})
	}
}
