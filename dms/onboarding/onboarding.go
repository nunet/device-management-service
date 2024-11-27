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
	"errors"
	"fmt"
	"sync"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

var (
	ErrMachineNotOnboarded = errors.New("machine is not onboarded")
	ErrOutOfRange          = errors.New("out of range")
)

// Onboarding implements the OnboardingManager interface
type Onboarding struct {
	ResourceManager types.ResourceManager
	Hardware        types.HardwareManager
	ConfigRepo      repositories.OnboardingConfig
	Config          types.OnboardingConfig
	Lock            sync.RWMutex
}

// Ensure Onboarding implements the OnboardingManager interface
var _ types.OnboardingManager = (*Onboarding)(nil)

// New is a constructor for Onboarding
func New(resourceManager types.ResourceManager,
	hardwareManager types.HardwareManager,
	configRepo repositories.OnboardingConfig,
) (*Onboarding, error) {
	if resourceManager == nil {
		return nil, fmt.Errorf("resource manager is required")
	}

	if hardwareManager == nil {
		return nil, fmt.Errorf("hardware manager is required")
	}

	if configRepo == nil {
		return nil, fmt.Errorf("config repo is required")
	}

	config, err := configRepo.Get(context.Background())
	if err != nil {
		if !errors.Is(err, repositories.ErrNotFound) {
			return nil, fmt.Errorf("could not get onboarding config: %w", err)
		}

		config = types.OnboardingConfig{}
	}

	return &Onboarding{
		ResourceManager: resourceManager,
		Hardware:        hardwareManager,
		ConfigRepo:      configRepo,
		Config:          config,
	}, nil
}

// IsOnboarded checks whether the machine is onboarded or not
func (o *Onboarding) IsOnboarded() (bool, error) {
	o.Lock.RLock()
	defer o.Lock.RUnlock()
	return o.Config.IsOnboarded, nil
}

// Info returns the onboarding configuration
func (o *Onboarding) Info(ctx context.Context) (types.OnboardingConfig, error) {
	o.Lock.RLock()
	info := o.Config
	o.Lock.RUnlock()

	// get onboarded resources from the resource manager
	resources, err := o.ResourceManager.GetOnboardedResources(ctx)
	if err != nil {
		return types.OnboardingConfig{}, fmt.Errorf("could not get onboarded resources: %w", err)
	}
	info.OnboardedResources = resources.Resources

	// get machine resources
	machineResources, err := o.Hardware.GetMachineResources()
	if err != nil {
		return types.OnboardingConfig{}, fmt.Errorf("could not get machine resources: %w", err)
	}
	info.MachineResources = machineResources.Resources

	return info, nil
}

// Onboard validates the onboarding params and onboards the machine to the network
func (o *Onboarding) Onboard(ctx context.Context, config types.OnboardingConfig) (types.OnboardingConfig, error) {
	o.Lock.Lock()
	defer o.Lock.Unlock()
	log.Debugf("onboarding the machine with the config: %+v", config)
	// populate the config with machine resources
	machineResources, err := o.Hardware.GetMachineResources()
	if err != nil {
		return types.OnboardingConfig{}, fmt.Errorf("could not get machine resources: %w", err)
	}
	config.MachineResources = machineResources.Resources

	if err := o.validatePrerequisites(config); err != nil {
		return types.OnboardingConfig{}, fmt.Errorf("could not validate onboarding prerequisites: %w", err)
	}

	if err := o.ResourceManager.UpdateOnboardedResources(ctx, config.OnboardedResources); err != nil {
		return types.OnboardingConfig{}, fmt.Errorf("could not update onboarded resources: %w", err)
	}

	config.IsOnboarded = true
	if _, err := o.ConfigRepo.Save(ctx, config); err != nil {
		return types.OnboardingConfig{}, fmt.Errorf("could not save onboarding config: %w", err)
	}

	o.Config = config
	return o.Config, nil
}

