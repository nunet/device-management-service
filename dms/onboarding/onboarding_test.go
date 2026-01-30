package onboarding

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

// mockResourceManager is a mock implementation of types.ResourceManager
type mockResourceManager struct {
	onboardedResources types.OnboardedResources
	getError           error
	updateError        error
}

func (m *mockResourceManager) GetOnboardedResources(_ context.Context) (types.OnboardedResources, error) {
	if m.getError != nil {
		return types.OnboardedResources{}, m.getError
	}
	return m.onboardedResources, nil
}

func (m *mockResourceManager) UpdateOnboardedResources(_ context.Context, resources types.Resources) error {
	if m.updateError != nil {
		return m.updateError
	}
	m.onboardedResources.Resources = resources
	return nil
}

// Implement other required methods with no-ops for testing
func (m *mockResourceManager) CommitResources(context.Context, types.CommittedResources) error {
	return nil
}

func (m *mockResourceManager) UncommitResources(context.Context, string) error {
	return nil
}

func (m *mockResourceManager) IsCommitted(string) (bool, error) {
	return false, nil
}

func (m *mockResourceManager) AllocateResources(context.Context, string) error {
	return nil
}

func (m *mockResourceManager) DeallocateResources(context.Context, string) error {
	return nil
}

func (m *mockResourceManager) IsAllocated(string) (bool, error) {
	return false, nil
}

func (m *mockResourceManager) GetTotalAllocation() (types.Resources, error) {
	return types.Resources{}, nil
}

func (m *mockResourceManager) GetFreeResources(context.Context) (types.FreeResources, error) {
	return types.FreeResources{}, nil
}

// mockHardwareManager is a mock implementation of types.HardwareManager
type mockHardwareManager struct {
	machineResources types.MachineResources
	freeResources    types.Resources
	getMachineError  error
	getFreeError     error
}

func (m *mockHardwareManager) GetMachineResources() (types.MachineResources, error) {
	if m.getMachineError != nil {
		return types.MachineResources{}, m.getMachineError
	}
	return m.machineResources, nil
}

func (m *mockHardwareManager) GetFreeResources() (types.Resources, error) {
	if m.getFreeError != nil {
		return types.Resources{}, m.getFreeError
	}
	return m.freeResources, nil
}

func (m *mockHardwareManager) GetUsage() (types.Resources, error) {
	return types.Resources{}, nil
}

func (m *mockHardwareManager) CheckCapacity(_ types.Resources) (bool, error) {
	return true, nil
}

func (m *mockHardwareManager) Shutdown() error {
	return nil
}

// mockConfigRepo is a mock implementation of repositories.GenericEntityRepository
type mockConfigRepo struct {
	config     types.OnboardingConfig
	getError   error
	saveError  error
	clearError error
}

func (m *mockConfigRepo) Get(_ context.Context) (types.OnboardingConfig, error) {
	if m.getError != nil {
		return types.OnboardingConfig{}, m.getError
	}

	return m.config, nil
}

func (m *mockConfigRepo) Save(_ context.Context, data types.OnboardingConfig) (types.OnboardingConfig, error) {
	if m.saveError != nil {
		return types.OnboardingConfig{}, m.saveError
	}
	m.config = data
	return m.config, nil
}

func (m *mockConfigRepo) Clear(_ context.Context) error {
	if m.clearError != nil {
		return m.clearError
	}
	m.config = types.OnboardingConfig{}
	return nil
}

func (m *mockConfigRepo) History(_ context.Context, _ repositories.Query[types.OnboardingConfig]) ([]types.OnboardingConfig, error) {
	return nil, nil
}

func (m *mockConfigRepo) GetQuery() repositories.Query[types.OnboardingConfig] {
	return repositories.Query[types.OnboardingConfig]{}
}

