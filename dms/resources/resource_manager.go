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
	"gitlab.com/nunet/device-management-service/types"
)

// ManagerRepos holds all the repositories needed for resource management
type ManagerRepos struct {
	FreeResources      repositories.FreeResources
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
	rmStore := newStore()
	return &DefaultManager{
		repos:    repos,
		store:    rmStore,
		hardware: hardware,
	}, nil
}

var _ types.ResourceManager = (*DefaultManager)(nil)

// CommitResources preallocates the resources required by the jobs
func (d *DefaultManager) CommitResources(ctx context.Context, allocation types.ResourceAllocation) error {
	d.committedLock.Lock()
	defer d.committedLock.Unlock()

	// Check if resources are already allocated for the job
	var ok bool
	d.store.withCommittedRLock(func() {
		_, ok = d.store.committedResources[allocation.JobID]
	})
	if ok {
		return fmt.Errorf("resources already committed for job %s", allocation.JobID)
	}

	ok = false
	d.store.withAllocationsLock(func() {
		_, ok = d.store.allocations[allocation.JobID]
	})
	if ok {
		return fmt.Errorf("resources already allocated for job %s", allocation.JobID)
	}

	freeResources, err := d.GetFreeResources(ctx)
	if err != nil {
		return fmt.Errorf("getting free resources: %w", err)
	}

	// Check if there are enough free resources in dms pool to allocate
	if err := freeResources.Subtract(allocation.Resources); err != nil {
		return fmt.Errorf("no free resources: %w", err)
	}

	// Check if there are enough free resources on the machine to preallocate
	systemFreeResources, err := d.hardware.GetFreeResources()
	if err != nil {
		return fmt.Errorf("get system free resources: %w", err)
	}

	if err := systemFreeResources.Subtract(allocation.Resources); err != nil {
		return fmt.Errorf("no free resources on the machine: %w", err)
	}

	// update the committed resources in the store
	d.store.withCommittedLock(func() {
		d.store.committedResources[allocation.JobID] = &types.CommittedResources{
			Resources: allocation.Resources,
			JobID:     allocation.JobID,
		}
	})
	if err := d.updateFreeResources(ctx, freeResources); err != nil {
		return fmt.Errorf("updating free resources in db: %w", err)
	}

	return nil
}

// ReleaseCommittedResources releases the resources that were preallocated
func (d *DefaultManager) ReleaseCommittedResources(ctx context.Context, jobID string) error {
	d.committedLock.Lock()
	defer d.committedLock.Unlock()
	// Check if resources are already deallocated for the job
	var (
		committedResources *types.CommittedResources
		ok                 bool
	)
	d.store.withCommittedLock(func() {
		committedResources, ok = d.store.committedResources[jobID]
	})
	if !ok {
		return fmt.Errorf("resources not committed for job %s", jobID)
	}

	// Get the free resources in order to update them
	freeResources, err := d.GetFreeResources(ctx)
	if err != nil {
		return fmt.Errorf("getting free resources: %w", err)
	}

	// Release the committed resources
	d.store.withCommittedLock(func() {
		delete(d.store.committedResources, jobID)
	})

	// Potential issue: if the free resources are updated in the db, the allocations should be updated as well
	// If the allocations update fails, the free resources should not be updated
	// Since we have no concept of transactions in the current implementation of db, we cannot handle this scenario
	// without writing a custom transaction manager
	if err := freeResources.Add(committedResources.Resources); err != nil {
		return fmt.Errorf("adding resources: %w", err)
	}
	if err := d.updateFreeResources(ctx, freeResources); err != nil {
		return fmt.Errorf("updating free resources in db: %w", err)
	}

	return nil
}

// AllocateResources allocates resources for a job
func (d *DefaultManager) AllocateResources(ctx context.Context, allocation types.ResourceAllocation) error {
	d.allocationLock.Lock()
	defer d.allocationLock.Unlock()

	// Check if resources are already allocated for the job
	var ok bool
	d.store.withAllocationsRLock(func() {
		_, ok = d.store.allocations[allocation.JobID]
	})
	if ok {
		return fmt.Errorf("resources already allocated for job %s", allocation.JobID)
	}

	freeResources, err := d.GetFreeResources(ctx)
	if err != nil {
		return fmt.Errorf("getting free resources: %w", err)
	}

	// Check if there are enough free resources in dms pool to allocate
	if err := freeResources.Subtract(allocation.Resources); err != nil {
		return fmt.Errorf("no free resources: %w", err)
	}

	// Check if there are enough free resources on the machine to allocate
	systemFreeResources, err := d.hardware.GetFreeResources()
	if err != nil {
		return fmt.Errorf("get system free resources: %w", err)
	}

	if err := systemFreeResources.Subtract(allocation.Resources); err != nil {
		return fmt.Errorf("no free resources on the machine: %w", err)
	}

	// Potential issue: if the free resources are updated in the db, the allocations should be updated as well
	// If the allocations update fails, the free resources should not be updated
	// Since we have no concept of transactions in the current implementation of db, we cannot handle this scenario
	// without writing a custom transaction manager
	if err := d.updateFreeResources(ctx, freeResources); err != nil {
		return fmt.Errorf("updating free resources in db: %w", err)
	}
	if err := d.storeAllocation(ctx, allocation); err != nil {
		return fmt.Errorf("storing allocations in db: %w", err)
	}

	return nil
}