// Offboard offboards the machine from the network by clearing the onboarding config from the database
func (o *Onboarding) Offboard(ctx context.Context) error {
	o.Lock.Lock()
	defer o.Lock.Unlock()

	if !o.Config.IsOnboarded {
		return ErrMachineNotOnboarded
	}

	log.Info("offboarding the machine")
	err := o.ConfigRepo.Clear(ctx)
	if err != nil {
		return fmt.Errorf("failed to clear onboarding config from db: %w", err)
	}

	o.Config.IsOnboarded = false
	// clear the onboarded resources
	if err := o.ResourceManager.UpdateOnboardedResources(ctx, types.Resources{}); err != nil {
		return fmt.Errorf("could not clear onboarded resources: %w", err)
	}

	return nil
}

func validateRange(actual, min, max float64) error {
	if actual < min || actual > max {
		return ErrOutOfRange
	}
	return nil
}

// validateCapacity validates the machine capacity for the requested onboarding resources
func (o *Onboarding) validateCapacity(onboardedResources, machineResources types.Resources) error {
	if onboardedResources.CPU.Cores < 1 || onboardedResources.CPU.Cores > machineResources.CPU.Cores {
		return fmt.Errorf("cores must be between %d and %.0f", 1, machineResources.CPU.Cores)
	}

	if err := validateRange(
		onboardedResources.RAM.Size,
		machineResources.RAM.Size/10,
		machineResources.RAM.Size*9/10,
	); err != nil {
		if errors.Is(err, ErrOutOfRange) {
			return fmt.Errorf("expected RAM to be between %.2f GB and %.2f GB, got %.2f GB",
				machineResources.RAM.SizeInGB()/10,
				machineResources.RAM.SizeInGB()*9/10,
				onboardedResources.RAM.SizeInGB(),
			)
		}

		return fmt.Errorf("validating resource range for RAM: %w", err)
	}

	// TODO: validate disk size

	for _, gpu := range onboardedResources.GPUs {
		selectedGPU, err := machineResources.GPUs.GetWithIndex(gpu.Index)
		if err != nil {
			return fmt.Errorf("could not get find gpu: %w", err)
		}

		if err := validateRange(
			gpu.VRAM,
			selectedGPU.VRAM/10,
			selectedGPU.VRAM*9/10,
		); err != nil {
			if errors.Is(err, ErrOutOfRange) {
				return fmt.Errorf("expected GPU %d VRAM to be between %.2f and %.2f, got %.2f",
					gpu.Index,
					selectedGPU.VRAMInGB()/10,
					selectedGPU.VRAMInGB()*9/10,
					gpu.VRAMInGB(),
				)
			}

			return fmt.Errorf("validating resource range for GPU %d: %w", gpu.Index, err)
		}
	}

	return nil
}

// validateUsage validates the machine usage for the requested onboarding resources
func (o *Onboarding) validateUsage(resources types.Resources) error {
	freeResources, err := o.Hardware.GetFreeResources()
	if err != nil {
		return fmt.Errorf("could not get usage data: %w", err)
	}

	if resources.CPU.Compute() > freeResources.CPU.Compute() {
		return fmt.Errorf("not enough free compute available on the system: %.2f GHz", freeResources.CPU.ComputeInGHz())
	}

	if resources.RAM.Size > freeResources.RAM.Size {
		return fmt.Errorf("not enough free RAM available on the system: %.2f GB", freeResources.RAM.SizeInGB())
	}

	// TODO: validate disk usage

	for _, gpu := range resources.GPUs {
		selectedGPU, err := freeResources.GPUs.GetWithIndex(gpu.Index)
		if err != nil {
			return fmt.Errorf("could not find gpu: %w", err)
		}

		if gpu.VRAM > selectedGPU.VRAM {
			return fmt.Errorf("not enough free VRAM available on GPU %s: %.2f GB", gpu.Model, selectedGPU.VRAMInGB())
		}
	}

	return nil
}

// validatePrerequisites validates the onboarding prerequisites
func (o *Onboarding) validatePrerequisites(config types.OnboardingConfig) error {
	if err := o.validateCapacity(config.OnboardedResources, config.MachineResources); err != nil {
		return fmt.Errorf("could not validate capacity data: %w", err)
	}

	if err := o.validateUsage(config.OnboardedResources); err != nil {
		return fmt.Errorf("could not validate usage data: %w", err)
	}

	return nil
}
