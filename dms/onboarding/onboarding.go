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
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

var (
	// ErrMachineNotOnboarded is returned when the machine is not onboarded and is expected to be
	ErrMachineNotOnboarded = errors.New("machine is not onboarded")
	// ErrUnmetCapacity is returned when the onboarded resources doesn't meet the required capacity
	ErrUnmetCapacity = errors.New("capacity not met")
	// ErrHighUsage is returned when the machine has high usage compared to the onboarded resources
	ErrHighUsage = errors.New("machine has high usage")
	// ErrOutOfRange is returned when the actual value is out of the expected range
	ErrOutOfRange = errors.New("out of range")
)

// validateRange validates the actual value is within the min and max range
func validateRange(actual, minimum, maximum float64) error {
	if actual < minimum || actual > maximum {
		return ErrOutOfRange
	}
	return nil
}

// populateOnboardingConfig populates the onboarding config fields
func populateOnboardingConfig(ctx context.Context,
	config *types.OnboardingConfig,
	resourceManager types.ResourceManager,
) error {
	onboardedResources, err := resourceManager.GetOnboardedResources(ctx)
	if err != nil {
		return fmt.Errorf("could not get onboarded resources: %w", err)
	}
	config.OnboardedResources = onboardedResources.Resources

	// ...
	// load other fields
	// ...
	return nil
}

// Onboarding implements the OnboardingManager interface
type Onboarding struct {
	ResourceManager types.ResourceManager
	Hardware        types.HardwareManager
	// ConfigRepo is the db repository to store the onboarding config
	ConfigRepo repositories.OnboardingConfig
	// Config is the cached onboarding configuration
	Config types.OnboardingConfig
	// Lock is the lock to protect the onboarding state
	Lock sync.RWMutex
}

// Ensure Onboarding implements the OnboardingManager interface
var _ types.OnboardingManager = (*Onboarding)(nil)

// New creates a new Onboarding instance
func New(ctx context.Context,
	resourceManager types.ResourceManager,
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

	// try loading the onboarding config from the database
	config, err := configRepo.Get(ctx)
	if err != nil {
		if !errors.Is(err, repositories.ErrNotFound) {
			return nil, fmt.Errorf("could not get onboarding config: %w", err)
		}

		config = types.OnboardingConfig{}
	}

	onboardingManager := &Onboarding{
		ResourceManager: resourceManager,
		Hardware:        hardwareManager,
		ConfigRepo:      configRepo,
		Config:          config,
	}
	// validate the onboarding config
	if onboardingManager.Config.IsOnboarded {
		// populate the onboarded resources
		if err := populateOnboardingConfig(ctx, &config, resourceManager); err != nil {
			return nil, fmt.Errorf("populate onboarding config: %w", err)
		}

		if err := onboardingManager.validatePrerequisites(config); err != nil {
			switch {
			case errors.Is(err, ErrUnmetCapacity):
				log.Errorw("machine onboarded, but capacity not fully met",
					"labels", []string{string(observability.LabelNode)},
					"error", err)
			case errors.Is(err, ErrHighUsage):
				log.Errorw("machine onboarded, but high usage detected",
					"labels", []string{string(observability.LabelNode)},
					"error", err)
				return onboardingManager, nil
			default:
				log.Errorw("machine is onboarded but prerequisites are not met",
					"labels", []string{string(observability.LabelNode)},
					"error", err)
			}

			// if the machine is onboarded but the prerequisites are not met, offboard the machine
			log.Infow("offboarding the machine because onboarded resources are no longer valid",
				"labels", []string{string(observability.LabelNode)})
			if err := onboardingManager.Offboard(context.Background()); err != nil {
				return nil, fmt.Errorf("offboard the machine: %w", err)
			}
		}
	}

	return onboardingManager, nil
}

// validateCapacity validates the machine capacity for the requested onboarding resources
func validateCapacity(onboardedResources, machineResources types.Resources) error {
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
func validateUsage(onboardedResources, systemFreeResources types.Resources) error {
	if onboardedResources.CPU.Compute() > systemFreeResources.CPU.Compute() {
		return fmt.Errorf("not enough free compute available on the system: %.2f GHz", systemFreeResources.CPU.ComputeInGHz())
	}

	if onboardedResources.RAM.Size > systemFreeResources.RAM.Size {
		return fmt.Errorf("not enough free RAM available on the system: %.2f GB", systemFreeResources.RAM.SizeInGB())
	}

	// TODO: validate disk usage

	for _, gpu := range onboardedResources.GPUs {
		selectedGPU, err := systemFreeResources.GPUs.GetWithIndex(gpu.Index)
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
	machineResources, err := o.Hardware.GetMachineResources()
	if err != nil {
		return fmt.Errorf("could not get machine resources: %w", err)
	}

	if err := validateCapacity(config.OnboardedResources, machineResources.Resources); err != nil {
		return fmt.Errorf("%w: %v", ErrUnmetCapacity, err)
	}

	systemFreeResources, err := o.Hardware.GetFreeResources()
	if err != nil {
		return fmt.Errorf("could not get system free resources: %w", err)
	}
	if err := validateUsage(config.OnboardedResources, systemFreeResources); err != nil {
		return fmt.Errorf("%w: %v", ErrHighUsage, err)
	}

	return nil
}

// Onboard validates the onboarding params and onboards the machine to the network
func (o *Onboarding) Onboard(ctx context.Context, config types.OnboardingConfig) (types.OnboardingConfig, error) {
	o.Lock.Lock()
	defer o.Lock.Unlock()

	log.Debugf("onboarding machine with config: %+v", config)

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

	log.Infow("machine_onboarded_successfully",
		"labels", []string{string(observability.LabelNode)})

	o.Config = config
	return o.Config, nil
}

// Offboard offboards the machine from the network
func (o *Onboarding) Offboard(ctx context.Context) error {
	o.Lock.Lock()
	defer o.Lock.Unlock()

	if !o.Config.IsOnboarded {
		return ErrMachineNotOnboarded
	}

	err := o.ConfigRepo.Clear(ctx)
	if err != nil {
		return fmt.Errorf("failed to clear onboarding config from db: %w", err)
	}

	o.Config.IsOnboarded = false
	// clear the onboarded resources
	if err := o.ResourceManager.UpdateOnboardedResources(ctx, types.Resources{}); err != nil {
		return fmt.Errorf("could not clear onboarded resources: %w", err)
	}

	log.Infow("machine_offboarded_successfully",
		"labels", []string{string(observability.LabelNode)})

	return nil
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
	defer o.Lock.RUnlock()
	info := o.Config

	if err := populateOnboardingConfig(ctx, &info, o.ResourceManager); err != nil {
		return types.OnboardingConfig{}, fmt.Errorf("populate onboarding config: %w", err)
	}
	return info, nil
}
