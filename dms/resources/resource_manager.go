// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package resources

import (
	"context"
	"fmt"
	"sync"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

// ManagerRepos holds all the repositories needed for resource management
type ManagerRepos struct {
	OnboardedResources repositories.OnboardedResources
	ResourceAllocation repositories.ResourceAllocation
}

// DefaultManager implements the ResourceManager interface
// TODO: Add telemetry for the methods https://gitlab.com/nunet/device-management-service/-/issues/535
type DefaultManager struct {
	repos    ManagerRepos
	store    *store
	hardware types.HardwareManager

	// allocationLock is used to synchronize access to the allocation pool during allocation and deallocation
	// it ensures that resource allocation and deallocation are atomic operations
	allocationLock sync.Mutex

	// committedLock is used to synchronize access to the committed resources pool during committing and releasing
	// it ensures that resource committing and releasing are atomic operations
	committedLock sync.Mutex
}

// NewResourceManager returns a new defaultResourceManager instance
func NewResourceManager(repos ManagerRepos, hardware types.HardwareManager) (*DefaultManager, error) {
	if hardware == nil {
		return nil, fmt.Errorf("hardware manager cannot be nil")
	}
	return &DefaultManager{
		repos:    repos,
		store:    newStore(),
		hardware: hardware,
	}, nil
}

var _ types.ResourceManager = (*DefaultManager)(nil)

// CommitResources commits the resources for an allocation
func (d *DefaultManager) CommitResources(ctx context.Context, commitment types.CommittedResources) error {
	if err := commitment.ValidateBasic(); err != nil {
		return fmt.Errorf("validating commitment: %w", err)
	}

	// Check if resources are already allocated for the allocation
	var ok bool
	d.store.withCommittedRLock(func() {
		_, ok = d.store.committedResources[commitment.AllocationID]
	})
	if ok {
		return fmt.Errorf("resources already committed for allocation %s", commitment.AllocationID)
	}

	d.committedLock.Lock()
	defer d.committedLock.Unlock()

	if err := d.checkCapacity(ctx, commitment.Resources); err != nil {
		return fmt.Errorf("checking capacity: %w", err)
	}

	// update the committed resources in the store
	d.store.withCommittedLock(func() {
		d.store.committedResources[commitment.AllocationID] = &types.CommittedResources{
			Resources:    commitment.Resources,
			AllocationID: commitment.AllocationID,
		}
	})
	return nil
}

// UncommitResources releases the committed resources for an allocation
func (d *DefaultManager) UncommitResources(_ context.Context, allocationID string) error {
	d.committedLock.Lock()
	defer d.committedLock.Unlock()
	// Check if resources are already deallocated for the allocation
	var (
		ok bool
	)
	d.store.withCommittedLock(func() {
		_, ok = d.store.committedResources[allocationID]
	})
	if !ok {
		return fmt.Errorf("resources not committed for allocation %s", allocationID)
	}

	// Release the committed resources
	d.store.withCommittedLock(func() {
		delete(d.store.committedResources, allocationID)
	})

	return nil
}

// IsCommitted checks if the resources are committed for an allocationID
func (d *DefaultManager) IsCommitted(allocationID string) (bool, error) {
	var ok bool
	d.store.withCommittedRLock(func() {
		_, ok = d.store.committedResources[allocationID]
	})
	return ok, nil
}

// AllocateResources allocates resources for an allocation
func (d *DefaultManager) AllocateResources(ctx context.Context, allocationID string) error {
	d.allocationLock.Lock()
	defer d.allocationLock.Unlock()

	// Ensure that the resources are committed for the allocation
	var (
		ok         bool
		allocation *types.CommittedResources
	)
	d.store.withCommittedRLock(func() {
		allocation, ok = d.store.committedResources[allocationID]
	})
	if !ok {
		return fmt.Errorf("resources not committed for allocation %s", allocationID)
	}

	// Check if resources are already allocated for the allocation
	d.store.withAllocationsRLock(func() {
		_, ok = d.store.allocations[allocation.AllocationID]
	})
	if ok {
		return fmt.Errorf("resources already allocated for allocation %s", allocation.AllocationID)
	}

	allocatedResource := types.ResourceAllocation{AllocationID: allocationID, Resources: allocation.Resources}
	if err := d.storeAllocation(ctx, allocatedResource); err != nil {
		return fmt.Errorf("storing allocations in db: %w", err)
	}

	// clear the committed resources
	d.store.withCommittedLock(func() {
		delete(d.store.committedResources, allocationID)
	})

	return nil
}

// DeallocateResources deallocates resources for a allocation
func (d *DefaultManager) DeallocateResources(ctx context.Context, allocationID string) error {
	d.allocationLock.Lock()
	defer d.allocationLock.Unlock()
	// Check if resources are already deallocated for the allocation
	var (
		ok bool
	)
	d.store.withAllocationsRLock(func() {
		_, ok = d.store.allocations[allocationID]
	})
	if !ok {
		return fmt.Errorf("resources not allocated for allocation %s", allocationID)
	}

	if err := d.deleteAllocation(ctx, allocationID); err != nil {
		return fmt.Errorf("deleting allocations from db: %w", err)
	}

	return nil
}

// IsAllocated checks if the resources are allocated for an allocationID
func (d *DefaultManager) IsAllocated(allocationID string) (bool, error) {
	var ok bool
	d.store.withAllocationsRLock(func() {
		_, ok = d.store.allocations[allocationID]
	})
	return ok, nil
}

// GetFreeResources returns the free resources in the allocation pool
func (d *DefaultManager) GetFreeResources(ctx context.Context) (types.FreeResources, error) {
	var freeResources types.FreeResources

	// get onboarded resources
	onboardedResources, err := d.GetOnboardedResources(ctx)
	if err != nil {
		return types.FreeResources{}, fmt.Errorf("getting onboarded resources: %w", err)
	}

	// get allocated resources
	totalAllocation, err := d.GetTotalAllocation()
	if err != nil {
		return types.FreeResources{}, fmt.Errorf("getting total allocations: %w", err)
	}

	// get committed resources
	var committedResources types.Resources
	d.store.withCommittedRLock(func() {
		for _, committedResource := range d.store.committedResources {
			_ = committedResources.Add(committedResource.Resources)
		}
	})

	log.Debugw("onboarded_resources",
		"labels", string(observability.LabelMetric),
		"resources", onboardedResources.Resources)
	log.Debugw("total_allocation",
		"labels", string(observability.LabelMetric),
		"allocation", totalAllocation)
	log.Debugw("committed_resources",
		"labels", string(observability.LabelMetric),
		"committed", committedResources)

	freeResources.Resources = onboardedResources.Resources
	if err := freeResources.Resources.Subtract(totalAllocation); err != nil {
		return types.FreeResources{}, fmt.Errorf("subtracting total allocation: %w", err)
	}

	if err := freeResources.Resources.Subtract(committedResources); err != nil {
		return types.FreeResources{}, fmt.Errorf("subtracting committed resources: %w", err)
	}

	log.Debugw("free_resources",
		"labels", string(observability.LabelMetric),
		"free", freeResources.Resources)
	return freeResources, nil
}

// GetTotalAllocation returns the total allocations of the jobs requiring resources
func (d *DefaultManager) GetTotalAllocation() (types.Resources, error) {
	var (
		totalAllocation types.Resources
		err             error
	)
	d.store.withAllocationsRLock(func() {
		for _, allocation := range d.store.allocations {
			err = totalAllocation.Add(allocation.Resources)
			if err != nil {
				break
			}
		}
	})
	return totalAllocation, err
}

// GetOnboardedResources returns the onboarded resources of the machine
func (d *DefaultManager) GetOnboardedResources(ctx context.Context) (types.OnboardedResources, error) {
	var (
		onboardedResources types.OnboardedResources
		ok                 bool
	)
	d.store.withOnboardedRLock(func() {
		if d.store.onboardedResources != nil {
			onboardedResources = *d.store.onboardedResources
			ok = true
		}
	})
	if ok {
		return onboardedResources, nil
	}

	onboardedResources, err := d.repos.OnboardedResources.Get(ctx)
	if err != nil {
		return types.OnboardedResources{}, fmt.Errorf("failed to get onboarded resources: %w", err)
	}

	_ = d.store.withOnboardedLock(func() error {
		d.store.onboardedResources = &onboardedResources
		return nil
	})
	return onboardedResources, nil
}

// UpdateOnboardedResources updates the onboarded resources of the machine in the database
func (d *DefaultManager) UpdateOnboardedResources(ctx context.Context, resources types.Resources) error {
	if err := d.store.withOnboardedLock(func() error {
		// calculate the new free resources based on the allocations
		totalAllocation, err := d.GetTotalAllocation()
		if err != nil {
			return fmt.Errorf("getting total allocations: %w", err)
		}

		// Check if the allocation is too high
		if err := resources.Subtract(totalAllocation); err != nil {
			return fmt.Errorf("couldn't subtract allocation: %w. Demand too high", err)
		}

		onboardedResources := types.OnboardedResources{Resources: resources}
		_, err = d.repos.OnboardedResources.Save(ctx, onboardedResources)
		if err != nil {
			return fmt.Errorf("failed to update onboarded resources: %w", err)
		}

		d.store.onboardedResources = &onboardedResources
		return nil
	}); err != nil {
		return err
	}

	return nil
}

// checkCapacity checks if the resources are available in the pool
func (d *DefaultManager) checkCapacity(ctx context.Context, resources types.Resources) error {
	freeResources, err := d.GetFreeResources(ctx)
	if err != nil {
		return fmt.Errorf("getting free resources: %w", err)
	}

	// Check if there are enough free resources in dms resource pool to allocate
	if err := freeResources.Subtract(resources); err != nil {
		return fmt.Errorf("no free resources: %w", err)
	}

	return nil
}

// storeAllocation stores the allocations in the database and the store
func (d *DefaultManager) storeAllocation(ctx context.Context, allocation types.ResourceAllocation) error {
	_, err := d.repos.ResourceAllocation.Create(ctx, allocation)
	if err != nil {
		return fmt.Errorf("storing allocations in db: %w", err)
	}

	d.store.withAllocationsLock(func() {
		d.store.allocations[allocation.AllocationID] = allocation
	})
	return nil
}

// deleteAllocation deletes the allocations from the database and the store
func (d *DefaultManager) deleteAllocation(ctx context.Context, allocationID string) error {
	query := d.repos.ResourceAllocation.GetQuery()
	query.Conditions = append(query.Conditions, repositories.EQ("AllocationID", allocationID))
	allocation, err := d.repos.ResourceAllocation.Find(context.Background(), query)
	if err != nil {
		return fmt.Errorf("finding allocations in db: %w", err)
	}

	if err := d.repos.ResourceAllocation.Delete(ctx, allocation.ID); err != nil {
		return fmt.Errorf("deleting allocations from db: %w", err)
	}

	d.store.withAllocationsLock(func() {
		delete(d.store.allocations, allocationID)
	})
	return nil
}