func TestValidateCapacity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		onboarded   types.Resources
		machine     types.Resources
		expectError bool
	}{
		{
			name: "RAM: less than 10% of total",
			onboarded: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(3)},
				CPU:  types.CPU{Cores: 2},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(20)},
			},
			machine: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(64)},
				CPU:  types.CPU{Cores: 8},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(100)},
			},
			expectError: true,
		},
		{
			name: "RAM: more than 90% of total",
			onboarded: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(60)},
				CPU:  types.CPU{Cores: 2},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(20)},
			},
			machine: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(64)},
				CPU:  types.CPU{Cores: 8},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(100)},
			},
			expectError: true,
		},
		{
			name: "RAM: within acceptable range",
			onboarded: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(32)},
				CPU:  types.CPU{Cores: 2},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(20)},
			},
			machine: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(64)},
				CPU:  types.CPU{Cores: 8},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(100)},
			},
			expectError: false,
		},
		{
			name: "CPU Cores: more than available",
			onboarded: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(32)},
				CPU:  types.CPU{Cores: 10},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(20)},
			},
			machine: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(64)},
				CPU:  types.CPU{Cores: 8},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(100)},
			},
			expectError: true,
		},
		{
			name: "CPU Cores: within available",
			onboarded: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(32)},
				CPU:  types.CPU{Cores: 6},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(20)},
			},
			machine: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(64)},
				CPU:  types.CPU{Cores: 8},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(100)},
			},
			expectError: false,
		},
		{
			name: "Disk: less than 10 GiB",
			onboarded: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(32)},
				CPU:  types.CPU{Cores: 2},
				Disk: types.Disk{Size: types.ConvertMibToBytes(50)},
			},
			machine: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(64)},
				CPU:  types.CPU{Cores: 8},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(100)},
			},
			expectError: true,
		},
		{
			name: "Disk: more than 90% of total",
			onboarded: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(32)},
				CPU:  types.CPU{Cores: 2},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(95)},
			},
			machine: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(64)},
				CPU:  types.CPU{Cores: 8},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(100)},
			},
			expectError: true,
		},
		{
			name: "Disk: within acceptable range",
			onboarded: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(32)},
				CPU:  types.CPU{Cores: 4},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(50)},
			},
			machine: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(64)},
				CPU:  types.CPU{Cores: 8},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(100)},
			},
			expectError: false,
		},
		{
			name: "GPU: machine has none",
			onboarded: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(32)},
				CPU:  types.CPU{Cores: 4},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(50)},
				GPUs: types.GPUs{
					{Model: "NVIDIA GTX 1080", VRAM: types.ConvertGiBToBytes(8)},
				},
			},
			machine: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(64)},
				CPU:  types.CPU{Cores: 8},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(100)},
			},
			expectError: true,
		},
		{
			name: "GPU: out of range vram",
			onboarded: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(32)},
				CPU:  types.CPU{Cores: 4},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(50)},
				GPUs: types.GPUs{
					{Model: "NVIDIA GTX 1080", VRAM: types.ConvertGiBToBytes(8)},
				},
			},
			machine: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(64)},
				CPU:  types.CPU{Cores: 8},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(100)},
				GPUs: types.GPUs{
					{Model: "NVIDIA GTX 1080", VRAM: types.ConvertGiBToBytes(4)},
				},
			},
			expectError: true,
		},
		{
			name: "GPU: in range vram",
			onboarded: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(32)},
				CPU:  types.CPU{Cores: 4},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(50)},
				GPUs: types.GPUs{
					{Model: "NVIDIA GTX 1080", VRAM: types.ConvertGiBToBytes(8)},
				},
			},
			machine: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(64)},
				CPU:  types.CPU{Cores: 8},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(100)},
				GPUs: types.GPUs{
					{Model: "NVIDIA GTX 1080", VRAM: types.ConvertGiBToBytes(16)},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateCapacity(tt.onboarded, tt.machine)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateUsage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		onboarded   types.Resources
		free        types.Resources
		expectError bool
	}{
		{
			name: "RAM: high usage - less ram available than onboarded",
			onboarded: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(16)},
				CPU:  types.CPU{Cores: 4, ClockSpeed: 2000},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(50)},
			},
			free: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(12)},
				CPU:  types.CPU{Cores: 8, ClockSpeed: 2000},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(10)},
			},
			expectError: true,
		},
		{
			name: "CPU: high usage - less cpu available than onboarded",
			onboarded: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(16)},
				CPU:  types.CPU{Cores: 8, ClockSpeed: 2000},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(50)},
			},
			free: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(32)},
				CPU:  types.CPU{Cores: 6, ClockSpeed: 2000},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(100)},
			},
			expectError: true,
		},
		{
			name: "Disk: high usage - less disk available than onboarded",
			onboarded: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(16)},
				CPU:  types.CPU{Cores: 4, ClockSpeed: 2000},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(50)},
			},
			free: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(32)},
				CPU:  types.CPU{Cores: 8, ClockSpeed: 2000},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(30)},
			},
			expectError: true,
		},
		{
			name: "GPU: high usage - less vram available than onboarded",
			onboarded: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(16)},
				CPU:  types.CPU{Cores: 4, ClockSpeed: 2000},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(50)},
				GPUs: types.GPUs{
					{Model: "NVIDIA GTX 1080", VRAM: types.ConvertGiBToBytes(8)},
				},
			},
			free: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(32)},
				CPU:  types.CPU{Cores: 6, ClockSpeed: 2000},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(100)},
				GPUs: types.GPUs{
					{Model: "NVIDIA GTX 1080", VRAM: types.ConvertGiBToBytes(4)},
				},
			},
			expectError: true,
		},
		{
			name: "GPU: onboarded gpu not found",
			onboarded: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(16)},
				CPU:  types.CPU{Cores: 4, ClockSpeed: 2000},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(50)},
				GPUs: types.GPUs{
					{Model: "NVIDIA GTX 1080", VRAM: types.ConvertGiBToBytes(8)},
				},
			},
			free: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(32)},
				CPU:  types.CPU{Cores: 6, ClockSpeed: 2000},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(100)},
			},
			expectError: true,
		},
		{
			name: "onboarded within free limits",
			onboarded: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(16)},
				CPU:  types.CPU{Cores: 4, ClockSpeed: 2000},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(50)},
			},
			free: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(32)},
				CPU:  types.CPU{Cores: 8, ClockSpeed: 2000},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(100)},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateUsage(tt.onboarded, tt.free)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		resourceMgr types.ResourceManager
		hardwareMgr types.HardwareManager
		configRepo  repositories.GenericEntityRepository[types.OnboardingConfig]
		wantErr     bool
		errContains string
	}{
		{
			name:        "nil resource manager",
			resourceMgr: nil,
			hardwareMgr: &mockHardwareManager{},
			configRepo:  &mockConfigRepo{},
			wantErr:     true,
			errContains: "resource manager is required",
		},
		{
			name:        "nil hardware manager",
			resourceMgr: &mockResourceManager{},
			hardwareMgr: nil,
			configRepo:  &mockConfigRepo{},
			wantErr:     true,
			errContains: "hardware manager is required",
		},
		{
			name:        "nil config repo",
			resourceMgr: &mockResourceManager{},
			hardwareMgr: &mockHardwareManager{},
			configRepo:  nil,
			wantErr:     true,
			errContains: "config repo is required",
		},
		{
			name:        "config repo get error",
			resourceMgr: &mockResourceManager{},
			hardwareMgr: &mockHardwareManager{},
			configRepo: &mockConfigRepo{
				getError: errors.New("database connection failed"),
			},
			wantErr:     true,
			errContains: "could not get onboarding config",
		},
		{
			name:        "config not found - creates empty config",
			resourceMgr: &mockResourceManager{},
			hardwareMgr: &mockHardwareManager{},
			configRepo: &mockConfigRepo{
				getError: repositories.ErrNotFound,
			},
			wantErr: false,
		},
		{
			name: "onboarded",
			resourceMgr: &mockResourceManager{
				onboardedResources: types.OnboardedResources{
					Resources: types.Resources{
						CPU:  types.CPU{Cores: 6, ClockSpeed: 2000},
						RAM:  types.RAM{Size: types.ConvertGiBToBytes(8)},
						Disk: types.Disk{Size: types.ConvertGiBToBytes(50)},
					},
				},
			},
			hardwareMgr: &mockHardwareManager{
				machineResources: types.MachineResources{
					Resources: types.Resources{
						CPU:  types.CPU{Cores: 8},
						RAM:  types.RAM{Size: types.ConvertGiBToBytes(16)},
						Disk: types.Disk{Size: types.ConvertGiBToBytes(100)},
					},
				},
				freeResources: types.Resources{
					CPU:  types.CPU{Cores: 6},
					RAM:  types.RAM{Size: types.ConvertGiBToBytes(14)},
					Disk: types.Disk{Size: types.ConvertGiBToBytes(80)},
				},
			},
			configRepo: &mockConfigRepo{
				config: types.OnboardingConfig{
					IsOnboarded: true,
				},
			},
			wantErr: false,
		},
		{
			name:        "successful creation",
			resourceMgr: &mockResourceManager{},
			hardwareMgr: &mockHardwareManager{},
			configRepo:  &mockConfigRepo{},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			onboarding, err := New(ctx, tt.resourceMgr, tt.hardwareMgr, tt.configRepo)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.ErrorContains(t, err, tt.errContains)
				}
				require.Nil(t, onboarding)
			} else {
				require.NoError(t, err)
				require.NotNil(t, onboarding)
			}
		})
	}
}

