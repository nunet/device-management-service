package types

import (
	"context"
	"fmt"
)

// Resources represents the resources of the machine
type Resources struct {
	CPU  CPU  `gorm:"embedded;embeddedPrefix:cpu_"`
	GPUs GPUs `gorm:"foreignKey:ResourceID"`
	RAM  RAM  `gorm:"embedded;embeddedPrefix:ram_"`
	Disk Disk `gorm:"embedded;embeddedPrefix:disk_"`
}

// implements the Calculable and Comparable interfaces
var (
	_ Calculable[Resources] = (*Resources)(nil)
	_ Comparable[Resources] = (*Resources)(nil)
)

// Compare compares two Resources objects
func (r *Resources) Compare(other Resources) (Comparison, error) {
	cpuComp, err := r.CPU.Compare(other.CPU)
	if err != nil {
		return None, fmt.Errorf("error comparing CPU: %v", err)
	}

	ramComp, err := r.RAM.Compare(other.RAM)
	if err != nil {
		return None, fmt.Errorf("error comparing RAM: %v", err)
	}

	diskComp, err := r.Disk.Compare(other.Disk)
	if err != nil {
		return None, fmt.Errorf("error comparing Disk: %v", err)
	}

	gpuComp, err := r.GPUs.Compare(other.GPUs)
	if err != nil {
		return None, fmt.Errorf("error comparing GPUs: %v", err)
	}

	return cpuComp.And(ramComp).And(diskComp).And(gpuComp), nil
}

// Equal returns true if the resources are equal
func (r *Resources) Equal(other Resources) bool {
	if r.RAM.Size != other.RAM.Size {
		return false
	}

	if r.CPU.Cores != other.CPU.Cores {
		return false
	}

	if r.Disk.Size != other.Disk.Size {
		return false
	}

	return true
}

// Add returns the sum of the resources
func (r *Resources) Add(other Resources) error {
	if err := r.CPU.Add(other.CPU); err != nil {
		return fmt.Errorf("error adding CPU: %v", err)
	}

	if err := r.RAM.Add(other.RAM); err != nil {
		return fmt.Errorf("error adding RAM: %v", err)
	}

	if err := r.Disk.Add(other.Disk); err != nil {
		return fmt.Errorf("error adding Disk: %v", err)
	}

	if err := r.GPUs.Add(other.GPUs); err != nil {
		return fmt.Errorf("error adding GPUs: %v", err)
	}

	return nil
}

// Subtract returns the difference of the resources
func (r *Resources) Subtract(other Resources) error {
	if err := r.CPU.Subtract(other.CPU); err != nil {
		return fmt.Errorf("error subtracting CPU: %v", err)
	}

	if err := r.RAM.Subtract(other.RAM); err != nil {
		return fmt.Errorf("error subtracting RAM: %v", err)
	}

	if err := r.Disk.Subtract(other.Disk); err != nil {
		return fmt.Errorf("error subtracting Disk: %v", err)
	}

	if err := r.GPUs.Subtract(other.GPUs); err != nil {
		return fmt.Errorf("error subtracting GPUs: %v", err)
	}

	return nil
}

// MachineResources represents the total resources of the machine
type MachineResources struct {
	BaseDBModel
	Resources
}

// FreeResources represents the free resources of the machine
type FreeResources struct {
	BaseDBModel
	Resources
}

// OnboardedResources represents the onboarded resources of the machine
type OnboardedResources struct {
	BaseDBModel
	Resources
}

// ResourceAllocation represents the allocation of resources for a job
type ResourceAllocation struct {
	BaseDBModel
	JobID string
	Resources
}

// ResourceManager is an interface that defines the methods to manage the resources of the machine
type ResourceManager interface {
	// AllocateResources allocates the resources required by a job
	AllocateResources(context.Context, ResourceAllocation) error
	// DeallocateResources deallocates the resources required by a job
	DeallocateResources(context.Context, string) error
	// GetTotalAllocation returns the total allocations for the jobs
	GetTotalAllocation() (Resources, error)
	// GetFreeResources returns the free resources in the allocation pool
	GetFreeResources(ctx context.Context) (FreeResources, error)
	// GetOnboardedResources returns the onboarded resources of the machine
	GetOnboardedResources(context.Context) (OnboardedResources, error)
	// UpdateOnboardedResources updates the onboarded resources of the machine in the database
	UpdateOnboardedResources(context.Context, Resources) error
}
