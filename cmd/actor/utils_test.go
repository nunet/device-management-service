package actor

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
)

// TestProcessEnsembleYaml tests the ProcessEnsembleYaml function using different ensemble configurations.
func TestProcessEnsembleYaml(t *testing.T) {
	t.Parallel()

	versionYaml := `version: "V1"`

	allocationYaml := `
allocations:
    alloc1:
        type: service
        executor: docker
        resources:
            cpu:
                cores: 1
            gpus: []
            ram:
                size: 1GiB
            disk:
                size: 1GiB
        execution:
            type: docker
            image: kennethreitz/httpbin
        failure_recovery: rest_for_one`

	keyYaml := `
        keys:
            - type: ssh
              file: /etc/keys1.pub`

	volumeYaml := `
        volume:
            - type: glusterfs
              src: /etc/client_private_key
              mount_destination: /etc/client_pem`
	nodeYaml := `
nodes:
    node1:
        allocations:
            - alloc1
        redundancy: 2
        failure_recovery: stay_down
        ports:
            - public: 17000
              private: 80
              allocation: alloc1`

	scriptYaml := `
scripts:
    script1: /etc/script1.sh`

	tests := []struct {
		name     string
		filePath string
		files    map[string]string
		wantErr  bool
		validate func(*testing.T, *jobtypes.EnsembleConfig)
	}{
		{
			name:     "invalid yaml format",
			filePath: "/etc/invalid.yaml",
			files: map[string]string{
				"/etc/invalid.yaml": `this is not valid yaml: ["`,
			},
			wantErr:  true,
			validate: nil,
		},
		{
			name:     "script file not found",
			filePath: "/etc/script_not_found.yaml",
			files: map[string]string{
				"/etc/script_not_found.yaml": versionYaml + allocationYaml + scriptYaml,
			},
			wantErr:  true,
			validate: nil,
		},
		{
			name:     "key file not found",
			filePath: "/etc/key_not_found.yaml",
			files: map[string]string{
				"/etc/key_not_found.yaml": versionYaml + allocationYaml + keyYaml,
			},
			wantErr:  true,
			validate: nil,
		},

		{
			name:     "valid allocation yaml",
			filePath: "/etc/valid_allocation.yaml",
			files: map[string]string{
				"/etc/valid_allocation.yaml": versionYaml + allocationYaml + volumeYaml + keyYaml + nodeYaml + scriptYaml,
				"/etc/script1.sh":            "#!/bin/bash",
				"/etc/keys1.pub":             "key1",
				"/etc/client_private_key":    "client_private_key",
				"/etc/client_pem":            "client_pem",
				"/etc/client_ca":             "client_ca",
			},
			validate: func(t *testing.T, result *jobtypes.EnsembleConfig) {
				require.NotNil(t, result)
				require.NotNil(t, result.V1)

				assert.Len(t, result.V1.Allocations, 1)
				// assert.Len(t, result.V1.Allocations["alloc1"].Volume, 1)
				assert.Len(t, result.V1.Allocations["alloc1"].Keys, 1)
				assert.Len(t, result.V1.Nodes, 1)
				assert.Len(t, result.V1.Scripts, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fs := afero.Afero{Fs: afero.NewMemMapFs()}

			if tt.files != nil {
				for filePath, content := range tt.files {
					err := fs.WriteFile(filePath, []byte(content), 0o644)
					require.NoError(t, err, "Failed to write test file")
				}
			}

			result, err := ProcessEnsembleYaml(fs, tt.filePath)

			if tt.wantErr {
				require.Error(t, err, "Expected an error but got none")
				return
			}
			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}