func TestOnboarding_Onboard(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		resourceMgr      *mockResourceManager
		hardwareMgr      *mockHardwareManager
		configRepo       *mockConfigRepo
		onboardResources types.Resources
		expectError      bool
	}{
		{
			name: "fail with beyond available CPU cores",
			hardwareMgr: &mockHardwareManager{
				machineResources: types.MachineResources{
					Resources: types.Resources{
						RAM:  types.RAM{Size: types.ConvertGiBToBytes(32)},
						CPU:  types.CPU{Cores: 2},
						Disk: types.Disk{Size: types.ConvertGiBToBytes(100)},
					},
				},
			},
			configRepo: &mockConfigRepo{},
			onboardResources: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(16)},
				CPU:  types.CPU{Cores: 4},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(50)},
			},
			expectError: true,
		},
		{
			name: "fail with beyond available RAM",
			hardwareMgr: &mockHardwareManager{
				machineResources: types.MachineResources{
					Resources: types.Resources{
						RAM:  types.RAM{Size: types.ConvertGiBToBytes(32)},
						CPU:  types.CPU{Cores: 8},
						Disk: types.Disk{Size: types.ConvertGiBToBytes(100)},
					},
				},
			},
			configRepo: &mockConfigRepo{},
			onboardResources: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(64)},
				CPU:  types.CPU{Cores: 4},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(50)},
			},
			expectError: true,
		},
		{
			name:        "successful onboard",
			resourceMgr: &mockResourceManager{},
			hardwareMgr: &mockHardwareManager{
				machineResources: types.MachineResources{
					Resources: types.Resources{
						RAM:  types.RAM{Size: types.ConvertGiBToBytes(32)},
						CPU:  types.CPU{Cores: 8},
						Disk: types.Disk{Size: types.ConvertGiBToBytes(100)},
					},
				},
				freeResources: types.Resources{
					RAM:  types.RAM{Size: types.ConvertGiBToBytes(20)},
					CPU:  types.CPU{Cores: 6},
					Disk: types.Disk{Size: types.ConvertGiBToBytes(80)},
				},
			},
			configRepo: &mockConfigRepo{},
			onboardResources: types.Resources{
				RAM:  types.RAM{Size: types.ConvertGiBToBytes(16)},
				CPU:  types.CPU{Cores: 4},
				Disk: types.Disk{Size: types.ConvertGiBToBytes(50)},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			onboardingManager, err := New(ctx, tt.resourceMgr, tt.hardwareMgr, tt.configRepo)
			require.NoError(t, err)

			oc, err := onboardingManager.Onboard(
				ctx,
				types.OnboardingConfig{
					OnboardedResources: tt.onboardResources,
				},
			)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.onboardResources.CPU.Cores, oc.OnboardedResources.CPU.Cores)
				assert.True(t, onboardingManager.IsOnboarded())
				info, err := onboardingManager.Info(ctx)
				require.NoError(t, err)
				assert.True(t, info.IsOnboarded)
				assert.Equal(t, tt.onboardResources.RAM.Size, info.OnboardedResources.RAM.Size)
			}
		})
	}
}

