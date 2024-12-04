// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package onboarding

import (
	"context"
	"fmt"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/types"
)

func Test_NewOnboarding(t *testing.T) {
	t.Parallel()

	t.Run("must be able to create a new onboarding instance", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resourceManager := NewMockResourceManager(ctrl)
		hardwareManager := NewMockHardwareManager(ctrl)
		configRepo := NewMockGenericEntityRepository[types.OnboardingConfig](ctrl)

		configRepo.EXPECT().Get(context.Background()).Return(types.OnboardingConfig{}, nil)
		onboarding, err := New(context.Background(), resourceManager, hardwareManager, configRepo)
		require.NoError(t, err)
		require.NotNil(t, onboarding)
	})

	t.Run("must return an error if resource manager is nil", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		hardwareManager := NewMockHardwareManager(ctrl)
		configRepo := NewMockGenericEntityRepository[types.OnboardingConfig](ctrl)

		onboarding, err := New(context.Background(), nil, hardwareManager, configRepo)
		require.Error(t, err)
		require.Nil(t, onboarding)
	})

	t.Run("must return an error if hardware manager is nil", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resourceManager := NewMockResourceManager(ctrl)
		configRepo := NewMockGenericEntityRepository[types.OnboardingConfig](ctrl)

		onboarding, err := New(context.Background(), resourceManager, nil, configRepo)
		require.Error(t, err)
		require.Nil(t, onboarding)
	})

	t.Run("must return an error if config repo is nil", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resourceManager := NewMockResourceManager(ctrl)
		hardwareManager := NewMockHardwareManager(ctrl)

		onboarding, err := New(context.Background(), resourceManager, hardwareManager, nil)
		require.Error(t, err)
		require.Nil(t, onboarding)
	})

	t.Run("must load the config from the repository", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resourceManager := NewMockResourceManager(ctrl)
		hardwareManager := NewMockHardwareManager(ctrl)
		configRepo := NewMockGenericEntityRepository[types.OnboardingConfig](ctrl)

		configRepo.EXPECT().Get(context.Background()).Return(types.OnboardingConfig{}, fmt.Errorf("db error"))
		_, err := New(context.Background(), resourceManager, hardwareManager, configRepo)
		require.Error(t, err)
	})

	t.Run("must offboard the machine if the prerequisites doesn't meet", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resourceManager := NewMockResourceManager(ctrl)
		hardwareManager := NewMockHardwareManager(ctrl)
		configRepo := NewMockGenericEntityRepository[types.OnboardingConfig](ctrl)

		configRepo.EXPECT().Get(context.Background()).Return(types.OnboardingConfig{
			IsOnboarded: true,
		}, nil)
		resourceManager.EXPECT().GetOnboardedResources(gomock.Any()).Return(types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      10,
					ClockSpeed: 24000,
				},
				RAM: types.RAM{
					Size: 8000000000,
				},
				Disk: types.Disk{
					Size: 100000000000,
				},
			},
		}, nil)
		hardwareManager.EXPECT().GetMachineResources().Return(types.MachineResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      2,
					ClockSpeed: 24000,
				},
				RAM: types.RAM{
					Size: 16000000000,
				},
				Disk: types.Disk{
					Size: 100000000000,
				},
			},
		}, nil)
		configRepo.EXPECT().Clear(gomock.Any()).Return(nil)
		resourceManager.EXPECT().UpdateOnboardedResources(gomock.Any(), types.Resources{}).Return(nil)
		onboarding, err := New(context.Background(), resourceManager, hardwareManager, configRepo)
		require.NoError(t, err)
		require.NotNil(t, onboarding)

		// Ensure the machine is offboarded
		isOnboarded, err := onboarding.IsOnboarded()
		require.NoError(t, err)
		require.False(t, isOnboarded)
	})
}

