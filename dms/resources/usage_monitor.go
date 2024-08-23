package resources

import (
	"context"
	"fmt"

	"gitlab.com/nunet/device-management-service/types"

	"gitlab.com/nunet/device-management-service/db/repositories"
)

// UsageMonitor defines the methods to monitor the system usage
type UsageMonitor interface {
	// GetUsage returns the resources used by the machine
	GetUsage(context.Context) (types.Resources, error)
}

// defaultUsageMonitor implements the UsageMonitor interface
type defaultUsageMonitor struct {
	systemSpecs           SystemSpecs
	vmRepo                repositories.VirtualMachine
	serviceRepo           repositories.Services
	requiredResourcesRepo repositories.RequiredResources
}

// newUsageMonitor creates a new defaultUsageMonitor
func newUsageMonitor(
	systemSpecs SystemSpecs,
	vmRepo repositories.VirtualMachine,
	serviceRepo repositories.Services,
	requiredResourcesRepo repositories.RequiredResources,
) *defaultUsageMonitor {
	return &defaultUsageMonitor{
		systemSpecs:           systemSpecs,
		vmRepo:                vmRepo,
		serviceRepo:           serviceRepo,
		requiredResourcesRepo: requiredResourcesRepo,
	}
}

var _ UsageMonitor = (*defaultUsageMonitor)(nil)

// GetUsage returns the resources used by the machine
func (um *defaultUsageMonitor) GetUsage(ctx context.Context) (types.Resources, error) {
	cpuInfo, err := um.systemSpecs.GetCPUInfo()
	if err != nil {
		return types.Resources{}, fmt.Errorf("getting CPU info: %w", err)
	}

	vmUsage, err := um.getVMUsage(ctx, cpuInfo)
	if err != nil {
		return types.Resources{}, fmt.Errorf("getting VM usage: %w", err)
	}

	contUsage, err := um.getContainerUsage(ctx)
	if err != nil {
		return types.Resources{}, fmt.Errorf("getting container usage: %w", err)
	}

	return vmUsage.Add(contUsage), nil
}

// getVMUsage returns the total usage of all running VMs
func (um *defaultUsageMonitor) getVMUsage(ctx context.Context, cpuInfo types.CPUInfo) (types.Resources, error) {
	query := um.vmRepo.GetQuery()
	query.Conditions = append(query.Conditions, repositories.EQ("State", "running"))
	vms, err := um.vmRepo.FindAll(ctx, query)
	if err != nil {
		return types.Resources{}, fmt.Errorf("unable to get running VMs: %w", err)
	}

	var resourcesUsage types.Resources
	if len(vms) == 0 {
		return resourcesUsage, nil
	}

	// TODO: disk usage
	var totalVCPU, totalMemSizeMib uint
	for _, vm := range vms {
		totalVCPU += vm.VCPUCount
		totalMemSizeMib += vm.MemSizeMib
	}
	resourcesUsage.RAM = uint64(totalMemSizeMib)
	resourcesUsage.CPU = float64(totalVCPU) * cpuInfo.MHzPerCore // CPU in MHz
	return resourcesUsage, nil
}

// getContainerUsage returns the total usage of all running containers
func (um *defaultUsageMonitor) getContainerUsage(ctx context.Context) (types.Resources, error) {
	query := um.serviceRepo.GetQuery()
	query.Conditions = append(query.Conditions, repositories.EQ("JobStatus", "running"))
	services, err := um.serviceRepo.FindAll(ctx, query)
	if err != nil {
		return types.Resources{}, fmt.Errorf("unable to get running containers: %w", err)
	}

	var resourcesUsage types.Resources
	if len(services) == 0 {
		return resourcesUsage, nil
	}

	requirements, err := um.getResourceRequirements(ctx)
	if err != nil {
		return types.Resources{}, fmt.Errorf("unable to get resource requirements: %w", err)
	}

	// TODO: disk usage
	for _, service := range services {
		resourcesReq := requirements[service.ResourceRequirements]
		resourcesUsage.CPU += resourcesReq.CPU
		resourcesUsage.RAM += resourcesReq.RAM
	}

	return resourcesUsage, nil
}

// getResourceRequirements returns the resource requirements of all jobs
func (um *defaultUsageMonitor) getResourceRequirements(ctx context.Context) (map[int]types.RequiredResources, error) {
	requiredResources, err := um.requiredResourcesRepo.FindAll(ctx, um.requiredResourcesRepo.GetQuery())
	if err != nil {
		return nil, fmt.Errorf("unable to query resource requirements: %w", err)
	}

	mappedRequiredResources := make(map[int]types.RequiredResources)
	for _, rr := range requiredResources {
		mappedRequiredResources[rr.JobID] = rr
	}

	return mappedRequiredResources, nil
}
