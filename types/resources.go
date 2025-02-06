// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package types

import (
	"context"
	"fmt"
)

// Resources represents the resources of the machine
type Resources struct {
	CPU  CPU  `json:"cpu"`
	GPUs GPUs `json:"gpus,omitempty"`
	RAM  RAM  `json:"ram"`
	Disk Disk `json:"disk"`
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

	if !r.GPUs.Equal(other.GPUs) {
		return false
	}

	return true
}

// Add returns the sum of the resources
func (r *Resources) Add(other Resources) error {
	log.Debugf("adding resources: %+v + %+v", r, other)
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
	log.Debugf("subtracting resources: %+v - %+v", r, other)
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

// CommittedResources represents the committed resources of the machine
type CommittedResources struct {
	BaseDBModel
	Resources
	AllocationID string `json:"allocationID"`
}

// OnboardedResources represents the onboarded resources of the machine
type OnboardedResources struct {
	BaseDBModel
	Resources
}

// ResourceAllocation represents the allocation of resources for a job
type ResourceAllocation struct {
	BaseDBModel
	AllocationID string `json:"allocation_id"`
	Resources
}

// ResourceManager is an interface that defines the methods to manage the resources of the machine
type ResourceManager interface {
	// CommitResources commits the resources required by the allocation
	CommitResources(context.Context, CommittedResources) error
	// UncommitResources releases the resources that were committed for the allocation
	UncommitResources(context.Context, string) error
	// IsCommitted returns true if the resources are committed for the allocation
	IsCommitted(string) (bool, error)
	// AllocateResources allocates the resources required by an allocation
	AllocateResources(context.Context, string) error
	// DeallocateResources deallocates the resources required by an allocation
	DeallocateResources(context.Context, string) error
	// IsAllocated returns true if the resources are allocated for the allocation
	IsAllocated(allocationID string) (bool, error)
	// GetTotalAllocation returns the total allocations for the allocation
	GetTotalAllocation() (Resources, error)
	// GetFreeResources returns the free resources in the pool
	GetFreeResources(ctx context.Context) (FreeResources, error)
	// GetOnboardedResources returns the onboarded resources of the machine
	GetOnboardedResources(context.Context) (OnboardedResources, error)
	// UpdateOnboardedResources updates the onboarded resources of the machine in the database
	UpdateOnboardedResources(context.Context, Resources) error
}