func TestOnboarding(t *testing.T) {
	t.Parallel()

	t.Run("must be able to onboard a machine", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resourceManager := NewMockResourceManager(ctrl)
		hardwareManager := NewMockHardwareManager(ctrl)
		configRepo := NewMockGenericEntityRepository[types.OnboardingConfig](ctrl)

		configRepo.EXPECT().Get(context.Background()).Return(types.OnboardingConfig{}, nil)
		onboarding, err := New(context.Background(), resourceManager, hardwareManager, configRepo)
		require.NoError(t, err)
		require.NotNil(t, onboarding)

		config := types.OnboardingConfig{
			IsOnboarded: false,
		}
		machineResources := types.MachineResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      4,
					ClockSpeed: 24000,
				},
				RAM: types.RAM{
					Size: 16000000000,
				},
				Disk: types.Disk{
					Size: 100000000000,
				},
				GPUs: []types.GPU{
					{
						Index: 0,
						VRAM:  16000000000,
					},
				},
			},
		}
		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      2,
					ClockSpeed: 24000,
				},
				RAM: types.RAM{
					Size: 8000000000,
				},
				Disk: types.Disk{
					Size: 50000000000,
				},
				GPUs: []types.GPU{
					{
						Index: 0,
						VRAM:  8000000000,
					},
				},
			},
		}
		config.OnboardedResources = onboardedResources.Resources

		configRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(config, nil)
		hardwareManager.EXPECT().GetMachineResources().Return(machineResources, nil)
		hardwareManager.EXPECT().GetFreeResources().Return(machineResources.Resources, nil)
		resourceManager.EXPECT().UpdateOnboardedResources(gomock.Any(), onboardedResources.Resources).Return(nil)
		_, err = onboarding.Onboard(context.Background(), config)
		require.NoError(t, err)

		isOnboarded, err := onboarding.IsOnboarded()
		require.NoError(t, err)
		require.True(t, isOnboarded)
	})

	t.Run("must be able to get onboarding info", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resourceManager := NewMockResourceManager(ctrl)
		hardwareManager := NewMockHardwareManager(ctrl)
		configRepo := NewMockGenericEntityRepository[types.OnboardingConfig](ctrl)

		configRepo.EXPECT().Get(context.Background()).Return(types.OnboardingConfig{}, nil)
		onboarding, err := New(context.Background(), resourceManager, hardwareManager, configRepo)
		require.NoError(t, err)
		require.NotNil(t, onboarding)

		config := types.OnboardingConfig{
			IsOnboarded: false,
		}
		machineResources := types.MachineResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      4,
					ClockSpeed: 24000,
				},
				RAM: types.RAM{
					Size: 16000000000,
				},
				Disk: types.Disk{
					Size: 100000000000,
				},
				GPUs: []types.GPU{
					{
						Index: 0,
						VRAM:  16000000000,
					},
				},
			},
		}
		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      2,
					ClockSpeed: 24000,
				},
				RAM: types.RAM{
					Size: 8000000000,
				},
				Disk: types.Disk{
					Size: 50000000000,
				},
				GPUs: []types.GPU{
					{
						Index: 0,
						VRAM:  8000000000,
					},
				},
			},
		}
		config.OnboardedResources = onboardedResources.Resources

		configRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(config, nil)
		hardwareManager.EXPECT().GetMachineResources().Return(machineResources, nil)
		hardwareManager.EXPECT().GetFreeResources().Return(machineResources.Resources, nil)
		resourceManager.EXPECT().UpdateOnboardedResources(gomock.Any(), onboardedResources.Resources).Return(nil)
		_, err = onboarding.Onboard(context.Background(), config)
		require.NoError(t, err)

		resourceManager.EXPECT().GetOnboardedResources(gomock.Any()).Return(onboardedResources, nil)
		info, err := onboarding.Info(context.Background())
		require.NoError(t, err)
		require.True(t, info.IsOnboarded)
		require.Equal(t, config.OnboardedResources, info.OnboardedResources)
	})

	t.Run("must be able to offboard a machine", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		resourceManager := NewMockResourceManager(ctrl)
		hardwareManager := NewMockHardwareManager(ctrl)
		configRepo := NewMockGenericEntityRepository[types.OnboardingConfig](ctrl)

		configRepo.EXPECT().Get(context.Background()).Return(types.OnboardingConfig{}, nil)
		onboarding, err := New(context.Background(), resourceManager, hardwareManager, configRepo)
		require.NoError(t, err)
		require.NotNil(t, onboarding)

		config := types.OnboardingConfig{
			IsOnboarded: false,
		}
		machineResources := types.MachineResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      4,
					ClockSpeed: 24000,
				},
				RAM: types.RAM{
					Size: 16000000000,
				},
				Disk: types.Disk{
					Size: 100000000000,
				},
				GPUs: []types.GPU{
					{
						Index: 0,
						VRAM:  16000000000,
					},
				},
			},
		}
		onboardedResources := types.OnboardedResources{
			Resources: types.Resources{
				CPU: types.CPU{
					Cores:      2,
					ClockSpeed: 24000,
				},
				RAM: types.RAM{
					Size: 8000000000,
				},
				Disk: types.Disk{
					Size: 50000000000,
				},
				GPUs: []types.GPU{
					{
						Index: 0,
						VRAM:  8000000000,
					},
				},
			},
		}
		config.OnboardedResources = onboardedResources.Resources

		configRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(config, nil)
		hardwareManager.EXPECT().GetMachineResources().Return(machineResources, nil)
		hardwareManager.EXPECT().GetFreeResources().Return(machineResources.Resources, nil)
		resourceManager.EXPECT().UpdateOnboardedResources(gomock.Any(), onboardedResources.Resources).Return(nil)
		_, err = onboarding.Onboard(context.Background(), config)
		require.NoError(t, err)

		isOnboarded, err := onboarding.IsOnboarded()
		require.NoError(t, err)
		require.True(t, isOnboarded)

		configRepo.EXPECT().Clear(gomock.Any()).Return(nil)
		resourceManager.EXPECT().UpdateOnboardedResources(gomock.Any(), types.Resources{}).Return(nil)
		err = onboarding.Offboard(context.Background())
		require.NoError(t, err)
	})
}