// DeallocateResources deallocates resources for a job
func (d *DefaultManager) DeallocateResources(ctx context.Context, jobID string) error {
	d.allocationLock.Lock()
	defer d.allocationLock.Unlock()
	// Check if resources are already deallocated for the job
	var (
		allocation types.ResourceAllocation
		ok         bool
	)
	d.store.withAllocationsRLock(func() {
		allocation, ok = d.store.allocations[jobID]
	})
	if !ok {
		return fmt.Errorf("resources not allocated for job %s", jobID)
	}

	// Get the free resources in order to update them
	freeResources, err := d.GetFreeResources(ctx)
	if err != nil {
		return fmt.Errorf("getting free resources: %w", err)
	}

	// Deallocate the resources

	// Potential issue: if the free resources are updated in the db, the allocations should be updated as well
	// If the allocations update fails, the free resources should not be updated
	// Since we have no concept of transactions in the current implementation of db, we cannot handle this scenario
	// without writing a custom transaction manager
	if err := freeResources.Add(allocation.Resources); err != nil {
		return fmt.Errorf("adding resources: %w", err)
	}
	if err := d.updateFreeResources(ctx, freeResources); err != nil {
		return fmt.Errorf("updating free resources in db: %w", err)
	}
	if err := d.deleteAllocation(ctx, jobID); err != nil {
		return fmt.Errorf("deleting allocations from db: %w", err)
	}

	return nil
}

// GetFreeResources returns the free resources in the allocation pool
func (d *DefaultManager) GetFreeResources(ctx context.Context) (types.FreeResources, error) {
	var (
		freeResources types.FreeResources
		ok            bool
	)

	d.store.withFreeRLock(func() {
		if d.store.freeResources != nil {
			freeResources = *d.store.freeResources
			ok = true
		}
	})
	if ok {
		return freeResources, nil
	}

	freeResources, err := d.repos.FreeResources.Get(ctx)
	if err != nil {
		return types.FreeResources{}, fmt.Errorf("failed to get free resources: %w", err)
	}

	d.store.withFreeLock(func() {
		d.store.freeResources = &freeResources
	})

	return freeResources, nil
}

// GetTotalAllocation returns the total allocations of the jobs requiring resources
func (d *DefaultManager) GetTotalAllocation() (types.Resources, error) {
	if len(d.store.allocations) == 0 {
		if err := d.getAllocationsFromDB(context.Background()); err != nil {
			return types.Resources{}, fmt.Errorf("getting allocations from db: %w", err)
		}
	}

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

		onboardedResources := types.OnboardedResources{Resources: resources}

		// Check if the demand is too high
		if err := resources.Subtract(totalAllocation); err != nil {
			return fmt.Errorf("couldn't subtract allocation: %w. Demand too high", err)
		}

		// Potential issue: if the onboarded resources are updated in the db, the free resources should be updated as well
		// If the free resources update fails, the onboarded resources should not be updated
		// Since we have no concept of transactions in the current implementation of db, we cannot handle this scenario
		// without writing a custom transaction manager
		_, err = d.repos.OnboardedResources.Save(ctx, onboardedResources)
		if err != nil {
			return fmt.Errorf("failed to update onboarded resources: %w", err)
		}

		d.store.onboardedResources = &onboardedResources
		if err := d.updateFreeResources(ctx, types.FreeResources{
			Resources: resources,
		}); err != nil {
			return fmt.Errorf("updating free resources in db: %w", err)
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// updateFreeResources updates the free resources in the database and the store
func (d *DefaultManager) updateFreeResources(ctx context.Context, freeResources types.FreeResources) error {
	_, err := d.repos.FreeResources.Save(ctx, freeResources)
	if err != nil {
		return fmt.Errorf("updating free resources: %w", err)
	}

	// update the free resources in the store
	d.store.withFreeLock(func() {
		d.store.freeResources = &freeResources
	})
	return nil
}

// getAllocationsFromDB fetches the allocations from the database
func (d *DefaultManager) getAllocationsFromDB(ctx context.Context) error {
	allocations, err := d.repos.ResourceAllocation.FindAll(ctx, d.repos.ResourceAllocation.GetQuery())
	if err != nil {
		return fmt.Errorf("getting allocations from db: %w", err)
	}
	d.store.withAllocationsLock(func() {
		for _, allocation := range allocations {
			d.store.allocations[allocation.JobID] = allocation
		}
	})
	return nil
}

// storeAllocation stores the allocations in the database and the store
func (d *DefaultManager) storeAllocation(ctx context.Context, allocation types.ResourceAllocation) error {
	_, err := d.repos.ResourceAllocation.Create(ctx, allocation)
	if err != nil {
		return fmt.Errorf("storing allocations in db: %w", err)
	}

	d.store.withAllocationsLock(func() {
		d.store.allocations[allocation.JobID] = allocation
	})
	return nil
}

// deleteAllocation deletes the allocations from the database and the store
func (d *DefaultManager) deleteAllocation(ctx context.Context, jobID string) error {
	query := d.repos.ResourceAllocation.GetQuery()
	query.Conditions = append(query.Conditions, repositories.EQ("JobID", jobID))
	allocation, err := d.repos.ResourceAllocation.Find(context.Background(), query)
	if err != nil {
		return fmt.Errorf("finding allocations in db: %w", err)
	}

	if err := d.repos.ResourceAllocation.Delete(ctx, allocation.ID); err != nil {
		return fmt.Errorf("deleting allocations from db: %w", err)
	}

	d.store.withAllocationsLock(func() {
		delete(d.store.allocations, jobID)
	})
	return nil
}
