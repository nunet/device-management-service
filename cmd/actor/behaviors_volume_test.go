package actor

import (
	"context"
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/client"
	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/dms/behaviors"

	"gitlab.com/nunet/device-management-service/dms/node"
)

const (
	testVolumeName = "test_volume"
)

type mockVolumeBehaviorClient struct {
	client.DmsClient
	createVolumeFn func(ctx context.Context, req node.CreateVolumeRequest, opts ...client.Option) (node.CreateVolumeResponse, error)
	deleteVolumeFn func(ctx context.Context, req node.DeleteVolumeRequest, opts ...client.Option) (node.DeleteVolumeResponse, error)
	startVolumeFn  func(ctx context.Context, req node.StartVolumeRequest, opts ...client.Option) (node.StartVolumeResponse, error)
}

func (m *mockVolumeBehaviorClient) CreateVolume(ctx context.Context, req node.CreateVolumeRequest, opts ...client.Option) (node.CreateVolumeResponse, error) {
	return m.createVolumeFn(ctx, req, opts...)
}

func (m *mockVolumeBehaviorClient) DeleteVolume(ctx context.Context, req node.DeleteVolumeRequest, opts ...client.Option) (node.DeleteVolumeResponse, error) {
	return m.deleteVolumeFn(ctx, req, opts...)
}

func (m *mockVolumeBehaviorClient) StartVolume(ctx context.Context, req node.StartVolumeRequest, opts ...client.Option) (node.StartVolumeResponse, error) {
	return m.startVolumeFn(ctx, req, opts...)
}

func TestVolumeCreateBehavior(t *testing.T) {
	t.Parallel()

	outputDir := "output"

	tests := []struct {
		name        string
		args        []string
		opts        client.MessageOptions
		expectedReq node.CreateVolumeRequest
		testFiles   map[string]string
		wantErr     bool
	}{
		{
			name:    "no args",
			args:    []string{},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "no name",
			args: []string{
				"--client-pem-file", "test_pem",
				"--ca-output-dir", outputDir,
			},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "no client pem file",
			args: []string{
				"--name", testVolumeName,
				"--ca-output-dir", outputDir,
			},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "no ca output dir",
			args: []string{
				"--name", testVolumeName,
				"--client-pem-file", "test_pem",
			},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "non exixting pem file",
			args: []string{
				"--name", testVolumeName,
				"--client-pem-file", "test_pem",
				"--ca-output-dir", outputDir,
			},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "valid create",
			args: []string{
				"--name", testVolumeName,
				"--client-pem-file", "test_pem",
				"--ca-output-dir", outputDir,
			},
			opts: client.NewMessageOptions(),
			expectedReq: node.CreateVolumeRequest{
				Name:      testVolumeName,
				ClientPEM: "test_client_pem_data",
			},
			testFiles: map[string]string{
				"test_pem": "test_client_pem_data",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockVolumeBehaviorClient{
				createVolumeFn: func(_ context.Context, req node.CreateVolumeRequest, opts ...client.Option) (node.CreateVolumeResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					assert.Equal(t, tt.expectedReq, req)
					return node.CreateVolumeResponse{}, nil
				},
			})

			afs := afero.Afero{Fs: dmsCli.FS()}
			for k, v := range tt.testFiles {
				err := afs.WriteFile(k, []byte(v), 0o644)
				assert.NoError(t, err)
			}
			err := afs.Mkdir(outputDir, os.ModeDir)
			assert.NoError(t, err)

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err = utils.ExecuteCommand(actorCmd, append([]string{behaviors.VolumeCreateBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestVolumeDeleteBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		opts        client.MessageOptions
		expectedReq node.DeleteVolumeRequest
		wantErr     bool
	}{
		{
			name:    "no args",
			args:    []string{},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "valid name",
			args: []string{
				"--name", testVolumeName,
			},
			opts: client.NewMessageOptions(),
			expectedReq: node.DeleteVolumeRequest{
				Name: testVolumeName,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockVolumeBehaviorClient{
				deleteVolumeFn: func(_ context.Context, req node.DeleteVolumeRequest, opts ...client.Option) (node.DeleteVolumeResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					assert.Equal(t, tt.expectedReq, req)
					return node.DeleteVolumeResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.VolumeDeleteBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestVolumeStartBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		opts        client.MessageOptions
		expectedReq node.StartVolumeRequest
		wantErr     bool
	}{
		{
			name:    "no args",
			args:    []string{},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "valid name",
			args: []string{
				"--name", testVolumeName,
			},
			opts: client.NewMessageOptions(),
			expectedReq: node.StartVolumeRequest{
				Name: testVolumeName,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockVolumeBehaviorClient{
				startVolumeFn: func(_ context.Context, req node.StartVolumeRequest, opts ...client.Option) (node.StartVolumeResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					assert.Equal(t, tt.expectedReq, req)
					return node.StartVolumeResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.VolumeStartBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
