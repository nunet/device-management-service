package actor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/client"
	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/types"

	"gitlab.com/nunet/device-management-service/dms/node"
)

type mockOnboardingBehaviorClient struct {
	client.DmsClient
	onboardFn       func(ctx context.Context, req node.OnboardRequest, opts ...client.Option) (node.OnboardResponse, error)
	offboardFn      func(ctx context.Context, req node.OffboardRequest, opts ...client.Option) (node.OffboardResponse, error)
	onboardStatusFn func(ctx context.Context, opts ...client.Option) (node.OnboardStatusResponse, error)

	hardwareSpecFn  func(ctx context.Context, opts ...client.Option) (node.ResourcesResponse, error)
	hardwareUsageFn func(ctx context.Context, opts ...client.Option) (node.ResourcesResponse, error)
}

func (m *mockOnboardingBehaviorClient) Onboard(ctx context.Context, req node.OnboardRequest, opts ...client.Option) (node.OnboardResponse, error) {
	return m.onboardFn(ctx, req, opts...)
}

func (m *mockOnboardingBehaviorClient) Offboard(ctx context.Context, req node.OffboardRequest, opts ...client.Option) (node.OffboardResponse, error) {
	return m.offboardFn(ctx, req, opts...)
}

func (m *mockOnboardingBehaviorClient) OnboardStatus(ctx context.Context, opts ...client.Option) (node.OnboardStatusResponse, error) {
	return m.onboardStatusFn(ctx, opts...)
}

func (m *mockOnboardingBehaviorClient) HardwareSpec(ctx context.Context, opts ...client.Option) (node.ResourcesResponse, error) {
	return m.hardwareSpecFn(ctx, opts...)
}

func (m *mockOnboardingBehaviorClient) HardwareUsage(ctx context.Context, opts ...client.Option) (node.ResourcesResponse, error) {
	return m.hardwareUsageFn(ctx, opts...)
}

func newResourceResponse(cores float32, ram, disk uint64, gpus map[types.GPUVendor]uint64) types.Resources {
	var gpuList types.GPUs
	models := map[types.GPUVendor]string{
		types.GPUVendorNvidia: "NVIDIA GeForce RTX 3080",
		types.GPUVendorAMDATI: "AMD Radeon RX 6800",
	}
	for vendor, vram := range gpus {
		gpuList = append(gpuList, types.GPU{
			Vendor: vendor,
			Model:  models[vendor],
			VRAM:   types.ConvertGBToBytes(vram),
		})
	}
	clockSpeed := float64(2e9)
	return types.Resources{
		CPU: types.CPU{
			Cores:      cores,
			ClockSpeed: clockSpeed,
		},
		RAM: types.RAM{
			Size: types.ConvertGBToBytes(ram),
		},
		Disk: types.Disk{
			Size: types.ConvertGBToBytes(disk),
		},
		GPUs: gpuList,
	}
}

func TestOnboardBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		opts        client.MessageOptions
		input       [][]byte
		expectedReq node.OnboardRequest
		wantErr     bool
	}{
		{
			name:    "no args",
			args:    []string{},
			opts:    client.NewMessageOptions(),
			wantErr: true,
		},
		{
			name: "no gpu args",
			args: []string{"--cpu", "2", "--ram", "4G", "--disk", "10G", "-N"},
			opts: client.NewMessageOptions(),
			expectedReq: node.OnboardRequest{
				NoGPU: true,
				Config: types.OnboardingConfig{
					OnboardedResources: newResourceResponse(2, 4, 10, map[types.GPUVendor]uint64{}),
				},
			},
			wantErr: false,
		},
		{
			name: "gpus arg",
			args: []string{"--cpu", "2", "--ram", "4G", "--disk", "10G", "--gpus", "0:4G"},
			opts: client.NewMessageOptions(),
			expectedReq: node.OnboardRequest{
				NoGPU: false,
				Config: types.OnboardingConfig{
					OnboardedResources: newResourceResponse(2, 4, 10, map[types.GPUVendor]uint64{
						types.GPUVendorNvidia: 4,
					}),
				},
			},
			wantErr: false,
		},
		{
			name: "with args",
			args: []string{"--cpu", "2", "--ram", "4G", "--disk", "10G"},
			opts: client.NewMessageOptions(),
			expectedReq: node.OnboardRequest{
				NoGPU: false,
				Config: types.OnboardingConfig{
					OnboardedResources: newResourceResponse(2, 4, 10, map[types.GPUVendor]uint64{
						types.GPUVendorNvidia: 4,
					}),
				},
			},
			// Down arrow and enter (2x) to select the first GPU then enter vram
			input:   [][]byte{{14, 13}, {13}, {'4', 13}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockOnboardingBehaviorClient{
				onboardFn: func(_ context.Context, req node.OnboardRequest, opts ...client.Option) (node.OnboardResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					assert.Equal(t, tt.expectedReq, req)
					return node.OnboardResponse{}, nil
				},
				hardwareSpecFn: func(_ context.Context, opts ...client.Option) (node.ResourcesResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					return node.ResourcesResponse{
						OK: true,
						Resources: newResourceResponse(8, 16, 100, map[types.GPUVendor]uint64{
							types.GPUVendorNvidia: 16,
						}),
					}, nil
				},
				hardwareUsageFn: func(_ context.Context, opts ...client.Option) (node.ResourcesResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					return node.ResourcesResponse{
						OK: true,
						Resources: newResourceResponse(2, 4, 50, map[types.GPUVendor]uint64{
							types.GPUVendorNvidia: 2,
						}),
					}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommandWithInput(actorCmd, tt.input, append([]string{behaviors.OnboardBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestOffboardBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		opts    client.MessageOptions
		wantErr bool
	}{
		{
			name:    "no args",
			args:    []string{},
			opts:    client.NewMessageOptions(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockOnboardingBehaviorClient{
				offboardFn: func(_ context.Context, _ node.OffboardRequest, opts ...client.Option) (node.OffboardResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					return node.OffboardResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.OffboardBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestOnboardStatusBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		opts    client.MessageOptions
		wantErr bool
	}{
		{
			name:    "no args",
			args:    []string{},
			opts:    client.NewMessageOptions(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dmsCli := setupTest(t, &mockOnboardingBehaviorClient{
				onboardStatusFn: func(_ context.Context, opts ...client.Option) (node.OnboardStatusResponse, error) {
					checkMessageOptions(t, tt.opts, opts...)
					return node.OnboardStatusResponse{}, nil
				},
			})

			actorCmd := newActorCmdGroup(dmsCli)
			_, _, err := utils.ExecuteCommand(actorCmd, append([]string{behaviors.OnboardStatusBehavior}, tt.args...)...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
