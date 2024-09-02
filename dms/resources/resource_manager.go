package resources

import (
	"context"
	"fmt"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

// SystemSpecs is an interface that defines the methods to get the system specifications of the machine
type SystemSpecs interface {
	// GetSpecInfo returns the detailed specifications of the machine
	GetSpecInfo() (types.SpecInfo, error)
	// GetGPUVendors returns the GPU vendors of the machine
	GetGPUVendors() ([]types.GPUVendor, error)
	// GetGPUs returns the GPUs of the machine for the given vendors
	// If no vendors are provided, it returns the information of all the GPUs
	GetGPUs(vendors ...types.GPUVendor) ([]types.GPU, error)
	// GetTotalMemory returns the total memory of the machine in MB
	GetTotalMemory() (uint64, error)
	// GetTotalStorage returns the total storage of the machine in MB
	GetTotalStorage() (uint64, error)
	// GetCPUInfo returns the CPU information of the machine
	GetCPUInfo() (types.CPUInfo, error)
	// GetProvisionedResources returns the total resources of the machine
	GetProvisionedResources() (types.Resources, error)
}

// Manager is an interface that defines the methods to manage the resources of the machine
type Manager interface {
	// UpdateFreeResources calculates, updates db and returns the free resources of the machine in the database
	UpdateFreeResources(context.Context) (types.FreeResources, error)
	// GetOnboardedResources returns the onboarded resources of the machine
	GetOnboardedResources(context.Context) (types.OnboardedResources, error)
	// GetRequiredResources returns the resources required by the jobs running on the machine
	GetRequiredResources(context.Context) (types.Resources, error)
	// UpdateOnboardedResources updates the onboarded resources of the machine in the database
	UpdateOnboardedResources(context.Context, types.OnboardedResources) error
	// SystemSpecs returns the SystemSpecs instance
	SystemSpecs() SystemSpecs
	// UsageMonitor returns the UsageMonitor instance
	UsageMonitor() UsageMonitor

	// ... other methods
}

// DefaultManager implements the Manager interface
// TODO: do we want to have an in-memory cache for the resources instead of querying the DB every time?
// TODO: Add telemetry for the methods https://gitlab.com/nunet/device-management-service/-/issues/535
type DefaultManager struct {
	usageMonitor UsageMonitor
	systemSpecs  SystemSpecs
	repos        ManagerRepos
}

// ManagerRepos holds all the repositories needed for resource management
type ManagerRepos struct {
	FreeResources      repositories.FreeResources
	OnboardedResources repositories.OnboardedResources
	RequiredResources  repositories.RequiredResources
	VirtualMachine     repositories.VirtualMachine
	Services           repositories.Services
}

// NewResourceManager returns a new defaultResourceManager instance
func NewResourceManager(repos ManagerRepos) *DefaultManager {
	sysSpecs := newSystemSpecs()
	return &DefaultManager{
		usageMonitor: newUsageMonitor(
			sysSpecs,
			repos.VirtualMachine,
			repos.Services,
			repos.RequiredResources,
		),
		systemSpecs: sysSpecs,
		repos:       repos,
	}
}

var _ Manager = (*DefaultManager)(nil)

// UpdateFreeResources calculates, updates db and returns the free resources of the machine in the database
func (d DefaultManager) UpdateFreeResources(ctx context.Context) (types.FreeResources, error) {
	usage, err := d.usageMonitor.GetUsage(ctx)
	if err != nil {
		return types.FreeResources{}, fmt.Errorf("getting usage: %w", err)
	}

	onboardedResources, err := d.GetOnboardedResources(ctx)
	if err != nil {
		return types.FreeResources{}, fmt.Errorf("getting total resources: %w", err)
	}

	freeResources, err := onboardedResources.Subtract(usage)
	if err != nil {
		return types.FreeResources{}, fmt.Errorf("calculating free resources: %w", err)
	}

	if err := d.updateDBFreeResources(ctx, types.FreeResources{Resources: freeResources}); err != nil {
		return types.FreeResources{}, fmt.Errorf("updating free resources in db: %w", err)
	}

	return types.FreeResources{Resources: freeResources}, nil
}

// GetOnboardedResources returns the onboarded resources of the machine
func (d DefaultManager) GetOnboardedResources(ctx context.Context) (types.OnboardedResources, error) {
	onboardedResources, err := d.repos.OnboardedResources.Get(ctx)
	if err != nil {
		return types.OnboardedResources{}, fmt.Errorf("failed to get onboarded resources: %w", err)
	}
	return onboardedResources, nil
}

// GetRequiredResources returns the resources required by the jobs running on the machine
func (d DefaultManager) GetRequiredResources(ctx context.Context) (types.Resources, error) {
	jobRequirements, err := d.repos.RequiredResources.FindAll(ctx, d.repos.RequiredResources.GetQuery())
	if err != nil {
		return types.Resources{}, fmt.Errorf("unable to get resource requirements from db - %v", err)
	}

	var totalRequiredResources types.Resources
	for _, req := range jobRequirements {
		totalRequiredResources = totalRequiredResources.Add(req.Resources)
	}

	return totalRequiredResources, nil
}

// UpdateOnboardedResources updates the onboarded resources of the machine in the database
func (d DefaultManager) UpdateOnboardedResources(ctx context.Context, resources types.OnboardedResources) error {
	_, err := d.repos.OnboardedResources.Save(ctx, resources)
	if err != nil {
		return fmt.Errorf("failed to update onboarded resources: %w", err)
	}
	return nil
}

// SystemSpecs returns the SystemSpecs instance
func (d DefaultManager) SystemSpecs() SystemSpecs {
	return d.systemSpecs
}

// UsageMonitor returns the UsageMonitor instance
func (d DefaultManager) UsageMonitor() UsageMonitor {
	return d.usageMonitor
}

// updateDBFreeResources updates the free resources in the database
func (d DefaultManager) updateDBFreeResources(ctx context.Context, freeResources types.FreeResources) error {
	_, err := d.repos.FreeResources.Save(ctx, freeResources)
	if err != nil {
		return fmt.Errorf("updating free resources: %w", err)
	}
	return nil
}
