package actor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/client"
	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	"gitlab.com/nunet/device-management-service/dms/orchestrator"
)

type mockSubnetBehaviorClient struct {
	client.DmsClient
	subnetCreateFn          func(ctx context.Context, req orchestrator.SubnetCreateRequest, opts ...client.Option) (orchestrator.SubnetCreateResponse, error)
	subnetDestroyFn         func(ctx context.Context, req orchestrator.SubnetDestroyRequest, opts ...client.Option) (orchestrator.SubnetDestroyResponse, error)
	subnetAddPeerFn         func(ctx context.Context, req jobs.SubnetAddPeerRequest, opts ...client.Option) (jobs.SubnetAddPeerResponse, error)
	subnetRemovePeerFn      func(ctx context.Context, req jobs.SubnetRemovePeerRequest, opts ...client.Option) (jobs.SubnetRemovePeerResponse, error)
	subnetAcceptPeerFn      func(ctx context.Context, req jobs.SubnetAcceptPeerRequest, opts ...client.Option) (jobs.SubnetAcceptPeerResponse, error)
	subnetMapPortFn         func(ctx context.Context, req jobs.SubnetMapPortRequest, opts ...client.Option) (jobs.SubnetMapPortResponse, error)
	subnetUnmapPortFn       func(ctx context.Context, req jobs.SubnetUnmapPortRequest, opts ...client.Option) (jobs.SubnetUnmapPortResponse, error)
	subnetDNSAddRecordsFn   func(ctx context.Context, req jobs.SubnetDNSAddRecordsRequest, opts ...client.Option) (jobs.SubnetDNSAddRecordsResponse, error)
	subnetDNSRemoveRecordFn func(ctx context.Context, req jobs.SubnetDNSRemoveRecordRequest, opts ...client.Option) (jobs.SubnetDNSRemoveRecordResponse, error)
}

func (m *mockSubnetBehaviorClient) SubnetCreate(ctx context.Context, req orchestrator.SubnetCreateRequest, opts ...client.Option) (orchestrator.SubnetCreateResponse, error) {
	return m.subnetCreateFn(ctx, req, opts...)
}

func (m *mockSubnetBehaviorClient) SubnetDestroy(ctx context.Context, req orchestrator.SubnetDestroyRequest, opts ...client.Option) (orchestrator.SubnetDestroyResponse, error) {
	return m.subnetDestroyFn(ctx, req, opts...)
}

func (m *mockSubnetBehaviorClient) SubnetAddPeer(ctx context.Context, req jobs.SubnetAddPeerRequest, opts ...client.Option) (jobs.SubnetAddPeerResponse, error) {
	return m.subnetAddPeerFn(ctx, req, opts...)
}

func (m *mockSubnetBehaviorClient) SubnetRemovePeer(ctx context.Context, req jobs.SubnetRemovePeerRequest, opts ...client.Option) (jobs.SubnetRemovePeerResponse, error) {
	return m.subnetRemovePeerFn(ctx, req, opts...)
}

func (m *mockSubnetBehaviorClient) SubnetAcceptPeer(ctx context.Context, req jobs.SubnetAcceptPeerRequest, opts ...client.Option) (jobs.SubnetAcceptPeerResponse, error) {
	return m.subnetAcceptPeerFn(ctx, req, opts...)
}

func (m *mockSubnetBehaviorClient) SubnetMapPort(ctx context.Context, req jobs.SubnetMapPortRequest, opts ...client.Option) (jobs.SubnetMapPortResponse, error) {
	return m.subnetMapPortFn(ctx, req, opts...)
}

func (m *mockSubnetBehaviorClient) SubnetUnmapPort(ctx context.Context, req jobs.SubnetUnmapPortRequest, opts ...client.Option) (jobs.SubnetUnmapPortResponse, error) {
	return m.subnetUnmapPortFn(ctx, req, opts...)
}

func (m *mockSubnetBehaviorClient) SubnetDNSAddRecords(ctx context.Context, req jobs.SubnetDNSAddRecordsRequest, opts ...client.Option) (jobs.SubnetDNSAddRecordsResponse, error) {
	return m.subnetDNSAddRecordsFn(ctx, req, opts...)
}

func (m *mockSubnetBehaviorClient) SubnetDNSRemoveRecord(ctx context.Context, req jobs.SubnetDNSRemoveRecordRequest, opts ...client.Option) (jobs.SubnetDNSRemoveRecordResponse, error) {
	return m.subnetDNSRemoveRecordFn(ctx, req, opts...)
}

const (
	testIP         = "10.0.0.1"
	testPeerID     = "test_id"
	testSubnetID   = "test_subnet"
	testProtocol   = "tcp"
	testSourceIP   = "10.0.0.2"
	testSourcePort = "80"
	testDestIP     = "10.0.0.3"
	testDestPort   = "8000"
	testDomainName = "example.com"
)

func TestSubnetCreateBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		opts        client.MessageOptions
		expectedReq orchestrator.SubnetCreateRequest
		wantErr     bool
	}{
		{
			name:        "no args",
			args:        []string{},
			opts:        client.NewMessageOptions(),
			expectedReq: orchestrator.SubnetCreateRequest{},
			wantErr:     true,
		},
		{
			name:    "no subnet id",
			args:    []string{"--routing-table", "0.0.0.0=test_host_id"},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "no routing table",
			args: []string{"--subnet-id", "test_subnet_id"},
			opts: client.NewMessageOptions(),
			expectedReq: orchestrator.SubnetCreateRequest{
				SubnetID: "test_subnet_id",
			},
			wantErr: false,
		},
		{
			name: "valid subnet id and routing table",
			args: []string{"--subnet-id", "test_subnet_id", "--routing-table", "0.0.0.0=test_host_id"},
			opts: client.NewMessageOptions(),
			expectedReq: orchestrator.SubnetCreateRequest{
				SubnetID: "test_subnet_id",
				RoutingTable: map[string]string{
					"0.0.0.0": "test_host_id",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockSubnetBehaviorClient{
				subnetCreateFn: func(_ context.Context, req orchestrator.SubnetCreateRequest, opts ...client.Option) (orchestrator.SubnetCreateResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					assert.Equal(t, tt.expectedReq, req)
					return orchestrator.SubnetCreateResponse{}, nil // Or tt.expectedClientResp, tt.expectedClientErr
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.SubnetCreateBehavior.Static}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestSubnetDestroyBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		opts        client.MessageOptions
		expectedReq orchestrator.SubnetDestroyRequest
		wantErr     bool
	}{
		{
			name:    "no args",
			args:    []string{},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},

		{
			name: "valid subnet-id",
			args: []string{"--subnet-id", "test_subnet"},
			opts: client.NewMessageOptions(),
			expectedReq: orchestrator.SubnetDestroyRequest{
				SubnetID: "test_subnet",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockSubnetBehaviorClient{
				subnetDestroyFn: func(_ context.Context, req orchestrator.SubnetDestroyRequest, opts ...client.Option) (orchestrator.SubnetDestroyResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					assert.Equal(t, tt.expectedReq, req)
					return orchestrator.SubnetDestroyResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.SubnetDestroyBehavior.Static}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestSubnetAddPeerBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		opts        client.MessageOptions
		expectedReq jobs.SubnetAddPeerRequest
		wantErr     bool
	}{
		{
			name:    "no args",
			args:    []string{},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name:    "no subnet id",
			args:    []string{"--peer-id", testPeerID, "--ip", testIP},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name:    "no peer id",
			args:    []string{"--subnet-id", testSubnetID, "--ip", testIP},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name:    "no ip",
			args:    []string{"--subnet-id", testSubnetID, "--peer-id", testPeerID},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "valid args",
			args: []string{"--subnet-id", testSubnetID, "--peer-id", testPeerID, "--ip", testIP},
			opts: client.NewMessageOptions(),
			expectedReq: jobs.SubnetAddPeerRequest{
				SubnetID: testSubnetID,
				PeerID:   testPeerID,
				IP:       testIP,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockSubnetBehaviorClient{
				subnetAddPeerFn: func(_ context.Context, req jobs.SubnetAddPeerRequest, opts ...client.Option) (jobs.SubnetAddPeerResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					assert.Equal(t, tt.expectedReq, req)
					return jobs.SubnetAddPeerResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.SubnetAddPeerBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestSubnetRemovePeerBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		opts        client.MessageOptions
		expectedReq jobs.SubnetRemovePeerRequest
		wantErr     bool
	}{
		{
			name:    "no args",
			args:    []string{},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name:    "no subnet id",
			args:    []string{"--peer-id", testPeerID},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name:    "no peer id",
			args:    []string{"--subnet-id", testSubnetID},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "valid args",
			args: []string{"--subnet-id", testSubnetID, "--peer-id", testPeerID},
			opts: client.NewMessageOptions(),
			expectedReq: jobs.SubnetRemovePeerRequest{
				SubnetID: testSubnetID,
				PeerID:   testPeerID,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockSubnetBehaviorClient{
				subnetRemovePeerFn: func(_ context.Context, req jobs.SubnetRemovePeerRequest, opts ...client.Option) (jobs.SubnetRemovePeerResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					assert.Equal(t, tt.expectedReq, req)
					return jobs.SubnetRemovePeerResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.SubnetRemovePeerBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestSubnetAcceptPeerBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		opts        client.MessageOptions
		expectedReq jobs.SubnetAcceptPeerRequest
		wantErr     bool
	}{
		{
			name:    "no args",
			args:    []string{},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name:    "no subnet id",
			args:    []string{"--peer-id", testPeerID, "--ip", testIP},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name:    "no peer id",
			args:    []string{"--subnet-id", testSubnetID, "--ip", testIP},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name:    "no ip",
			args:    []string{"--subnet-id", testSubnetID, "--peer-id", testPeerID},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "valid args",
			args: []string{"--subnet-id", testSubnetID, "--peer-id", testPeerID, "--ip", testIP},
			opts: client.NewMessageOptions(),
			expectedReq: jobs.SubnetAcceptPeerRequest{
				SubnetID: testSubnetID,
				PeerID:   testPeerID,
				IP:       testIP,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockSubnetBehaviorClient{
				subnetAcceptPeerFn: func(_ context.Context, req jobs.SubnetAcceptPeerRequest, opts ...client.Option) (jobs.SubnetAcceptPeerResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					assert.Equal(t, tt.expectedReq, req)
					return jobs.SubnetAcceptPeerResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.SubnetAcceptPeerBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestSubnetMapPortBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		opts        client.MessageOptions
		expectedReq jobs.SubnetMapPortRequest
		wantErr     bool
	}{
		{
			name:    "no args",
			args:    []string{},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "valid args",
			args: []string{"--subnet-id", testSubnetID, "--protocol", testProtocol, "--source-ip", testSourceIP, "--source-port", testSourcePort, "--dest-ip", testDestIP, "--dest-port", testDestPort},
			opts: client.NewMessageOptions(),
			expectedReq: jobs.SubnetMapPortRequest{
				SubnetID:   testSubnetID,
				Protocol:   testProtocol,
				SourceIP:   testSourceIP,
				SourcePort: testSourcePort,
				DestIP:     testDestIP,
				DestPort:   testDestPort,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockSubnetBehaviorClient{
				subnetMapPortFn: func(_ context.Context, req jobs.SubnetMapPortRequest, opts ...client.Option) (jobs.SubnetMapPortResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					assert.Equal(t, tt.expectedReq, req)
					return jobs.SubnetMapPortResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.SubnetMapPortBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestSubnetUnmapPortBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		opts        client.MessageOptions
		expectedReq jobs.SubnetUnmapPortRequest
		wantErr     bool
	}{
		{
			name:    "no args",
			args:    []string{},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "valid args",
			args: []string{"--subnet-id", testSubnetID, "--protocol", testProtocol, "--source-ip", testSourceIP, "--source-port", testSourcePort, "--dest-ip", testDestIP, "--dest-port", testDestPort},
			opts: client.NewMessageOptions(),
			expectedReq: jobs.SubnetUnmapPortRequest{
				SubnetID:   testSubnetID,
				Protocol:   testProtocol,
				SourceIP:   testSourceIP,
				SourcePort: testSourcePort,
				DestIP:     testDestIP,
				DestPort:   testDestPort,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockSubnetBehaviorClient{
				subnetUnmapPortFn: func(_ context.Context, req jobs.SubnetUnmapPortRequest, opts ...client.Option) (jobs.SubnetUnmapPortResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					assert.Equal(t, tt.expectedReq, req)
					return jobs.SubnetUnmapPortResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.SubnetUnmapPortBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestSubnetDNSAddRecordsBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		opts        client.MessageOptions
		expectedReq jobs.SubnetDNSAddRecordsRequest
		wantErr     bool
	}{
		{
			name:    "no args",
			args:    []string{},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "valid args",
			args: []string{"--subnet-id", testSubnetID, "--records", testDomainName + "=" + testIP},
			opts: client.NewMessageOptions(),
			expectedReq: jobs.SubnetDNSAddRecordsRequest{
				SubnetID: testSubnetID,
				Records: map[string]string{
					testDomainName: testIP,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockSubnetBehaviorClient{
				subnetDNSAddRecordsFn: func(_ context.Context, req jobs.SubnetDNSAddRecordsRequest, opts ...client.Option) (jobs.SubnetDNSAddRecordsResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					assert.Equal(t, tt.expectedReq, req)
					return jobs.SubnetDNSAddRecordsResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.SubnetDNSAddRecordsBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestSubnetDNSRemoveRecordBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		opts        client.MessageOptions
		expectedReq jobs.SubnetDNSRemoveRecordRequest
		wantErr     bool
	}{
		{
			name:    "no args",
			args:    []string{},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "valid args",
			args: []string{"--subnet-id", testSubnetID, "--domain-name", testDomainName},
			opts: client.NewMessageOptions(),
			expectedReq: jobs.SubnetDNSRemoveRecordRequest{
				SubnetID:   testSubnetID,
				DomainName: testDomainName,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockSubnetBehaviorClient{
				subnetDNSRemoveRecordFn: func(_ context.Context, req jobs.SubnetDNSRemoveRecordRequest, opts ...client.Option) (jobs.SubnetDNSRemoveRecordResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					assert.Equal(t, tt.expectedReq, req)
					return jobs.SubnetDNSRemoveRecordResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.SubnetDNSRemoveRecordBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
