package resources

import (
	"sync"

	"gitlab.com/nunet/device-management-service/types"
)

// locks holds the locks for the resource manager
// allocations: lock for the allocations map
// onboarded: lock for the onboarded resources
// free: lock for the free resources
type locks struct {
	allocations sync.RWMutex
	onboarded   sync.RWMutex
	free        sync.RWMutex
}

// newLocks returns a new locks instance
func newLocks() *locks {
	return &locks{}
}

// store holds the resources of the machine
// onboardedResources: resources that are onboarded to the machine
// freeResources: resources that are free to be allocated
// allocations: resources that are requested by the jobs
type store struct {
	onboardedResources *types.OnboardedResources
	freeResources      *types.FreeResources
	allocations        map[string]types.ResourceAllocation
	gpuMetadata        map[types.GPUVendor][]gpuMetadata
	machineResources   *types.MachineResources

	locks *locks
}

// newStore returns a new store instance
func newStore() *store {
	return &store{
		allocations: make(map[string]types.ResourceAllocation),
		gpuMetadata: make(map[types.GPUVendor][]gpuMetadata),
		locks:       newLocks(),
	}
}

// withAllocationsLock locks the allocations lock and executes the function
func (s *store) withAllocationsLock(fn func()) {
	s.locks.allocations.Lock()
	defer s.locks.allocations.Unlock()
	fn()
}

// withOnboardedLock locks the onboarded lock and executes the function
func (s *store) withOnboardedLock(fn func() error) error {
	s.locks.onboarded.Lock()
	defer s.locks.onboarded.Unlock()
	return fn()
}

// withFreeLock locks the free lock and executes the function
func (s *store) withFreeLock(fn func()) {
	s.locks.free.Lock()
	defer s.locks.free.Unlock()
	fn()
}

// withGpuMetadataLock locks the gpu metadata lock and executes the function
func (s *store) withGpuMetadataLock(fn func()) {
	s.locks.allocations.RLock()
	defer s.locks.allocations.Unlock()
	fn()
}

// withMachineResourcesLock locks the machine resources lock and executes the function
func (s *store) withMachineResourcesLock(fn func()) {
	s.locks.allocations.Lock()
	defer s.locks.allocations.Unlock()
	fn()
}

// withAllocationsRLock performs a read lock and returns the result and error
func (s *store) withAllocationsRLock(fn func()) {
	s.locks.allocations.RLock()
	defer s.locks.allocations.RUnlock()
	fn()
}

// withOnboardedRLock performs a read lock and returns the result and error
func (s *store) withOnboardedRLock(fn func()) {
	s.locks.onboarded.RLock()
	defer s.locks.onboarded.RUnlock()
	fn()
}

// withFreeRLock performs a read lock and returns the result and error
func (s *store) withFreeRLock(fn func()) {
	s.locks.free.RLock()
	defer s.locks.free.RUnlock()
	fn()
}

// withGpuMetadataLock locks the gpu metadata lock and executes the function
// commenting out this function as it is not used but will be used in the future
// func (s *store) withGpuMetadataRLock(fn func() map[types.GPUVendor][]gpuMetadata) map[types.GPUVendor][]gpuMetadata {
//	 s.locks.allocations.RLock()
//	 defer s.locks.allocations.RUnlock()
//	 return fn()
// }

// withMachineResourcesRLock locks the machine resources lock and executes the function
func (s *store) withMachineResourcesRLock(fn func()) {
	s.locks.allocations.RLock()
	defer s.locks.allocations.RUnlock()
	fn()
}