func TestOnboarding_Validations(t *testing.T) {
	t.Parallel()

	t.Run("must pass onboarding capacity checks", func(t *testing.T) {
		t.Parallel()

		t.Run("must ensure min cpu cores", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			resourceManager := NewMockResourceManager(ctrl)
			hardwareManager := NewMockHardwareManager(ctrl)
			configRepo := NewMockGenericEntityRepository[types.OnboardingConfig](ctrl)

			configRepo.EXPECT().Get(context.Background()).Return(types.OnboardingConfig{}, nil)
			onboarding, err := New(context.Background(), resourceManager, hardwareManager, configRepo)
			require.NoError(t, err)
			require.NotNil(t, onboarding)

			var config types.OnboardingConfig
			machineResources := types.MachineResources{
				Resources: types.Resources{
					CPU: types.CPU{
						Cores:      2,
						ClockSpeed: 24000,
					},
				},
			}
			onboardedResources := types.OnboardedResources{
				Resources: types.Resources{
					CPU: types.CPU{
						Cores:      0.5,
						ClockSpeed: 1000,
					},
				},
			}
			config.OnboardedResources = onboardedResources.Resources

			hardwareManager.EXPECT().GetMachineResources().Return(machineResources, nil)
			_, err = onboarding.Onboard(context.Background(), config)
			require.Error(t, err)
			require.Contains(t, err.Error(), "cores must be between 1 and 2")
		})

		t.Run("must ensure max cpu cores", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			resourceManager := NewMockResourceManager(ctrl)
			hardwareManager := NewMockHardwareManager(ctrl)
			configRepo := NewMockGenericEntityRepository[types.OnboardingConfig](ctrl)

			configRepo.EXPECT().Get(context.Background()).Return(types.OnboardingConfig{}, nil)
			onboarding, err := New(context.Background(), resourceManager, hardwareManager, configRepo)
			require.NoError(t, err)
			require.NotNil(t, onboarding)

			var config types.OnboardingConfig
			machineResources := types.MachineResources{
				Resources: types.Resources{
					CPU: types.CPU{
						Cores:      2,
						ClockSpeed: 24000,
					},
				},
			}
			onboardedResources := types.OnboardedResources{
				Resources: types.Resources{
					CPU: types.CPU{
						Cores:      3,
						ClockSpeed: 1000,
					},
				},
			}
			config.OnboardedResources = onboardedResources.Resources

			hardwareManager.EXPECT().GetMachineResources().Return(machineResources, nil)
			_, err = onboarding.Onboard(context.Background(), config)
			require.Error(t, err)
			require.Contains(t, err.Error(), "cores must be between 1 and 2")
		})

		t.Run("must ensure min ram size", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			resourceManager := NewMockResourceManager(ctrl)
			hardwareManager := NewMockHardwareManager(ctrl)
			configRepo := NewMockGenericEntityRepository[types.OnboardingConfig](ctrl)

			configRepo.EXPECT().Get(context.Background()).Return(types.OnboardingConfig{}, nil)
			onboarding, err := New(context.Background(), resourceManager, hardwareManager, configRepo)
			require.NoError(t, err)
			require.NotNil(t, onboarding)

			var config types.OnboardingConfig
			machineResources := types.MachineResources{
				Resources: types.Resources{
					CPU: types.CPU{
						Cores:      2,
						ClockSpeed: 24000,
					},
					RAM: types.RAM{
						Size: 8000000000, // 8 GB
					},
				},
			}
			onboardedResources := types.OnboardedResources{
				Resources: types.Resources{
					CPU: types.CPU{
						Cores:      1,
						ClockSpeed: 24000,
					},
					RAM: types.RAM{
						Size: 10000000, // 0.01 GB
					},
				},
			}
			config.OnboardedResources = onboardedResources.Resources

			hardwareManager.EXPECT().GetMachineResources().Return(machineResources, nil)
			_, err = onboarding.Onboard(context.Background(), config)
			require.Error(t, err)
			require.Contains(t, err.Error(), "expected RAM to be between 0.80 GB and 7.20 GB, got 0.01 GB")
		})

		t.Run("must ensure max ram size", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			resourceManager := NewMockResourceManager(ctrl)
			hardwareManager := NewMockHardwareManager(ctrl)
			configRepo := NewMockGenericEntityRepository[types.OnboardingConfig](ctrl)

			configRepo.EXPECT().Get(context.Background()).Return(types.OnboardingConfig{}, nil)
			onboarding, err := New(context.Background(), resourceManager, hardwareManager, configRepo)
			require.NoError(t, err)
			require.NotNil(t, onboarding)

			var config types.OnboardingConfig
			machineResources := types.MachineResources{
				Resources: types.Resources{
					CPU: types.CPU{
						Cores:      2,
						ClockSpeed: 24000,
					},
					RAM: types.RAM{
						Size: 8000000000, // 8 GB
					},
				},
			}
			onboardedResources := types.OnboardedResources{
				Resources: types.Resources{
					CPU: types.CPU{
						Cores:      1,
						ClockSpeed: 24000,
					},
					RAM: types.RAM{
						Size: 100000000000, // 100 GB
					},
				},
			}
			config.OnboardedResources = onboardedResources.Resources

			hardwareManager.EXPECT().GetMachineResources().Return(machineResources, nil)
			_, err = onboarding.Onboard(context.Background(), config)
			require.Error(t, err)
			require.Contains(t, err.Error(), "expected RAM to be between 0.80 GB and 7.20 GB, got 100.00 GB")
		})

		t.Run("must ensure min gpu vram size", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			resourceManager := NewMockResourceManager(ctrl)
			hardwareManager := NewMockHardwareManager(ctrl)
			configRepo := NewMockGenericEntityRepository[types.OnboardingConfig](ctrl)

			configRepo.EXPECT().Get(context.Background()).Return(types.OnboardingConfig{}, nil)
			onboarding, err := New(context.Background(), resourceManager, hardwareManager, configRepo)
			require.NoError(t, err)
			require.NotNil(t, onboarding)

			var config types.OnboardingConfig
			machineResources := types.MachineResources{
				Resources: types.Resources{
					CPU: types.CPU{
						Cores:      2,
						ClockSpeed: 24000,
					},
					RAM: types.RAM{
						Size: 8000000000, // 8 GB
					},
					GPUs: []types.GPU{
						{
							Index: 0,
							VRAM:  8000000000, // 8 GB
						},
					},
				},
			}
			onboardedResources := types.OnboardedResources{
				Resources: types.Resources{
					CPU: types.CPU{
						Cores:      1,
						ClockSpeed: 24000,
					},
					RAM: types.RAM{
						Size: 4000000000, // 4 GB
					},
					GPUs: []types.GPU{
						{
							Index: 0,
							VRAM:  10000000, // 0.01 GB
						},
					},
				},
			}
			config.OnboardedResources = onboardedResources.Resources

			hardwareManager.EXPECT().GetMachineResources().Return(machineResources, nil)
			_, err = onboarding.Onboard(context.Background(), config)
			require.Error(t, err)
			require.Contains(t, err.Error(), "expected GPU 0 VRAM to be between 0.80 and 7.20, got 0.01")
		})

		t.Run("must ensure max gpu vram size", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			resourceManager := NewMockResourceManager(ctrl)
			hardwareManager := NewMockHardwareManager(ctrl)
			configRepo := NewMockGenericEntityRepository[types.OnboardingConfig](ctrl)

			configRepo.EXPECT().Get(context.Background()).Return(types.OnboardingConfig{}, nil)
			onboarding, err := New(context.Background(), resourceManager, hardwareManager, configRepo)
			require.NoError(t, err)
			require.NotNil(t, onboarding)

			var config types.OnboardingConfig
			machineResources := types.MachineResources{
				Resources: types.Resources{
					CPU: types.CPU{
						Cores:      2,
						ClockSpeed: 24000,
					},
					RAM: types.RAM{
						Size: 8000000000, // 8 GB
					},
					GPUs: []types.GPU{
						{
							Index: 0,
							VRAM:  8000000000, // 8 GB
						},
					},
				},
			}
			onboardedResources := types.OnboardedResources{
				Resources: types.Resources{
					CPU: types.CPU{
						Cores:      1,
						ClockSpeed: 24000,
					},
					RAM: types.RAM{
						Size: 4000000000, // 4 GB
					},
					GPUs: []types.GPU{
						{
							Index: 0,
							VRAM:  100000000000, // 100 GB
						},
					},
				},
			}
			config.OnboardedResources = onboardedResources.Resources

			hardwareManager.EXPECT().GetMachineResources().Return(machineResources, nil)
			_, err = onboarding.Onboard(context.Background(), config)
			require.Error(t, err)
			require.Contains(t, err.Error(), "expected GPU 0 VRAM to be between 0.80 and 7.20, got 100.00")
		})
	})

	t.Run("must pass machine usage checks", func(t *testing.T) {
		t.Parallel()

		t.Run("must ensure there are enough free compute on the system", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			resourceManager := NewMockResourceManager(ctrl)
			hardwareManager := NewMockHardwareManager(ctrl)
			configRepo := NewMockGenericEntityRepository[types.OnboardingConfig](ctrl)

			configRepo.EXPECT().Get(context.Background()).Return(types.OnboardingConfig{}, nil)
			onboarding, err := New(context.Background(), resourceManager, hardwareManager, configRepo)
			require.NoError(t, err)
			require.NotNil(t, onboarding)

			var config types.OnboardingConfig
			machineResources := types.MachineResources{
				Resources: types.Resources{
					CPU: types.CPU{
						Cores:      2,
						ClockSpeed: 24000,
					},
				},
			}
			onboardedResources := types.OnboardedResources{
				Resources: types.Resources{
					CPU: types.CPU{
						Cores:      1,
						ClockSpeed: 24000,
					},
				},
			}
			config.OnboardedResources = onboardedResources.Resources

			hardwareManager.EXPECT().GetMachineResources().Return(machineResources, nil)
			machineResources.CPU = types.CPU{
				Cores:      0.1,
				ClockSpeed: 24000,
			}
			hardwareManager.EXPECT().GetFreeResources().Return(machineResources.Resources, nil)

			_, err = onboarding.Onboard(context.Background(), config)
			require.Error(t, err)
			require.Contains(t, err.Error(), "not enough free compute available on the system: 0.00 GHz")
		})

		t.Run("must ensure there is enough free RAM on the system", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			resourceManager := NewMockResourceManager(ctrl)
			hardwareManager := NewMockHardwareManager(ctrl)
			configRepo := NewMockGenericEntityRepository[types.OnboardingConfig](ctrl)

			configRepo.EXPECT().Get(context.Background()).Return(types.OnboardingConfig{}, nil)
			onboarding, err := New(context.Background(), resourceManager, hardwareManager, configRepo)
			require.NoError(t, err)
			require.NotNil(t, onboarding)

			var config types.OnboardingConfig
			machineResources := types.MachineResources{
				Resources: types.Resources{
					CPU: types.CPU{
						Cores:      2,
						ClockSpeed: 24000,
					},
					RAM: types.RAM{
						Size: 8000000000,
					},
				},
			}
			onboardedResources := types.OnboardedResources{
				Resources: types.Resources{
					CPU: types.CPU{
						Cores:      1,
						ClockSpeed: 24000,
					},
					RAM: types.RAM{
						Size: 4000000000,
					},
				},
			}
			config.OnboardedResources = onboardedResources.Resources

			hardwareManager.EXPECT().GetMachineResources().Return(machineResources, nil)
			machineResources.RAM = types.RAM{
				Size: 1000000000,
			}
			hardwareManager.EXPECT().GetFreeResources().Return(machineResources.Resources, nil)
			_, err = onboarding.Onboard(context.Background(), config)
			require.Error(t, err)
			require.Contains(t, err.Error(), "not enough free RAM available on the system: 1.00 GB")
		})

		t.Run("must ensure there is enough free VRAM on the gpu", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			resourceManager := NewMockResourceManager(ctrl)
			hardwareManager := NewMockHardwareManager(ctrl)
			configRepo := NewMockGenericEntityRepository[types.OnboardingConfig](ctrl)

			configRepo.EXPECT().Get(context.Background()).Return(types.OnboardingConfig{}, nil)
			onboarding, err := New(context.Background(), resourceManager, hardwareManager, configRepo)
			require.NoError(t, err)
			require.NotNil(t, onboarding)

			var config types.OnboardingConfig
			machineResources := types.MachineResources{
				Resources: types.Resources{
					CPU: types.CPU{
						Cores:      2,
						ClockSpeed: 24000,
					},
					RAM: types.RAM{
						Size: 8000000000,
					},
					GPUs: []types.GPU{
						{
							Index: 0,
							VRAM:  8000000000, // 8 GB
						},
					},
				},
			}
			onboardedResources := types.OnboardedResources{
				Resources: types.Resources{
					CPU: types.CPU{
						Cores:      1,
						ClockSpeed: 24000,
					},
					RAM: types.RAM{
						Size: 4000000000,
					},
					GPUs: []types.GPU{
						{
							Index: 0,
							VRAM:  4000000000, // 4 GB
						},
					},
				},
			}
			config.OnboardedResources = onboardedResources.Resources

			hardwareManager.EXPECT().GetMachineResources().Return(machineResources, nil)
			hardwareManager.EXPECT().GetFreeResources().Return(
				types.Resources{
					CPU: machineResources.CPU,
					RAM: machineResources.RAM,
					GPUs: []types.GPU{
						{
							Index: 0,
							VRAM:  100000000, // 0.10 GB
						},
					},
				}, nil)
			_, err = onboarding.Onboard(context.Background(), config)
			require.Error(t, err)
			require.Contains(t, err.Error(), "not enough free VRAM available on GPU : 0.10 GB")
		})
	})
}
