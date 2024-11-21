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

	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

var (
	ErrMachineNotOnboarded = errors.New("machine is not onboarded")
	ErrOutOfRange          = errors.New("out of range")
)

type Config struct {
	Fs              afero.Afero
	WorkDir         string
	DatabasePath    string
	ConfigRepo      repositories.OnboardingConfig
	ResourceManager types.ResourceManager
	Hardware        types.HardwareManager
}

// NewConfig is a constructor for Config
func NewConfig(
	fs afero.Afero,
	workDir, dbPath string,
	configRepo repositories.OnboardingConfig,
) *Config {
	return &Config{
		Fs:           fs,
		WorkDir:      workDir,
		DatabasePath: dbPath,
		ConfigRepo:   configRepo,
	}
}

// Onboarding acts a receiver for methods related to onboarding
type Onboarding struct {
	Config
}

// New is a constructor for Onboarding
func New(config *Config) *Onboarding {
	return &Onboarding{Config: *config}
}

// IsOnboarded checks whether the machine is onboarded or not
func (o *Onboarding) IsOnboarded(ctx context.Context) (bool, error) {
	_, err := o.ConfigRepo.Get(ctx)
	if err != nil {
		return false, err
	}
	// TODO: validate onboarding params
	return true, nil
}

// Info returns the onboarding configuration
// It fetches the onboarding config from the database and the onboarded resources from the resource manager
// It also fetches the machine resources from the hardware package
func (o *Onboarding) Info(ctx context.Context) (types.OnboardingConfig, error) {
	info, err := o.ConfigRepo.Get(ctx)
	if err != nil {
		return types.OnboardingConfig{}, fmt.Errorf("could not get onboarding config: %w", err)
	}

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
// It saves the onboarding config to the database and updates the onboarded resources in the resource manager
func (o *Onboarding) Onboard(ctx context.Context, config types.OnboardingConfig) error {
	log.Debugf("onboarding the machine with the config: %+v", config)
	if err := o.validatePrerequisites(config); err != nil {
		return fmt.Errorf("could not validate onboarding prerequisites: %w", err)
	}

	if err := o.ResourceManager.UpdateOnboardedResources(ctx, config.OnboardedResources); err != nil {
		return fmt.Errorf("could not update onboarded resources: %w", err)
	}

	if _, err := o.ConfigRepo.Save(ctx, config); err != nil {
		return fmt.Errorf("could not save onboarding config: %w", err)
	}

	return nil
}

// Offboard offboards the machine from the network by clearing the onboarding config from the database
func (o *Onboarding) Offboard(ctx context.Context, force bool) error {
	onboarded, err := o.IsOnboarded(ctx)
	if err != nil && !force {
		if errors.Is(err, ErrMachineNotOnboarded) {
			return ErrMachineNotOnboarded
		}

		return fmt.Errorf("could not retrieve onboard status: %w", err)
	}

	if err != nil {
		log.Errorf("problem with onboarding state: %v", err)
		log.Info("continuing with offboarding because forced")
	}

	if !onboarded {
		return fmt.Errorf("machine is not onboarded")
	}

	// TODO: shutdown routine to stop networking etc... here

	err = o.ConfigRepo.Clear(ctx)
	if err != nil && !force {
		return fmt.Errorf("failed to clear onboarding config from db: %w", err)
	}

	if err != nil {
		log.Errorf("failed to clear onboarding config from db: %v", err)
		log.Info("continuing with offboarding because forced")
	}

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

func (o *Onboarding) validateCapacity(resources types.Resources) error {
	// TODO: https://gitlab.com/nunet/device-management-service/-/merge_requests/563#note_2139212199
	machineResources, err := o.Hardware.GetMachineResources()
	if err != nil {
		return fmt.Errorf("retrieve provisioned machine resources: %w", err)
	}

	if resources.CPU.Cores < 1 || resources.CPU.Cores > machineResources.CPU.Cores {
		return fmt.Errorf("cores must be between %d and %.0f", 1, machineResources.CPU.Cores)
	}

	if err := validateRange(
		resources.RAM.Size,
		machineResources.RAM.Size/10,
		machineResources.RAM.Size*9/10,
	); err != nil {
		if errors.Is(err, ErrOutOfRange) {
			return fmt.Errorf("expected RAM to be between %.2f and %.2f, got %.2f ",
				machineResources.RAM.SizeInGB()/10,
				machineResources.RAM.SizeInGB()*9/10,
				resources.RAM.SizeInGB(),
			)
		}

		return fmt.Errorf("validating resource range for RAM: %w", err)
	}

	for _, gpu := range resources.GPUs {
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

// validateUsage validates the resource usage data
// It checks if the there is enough resources available to onboard
func (o *Onboarding) validateUsage(resources types.Resources) error {
	freeResources, err := o.Hardware.GetFreeResources()
	if err != nil {
		return fmt.Errorf("could not get usage data: %w", err)
	}

	if resources.CPU.Compute() > freeResources.CPU.Compute() {
		return fmt.Errorf("CPU usage is too high: %.2f", freeResources.CPU.Compute())
	}

	if resources.RAM.Size > freeResources.RAM.Size {
		return fmt.Errorf("memory usage is too high: %.2f", freeResources.RAM.Size)
	}

	for _, gpu := range resources.GPUs {
		selectedGPU, err := freeResources.GPUs.GetWithIndex(gpu.Index)
		if err != nil {
			return fmt.Errorf("could not find gpu: %w", err)
		}

		if gpu.VRAM > selectedGPU.VRAM {
			return fmt.Errorf("GPU %s usage is too high: %.2f", gpu.Model, gpu.VRAM)
		}
	}

	return nil
}

// validatePrerequisites validates the onboarding prerequisites
func (o *Onboarding) validatePrerequisites(config types.OnboardingConfig) error {
	ok, err := o.Fs.DirExists(o.WorkDir)
	if err != nil {
		return fmt.Errorf("could not check if config directory exists: %w", err)
	}
	if !ok {
		return fmt.Errorf("working directory does not exist")
	}

	if err := o.validateCapacity(config.OnboardedResources); err != nil {
		return fmt.Errorf("could not validate capacity data: %w", err)
	}

	if err := o.validateUsage(config.OnboardedResources); err != nil {
		return fmt.Errorf("could not validate usage data: %w", err)
	}

	return nil
}
