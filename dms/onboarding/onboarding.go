package onboarding

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

var ErrMachineNotOnboarded = errors.New("machine is not onboarded")

type Config struct {
	Fs              afero.Afero
	WorkDir         string
	DatabasePath    string
	ConfigRepo      repositories.OnboardingConfig
	P2PRepo         repositories.Libp2pInfo
	ResourceManager types.ResourceManager
	Hardware        types.HardwareManager
	UUIDRepo        repositories.MachineUUID
}

// NewConfig is a constructor for Config
func NewConfig(
	fs afero.Afero,
	workDir, dbPath string,
	configRepo repositories.OnboardingConfig,
	p2pRepo repositories.Libp2pInfo,
	uuidRepo repositories.MachineUUID,
) *Config {
	return &Config{
		Fs:           fs,
		WorkDir:      workDir,
		DatabasePath: dbPath,
		ConfigRepo:   configRepo,
		P2PRepo:      p2pRepo,
		UUIDRepo:     uuidRepo,
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
	log.Debugf("machine onboarded successfully")

	return nil
}

// Update updates the onboarding configuration
// Currently, it only updates the onboarded resources
func (o *Onboarding) Update(ctx context.Context, config types.OnboardingConfig) error {
	log.Debugf("updating the onboarding config with the new config: %+v", config)
	// update onboarded resources if there is a change with the existing config
	onboardedResources, err := o.ResourceManager.GetOnboardedResources(ctx)
	if err != nil {
		return fmt.Errorf("could not get onboarded resources: %w", err)
	}

	if onboardedResources.Equal(config.OnboardedResources) {
		return nil
	}

	if err := o.ResourceManager.UpdateOnboardedResources(ctx, config.OnboardedResources); err != nil {
		return fmt.Errorf("could not update onboarded resources: %w", err)
	}
	log.Debugf("onboarding config updated successfully")

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

// validateCapacity validates the resource capacity data
// It checks if the CPU and memory are within 10% and 90% of the available resources
func (o *Onboarding) validateCapacity(resources types.Resources) error {
	// TODO: https://gitlab.com/nunet/device-management-service/-/merge_requests/563#note_2139212199
	machineResources, err := o.Hardware.GetMachineResources()
	if err != nil {
		return fmt.Errorf("could not get provisioned resources: %w", err)
	}

	if resources.CPU.Cores < 1 || resources.CPU.Cores > machineResources.CPU.Cores {
		return fmt.Errorf("cores must be between %d and %.0f", 1, machineResources.CPU.Cores)
	}

	if resources.RAM.Size > machineResources.RAM.Size*9/10 || resources.RAM.Size < machineResources.RAM.Size/10 {
		return fmt.Errorf("memory should be between 10%% and 90%% of the available memory (%.2f and %.2f): %.2f",
			types.ConvertBytesToGB(machineResources.RAM.Size/10),
			types.ConvertBytesToGB(machineResources.RAM.Size*9/10),
			types.ConvertBytesToGB(resources.RAM.Size),
		)
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

	return nil
}