func TestOnboarding_Offboard(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		resourceMgr *mockResourceManager
		hardwareMgr *mockHardwareManager
		configRepo  *mockConfigRepo
		expectError bool
	}{
		{
			name: "fail: not onboarded",
			resourceMgr: &mockResourceManager{
				onboardedResources: types.OnboardedResources{},
			},
			hardwareMgr: &mockHardwareManager{
				machineResources: types.MachineResources{
					Resources: types.Resources{
						RAM:  types.RAM{Size: types.ConvertGiBToBytes(32)},
						CPU:  types.CPU{Cores: 2},
						Disk: types.Disk{Size: types.ConvertGiBToBytes(100)},
					},
				},
			},
			configRepo: &mockConfigRepo{
				config: types.OnboardingConfig{
					IsOnboarded: false,
				},
			},
			expectError: true,
		},
		{
			name: "success",
			resourceMgr: &mockResourceManager{
				onboardedResources: types.OnboardedResources{
					Resources: types.Resources{
						RAM:  types.RAM{Size: types.ConvertGiBToBytes(16)},
						CPU:  types.CPU{Cores: 4},
						Disk: types.Disk{Size: types.ConvertGiBToBytes(50)},
					},
				},
			},
			hardwareMgr: &mockHardwareManager{
				machineResources: types.MachineResources{
					Resources: types.Resources{
						RAM:  types.RAM{Size: types.ConvertGiBToBytes(32)},
						CPU:  types.CPU{Cores: 8},
						Disk: types.Disk{Size: types.ConvertGiBToBytes(100)},
					},
				},
			},
			configRepo: &mockConfigRepo{
				config: types.OnboardingConfig{
					IsOnboarded: true,
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			onboardingManager, err := New(ctx, tt.resourceMgr, tt.hardwareMgr, tt.configRepo)
			require.NoError(t, err)

			err = onboardingManager.Offboard(ctx)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
