package actor

import (
	"context"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/client"
	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	dmstypes "gitlab.com/nunet/device-management-service/types"

	"gitlab.com/nunet/device-management-service/dms/node"
)

type mockDeploymentBehaviorClient struct {
	client.DmsClient
	deploymentListFn     func(ctx context.Context, req node.DeploymentListRequest, opts ...client.Option) (node.DeploymentListResponse, error)
	deploymentStatusFn   func(ctx context.Context, req node.DeploymentStatusRequest, opts ...client.Option) (node.DeploymentStatusResponse, error)
	deploymentLogsFn     func(ctx context.Context, req node.DeploymentLogsRequest, opts ...client.Option) (node.DeploymentLogsResponse, error)
	deploymentManifestFn func(ctx context.Context, req node.DeploymentManifestRequest, opts ...client.Option) (node.DeploymentManifestResponse, error)
	deploymentShutdownFn func(ctx context.Context, req node.DeploymentShutdownRequest, opts ...client.Option) (node.DeploymentShutdownResponse, error)
	deploymentNewFn      func(ctx context.Context, req node.NewDeploymentRequest, opts ...client.Option) (node.NewDeploymentResponse, error)
	deploymentUpdateFn   func(ctx context.Context, req node.UpdateDeploymentRequest, opts ...client.Option) (node.UpdateDeploymentResponse, error)
}

func (m *mockDeploymentBehaviorClient) DeploymentList(ctx context.Context, req node.DeploymentListRequest, opts ...client.Option) (node.DeploymentListResponse, error) {
	return m.deploymentListFn(ctx, req, opts...)
}

func (m *mockDeploymentBehaviorClient) DeploymentStatus(ctx context.Context, req node.DeploymentStatusRequest, opts ...client.Option) (node.DeploymentStatusResponse, error) {
	return m.deploymentStatusFn(ctx, req, opts...)
}

func (m *mockDeploymentBehaviorClient) DeploymentLogs(ctx context.Context, req node.DeploymentLogsRequest, opts ...client.Option) (node.DeploymentLogsResponse, error) {
	return m.deploymentLogsFn(ctx, req, opts...)
}

func (m *mockDeploymentBehaviorClient) DeploymentManifest(ctx context.Context, req node.DeploymentManifestRequest, opts ...client.Option) (node.DeploymentManifestResponse, error) {
	return m.deploymentManifestFn(ctx, req, opts...)
}

func (m *mockDeploymentBehaviorClient) DeploymentShutdown(ctx context.Context, req node.DeploymentShutdownRequest, opts ...client.Option) (node.DeploymentShutdownResponse, error) {
	return m.deploymentShutdownFn(ctx, req, opts...)
}

func (m *mockDeploymentBehaviorClient) DeploymentNew(ctx context.Context, req node.NewDeploymentRequest, opts ...client.Option) (node.NewDeploymentResponse, error) {
	return m.deploymentNewFn(ctx, req, opts...)
}

func (m *mockDeploymentBehaviorClient) DeploymentUpdate(ctx context.Context, req node.UpdateDeploymentRequest, opts ...client.Option) (node.UpdateDeploymentResponse, error) {
	return m.deploymentUpdateFn(ctx, req, opts...)
}

func TestDeploymentListBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		opts        client.MessageOptions
		expectedReq node.DeploymentListRequest
		wantErr     bool
	}{
		{
			name:        "no args",
			args:        []string{},
			opts:        client.NewMessageOptions(),
			expectedReq: node.DeploymentListRequest{},
			wantErr:     false,
		},
		{
			name: "valid filter",
			args: []string{"--filter", "namespace=test_namespace"},
			opts: client.NewMessageOptions(),
			expectedReq: node.DeploymentListRequest{
				Metadata: map[string]string{
					"namespace": "test_namespace",
				},
			},
			wantErr: false,
		},
		{
			name: "multiple filters",
			args: []string{"--filter", "namespace=test_namespace", "--filter", "name=test_name"},
			opts: client.NewMessageOptions(),
			expectedReq: node.DeploymentListRequest{
				Metadata: map[string]string{
					"namespace": "test_namespace",
					"name":      "test_name",
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid filter",
			args:    []string{"--filter", "invalid_filter"},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockDeploymentBehaviorClient{
				deploymentListFn: func(_ context.Context, req node.DeploymentListRequest, opts ...client.Option) (node.DeploymentListResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					assert.Equal(t, tt.expectedReq, req)
					return node.DeploymentListResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.DeploymentListBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestDeploymentStatusBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		opts        client.MessageOptions
		expectedReq node.DeploymentStatusRequest
		wantErr     bool
	}{
		{
			name:    "no args",
			args:    []string{},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "valid id",
			args: []string{"--id", "test_id"},
			expectedReq: node.DeploymentStatusRequest{
				ID: "test_id",
			},
			opts:    client.NewMessageOptions(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockDeploymentBehaviorClient{
				deploymentStatusFn: func(_ context.Context, req node.DeploymentStatusRequest, opts ...client.Option) (node.DeploymentStatusResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					assert.Equal(t, tt.expectedReq, req)
					return node.DeploymentStatusResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.DeploymentStatusBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestDeploymentLogsBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		opts        client.MessageOptions
		expectedReq node.DeploymentLogsRequest
		wantErr     bool
	}{
		{
			name:    "no args",
			args:    []string{},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name:    "no id",
			args:    []string{"--allocation", "test_allocation"},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name:    "no allocation",
			args:    []string{"--id", "test_id"},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "valid id and allocation",
			args: []string{"--id", "test_id", "--allocation", "test_allocation"},
			opts: client.NewMessageOptions(),
			expectedReq: node.DeploymentLogsRequest{
				EnsembleID:     "test_id",
				AllocationName: "test_allocation",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockDeploymentBehaviorClient{
				deploymentLogsFn: func(_ context.Context, req node.DeploymentLogsRequest, opts ...client.Option) (node.DeploymentLogsResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					assert.Equal(t, tt.expectedReq, req)
					return node.DeploymentLogsResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.DeploymentLogsBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestDeploymentManifestBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		opts        client.MessageOptions
		expectedReq node.DeploymentManifestRequest
		wantErr     bool
	}{
		{
			name:    "no args",
			args:    []string{},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "valid id",
			args: []string{"--id", "test_id"},
			opts: client.NewMessageOptions(),
			expectedReq: node.DeploymentManifestRequest{
				ID: "test_id",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockDeploymentBehaviorClient{
				deploymentManifestFn: func(_ context.Context, req node.DeploymentManifestRequest, opts ...client.Option) (node.DeploymentManifestResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					assert.Equal(t, tt.expectedReq, req)
					return node.DeploymentManifestResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.DeploymentManifestBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestDeploymentShutdownBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		opts        client.MessageOptions
		expectedReq node.DeploymentShutdownRequest
		wantErr     bool
	}{
		{
			name:    "no args",
			args:    []string{},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "valid id",
			args: []string{"--id", "test_id"},
			opts: client.NewMessageOptions(),
			expectedReq: node.DeploymentShutdownRequest{
				ID: "test_id",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockDeploymentBehaviorClient{
				deploymentShutdownFn: func(_ context.Context, req node.DeploymentShutdownRequest, opts ...client.Option) (node.DeploymentShutdownResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					assert.Equal(t, tt.expectedReq, req)
					return node.DeploymentShutdownResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.DeploymentShutdownBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestNewDeploymentBehavior(t *testing.T) {
	t.Parallel()

	testEnsembleConfig := `version: "V1"

allocations:
    alloc1:
        type: task
        executor: docker
        execution:
            type: docker
            image: hello-world
        resources:
          cpu:
                cores: 1
          ram:
              size: 1000000B
          disk:
              size: 1000000B
        failure_recovery: stay_down`

	testEnsemble := jobtypes.EnsembleConfig{
		V1: &jobtypes.EnsembleConfigV1{
			Allocations: map[string]jobtypes.AllocationConfig{
				"alloc1": {
					Type:     jobtypes.AllocationTypeTask,
					Executor: jobtypes.ExecutorDocker,
					Resources: dmstypes.Resources{
						CPU: dmstypes.CPU{
							Cores: 1,
						},
						RAM: dmstypes.RAM{
							Size: 1000000,
						},
						Disk: dmstypes.Disk{
							Size: 1000000,
						},
					},
					Execution: dmstypes.SpecConfig{
						Type: string(jobtypes.ExecutorDocker),
						Params: map[string]interface{}{
							"image": "hello-world",
						},
					},
					DNSName:         "alloc1",
					FailureRecovery: jobtypes.AllocationFailureRecoveryStayDown,
				},
			},
		},
	}

	tests := []struct {
		name        string
		args        []string
		opts        client.MessageOptions
		expectedReq node.NewDeploymentRequest
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
			name: "no args with default ensemble file",
			args: []string{},
			opts: client.NewMessageOptions(),
			testFiles: map[string]string{
				"ensemble.yaml": testEnsembleConfig,
			},
			expectedReq: node.NewDeploymentRequest{
				Ensemble: testEnsemble,
			},
			wantErr: false,
		},
		{
			name: "valid spec file",
			args: []string{"--spec-file", "test_spec_file.yaml"},
			opts: client.NewMessageOptions(),
			testFiles: map[string]string{
				"test_spec_file.yaml": testEnsembleConfig,
			},
			expectedReq: node.NewDeploymentRequest{
				Ensemble: testEnsemble,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockDeploymentBehaviorClient{
				deploymentNewFn: func(_ context.Context, req node.NewDeploymentRequest, opts ...client.Option) (node.NewDeploymentResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					assert.Equal(t, tt.expectedReq, req)
					return node.NewDeploymentResponse{}, nil
				},
			})

			afs := afero.Afero{Fs: dmsCli.FS()}
			for k, v := range tt.testFiles {
				err := afs.WriteFile(k, []byte(v), 0o644)
				assert.NoError(t, err)
			}

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.NewDeploymentBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestUpdateDeploymentBehavior(t *testing.T) {
	t.Parallel()

	testEnsembleConfig := `version: "V1"

allocations:
    alloc1:
        type: task
        executor: docker
        execution:
            type: docker
            image: hello-world
        resources:
          cpu:
                cores: 1
          ram:
              size: 1000000B
          disk:
              size: 1000000B
        failure_recovery: stay_down`

	testEnsemble := jobtypes.EnsembleConfig{
		V1: &jobtypes.EnsembleConfigV1{
			Allocations: map[string]jobtypes.AllocationConfig{
				"alloc1": {
					Type:     jobtypes.AllocationTypeTask,
					Executor: jobtypes.ExecutorDocker,
					Resources: dmstypes.Resources{
						CPU: dmstypes.CPU{
							Cores: 1,
						},
						RAM: dmstypes.RAM{
							Size: 1000000,
						},
						Disk: dmstypes.Disk{
							Size: 1000000,
						},
					},
					Execution: dmstypes.SpecConfig{
						Type: string(jobtypes.ExecutorDocker),
						Params: map[string]interface{}{
							"image": "hello-world",
						},
					},
					DNSName:         "alloc1",
					FailureRecovery: jobtypes.AllocationFailureRecoveryStayDown,
				},
			},
		},
	}

	tests := []struct {
		name        string
		args        []string
		opts        client.MessageOptions
		expectedReq node.UpdateDeploymentRequest
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
			name:    "no ensemble ID",
			args:    []string{"--spec-file", "test_spec_file.yaml"},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "with default ensemble file",
			args: []string{"--id", "test-id"},
			opts: client.NewMessageOptions(),
			testFiles: map[string]string{
				"ensemble.yaml": testEnsembleConfig,
			},
			expectedReq: node.UpdateDeploymentRequest{
				EnsembleID: "test-id",
				Ensemble:   testEnsemble,
			},
			wantErr: false,
		},
		{
			name: "valid spec file",
			args: []string{"--id", "test-id", "--spec-file", "test_spec_file.yaml"},
			opts: client.NewMessageOptions(),
			testFiles: map[string]string{
				"test_spec_file.yaml": testEnsembleConfig,
			},
			expectedReq: node.UpdateDeploymentRequest{
				EnsembleID: "test-id",
				Ensemble:   testEnsemble,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockDeploymentBehaviorClient{
				deploymentUpdateFn: func(_ context.Context, req node.UpdateDeploymentRequest, opts ...client.Option) (node.UpdateDeploymentResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					assert.Equal(t, tt.expectedReq, req)
					return node.UpdateDeploymentResponse{}, nil
				},
			})

			afs := afero.Afero{Fs: dmsCli.FS()}
			for k, v := range tt.testFiles {
				err := afs.WriteFile(k, []byte(v), 0o644)
				assert.NoError(t, err)
			}

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.DeploymentUpdateBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
