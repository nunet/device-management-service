// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package node

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/storage"
	"gitlab.com/nunet/device-management-service/storage/volume"

	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/actor"

	"gitlab.com/nunet/device-management-service/dms/jobs"

	"gitlab.com/nunet/device-management-service/types"

	"gitlab.com/nunet/device-management-service/network/utils"
)

var (
	// ErrAllocationNotFound is returned when an allocation is not found but is expected.
	ErrAllocationNotFound = fmt.Errorf("allocation not found")
	// ErrDynamicPortsNotAvailable is returned when dynamic ports are not available for allocation.
	ErrDynamicPortsNotAvailable = fmt.Errorf("dynamic ports not available")
	// ErrPortsBusy is returned when the requested ports are already allocated.
	ErrPortsBusy = fmt.Errorf("ports are already allocated")
	// ErrResourcesNotAvailable is returned when the requested resources are not available.
	ErrResourcesNotAvailable = fmt.Errorf("resources not available")
	// ErrNoHardwareCapacity is returned when there is no capacity left on the hardware.
	ErrNoHardwareCapacity = fmt.Errorf("no capacity left on the hardware")
)

// portAllocator keeps track of port allocations and manages state.
type portAllocator struct {
	config PortConfig

	lock        sync.Mutex
	allocations map[string][]int // allocationID -> ports
	reserved    map[int]struct{} // reserved ports
}

// newPortAllocator initializes a new portAllocator with a PortConfig.
func newPortAllocator(config PortConfig) *portAllocator {
	return &portAllocator{
		config:      config,
		allocations: make(map[string][]int),
		reserved:    make(map[int]struct{}),
	}
}

// allocate a helper function to allocate a port.
func (pa *portAllocator) allocate(port int) error {
	if port < pa.config.AvailableRangeFrom || port > pa.config.AvailableRangeTo {
		return fmt.Errorf("port %d is outside allowed range [%d-%d]",
			port, pa.config.AvailableRangeFrom, pa.config.AvailableRangeTo)
	}

	if _, reserved := pa.reserved[port]; reserved {
		return fmt.Errorf("port %d is already reserved", port)
	}

	if !utils.IsFreePort(port) {
		return fmt.Errorf("port %d is not free", port)
	}

	pa.reserved[port] = struct{}{}
	return nil
}

// Allocate allocates the requested ports, associating them with an allocationID.
// If it's not possible to allocate one of the ports, an error is returned and no ports are allocated.
func (pa *portAllocator) Allocate(allocationID string, ports []int) error {
	pa.lock.Lock()
	defer pa.lock.Unlock()

	if len(ports) == 0 {
		return nil
	}

	for _, port := range ports {
		if err := pa.allocate(port); err != nil {
			pa.release(ports)
			return fmt.Errorf("cannot allocate port %d: %w", port, err)
		}
	}

	pa.allocations[allocationID] = ports
	return nil
}

// getAvailablePorts returns a list of available ports in the range specified in the config.
func (pa *portAllocator) getAvailablePorts(numPorts int) []int {
	ports := make([]int, 0, numPorts)

	for port := pa.config.AvailableRangeFrom; port <= pa.config.AvailableRangeTo && len(ports) < numPorts; port++ {
		// Skip if port is reserved
		if _, reserved := pa.reserved[port]; reserved {
			continue
		}

		// Check if port is actually free on the system
		if utils.IsFreePort(port) {
			ports = append(ports, port)
		}
	}

	return ports
}

// AllocateRandom allocates the requested number of ports and associates them with the allocation ID.
func (pa *portAllocator) AllocateRandom(allocationID string, numPorts int) ([]int, error) {
	pa.lock.Lock()
	defer pa.lock.Unlock()

	if numPorts == 0 {
		return nil, fmt.Errorf("cannot allocate 0 ports")
	}

	portsToAllocate := pa.getAvailablePorts(numPorts)

	if len(portsToAllocate) != numPorts {
		pa.release(portsToAllocate)
		return nil, fmt.Errorf("failed to allocate %d ports", numPorts)
	}

	// allocate them
	for _, port := range portsToAllocate {
		if err := pa.allocate(port); err != nil {
			pa.release(portsToAllocate)
			return nil, fmt.Errorf("failed to allocate port %d: %w", port, err)
		}
	}

	pa.allocations[allocationID] = portsToAllocate
	return portsToAllocate, nil
}

// release a helper function to release ports.
func (pa *portAllocator) release(ports []int) {
	for _, p := range ports {
		if _, ok := pa.reserved[p]; !ok {
			continue
		}
		delete(pa.reserved, p)
	}
}

// Release releases the ports associated with the allocation ID.
func (pa *portAllocator) Release(allocationID string) {
	pa.lock.Lock()
	defer pa.lock.Unlock()

	allocated, ok := pa.allocations[allocationID]
	if !ok {
		return
	}

	pa.release(allocated)
	delete(pa.allocations, allocationID)
}

// GetAllocation returns the allocated ports for a specific allocation ID.
func (pa *portAllocator) GetAllocation(allocationID string) ([]int, error) {
	ports, exists := pa.allocations[allocationID]
	if !exists {
		return nil, fmt.Errorf("port allocation ID not found: %s", allocationID)
	}
	return ports, nil
}

// isAllocated checks if the given ports are already allocated.
func (pa *portAllocator) isAllocated(ports []int) bool {
	for _, port := range ports {
		if _, reserved := pa.reserved[port]; reserved {
			return true
		}
	}

	return false
}

// portsAvailable checks if the requested number of ports are available.
func (pa *portAllocator) portsAvailable(numPorts int) bool {
	pa.lock.Lock()
	defer pa.lock.Unlock()

	ports := pa.getAvailablePorts(numPorts)

	return len(ports) == numPorts
}

// Allocator is the interface for the node allocator.
// It is responsible for managing resources and allocations.
type Allocator interface {
	// Run starts the allocator.
	Run() error
	// Commit commits resources and ports for an allocation.
	Commit(ctx context.Context,
		allocationID string,
		resources types.CommittedResources,
		ports map[int]int,
		numDynamicPorts int,
		expiry int64,
	) error
	// Uncommit uncommits resources and ports for an allocation.
	Uncommit(ctx context.Context, allocationID string) error
	// Allocate allocates resources and ports for an allocation.
	Allocate(
		ctx context.Context,
		allocationID string,
		actr actor.Actor,
		job jobs.Job,
		executor types.Executor,
	) (*jobs.Allocation, error)
	// Release releases allocated resources and ports for an allocation.
	Release(ctx context.Context, allocationID string) error
	// Stop stops the allocator.
	Stop(ctx context.Context) error

	// CheckAvailability checks if the requested resources and ports are available.
	CheckAvailability(ports []int, numDynamicPorts int, resources types.Resources) error
	// GetAllocations returns all allocations.
	GetAllocations() map[string]*jobs.Allocation
	// GetAllocation returns a specific allocation.
	GetAllocation(allocationID string) (*jobs.Allocation, error)
}

// allocator is the implementation of the Allocator interface.
type allocator struct {
	network   network.Network
	ports     *portAllocator
	resources types.ResourceManager
	hardware  types.HardwareManager

	allocations map[string]*jobs.Allocation
	commits     map[string]int64
	workDir     string
	hostID      string

	lock   sync.Mutex
	fs     afero.Afero
	ctx    context.Context
	cancel context.CancelFunc

	volumeTracker *storage.VoumeTracker
}

var _ Allocator = (*allocator)(nil)

// newAllocator returns a new default allocator
func newAllocator(
	vt *storage.VoumeTracker,
	portAllocator *portAllocator,
	resourceManager types.ResourceManager,
	hardwareManager types.HardwareManager,
	network network.Network,
	fs afero.Afero,
	workDir,
	hostID string,
) *allocator {
	ctx, cancel := context.WithCancel(context.Background())
	return &allocator{
		ports:         portAllocator,
		resources:     resourceManager,
		hardware:      hardwareManager,
		network:       network,
		allocations:   make(map[string]*jobs.Allocation),
		fs:            fs,
		workDir:       workDir,
		hostID:        hostID,
		commits:       make(map[string]int64),
		ctx:           ctx,
		cancel:        cancel,
		volumeTracker: vt,
	}
}

func (a *allocator) Run() error {
	// start a ticker to clear the commits after expiry
	a.clearCommits()

	return nil
}

func (a *allocator) Commit(ctx context.Context,
	allocationID string,
	resources types.CommittedResources,
	ports map[int]int,
	numDynamicPorts int,
	expiry int64,
) error {
	// Check against the actual hardware usage to ensure dms can guarantee the commitment
	hasCapacity, err := a.hardware.CheckCapacity(resources.Resources)
	if err != nil {
		return fmt.Errorf("check hardware capacity: %w", err)
	}

	if !hasCapacity {
		return ErrNoHardwareCapacity
	}

	// commit the resources
	if err := a.resources.CommitResources(ctx, resources); err != nil {
		return fmt.Errorf("commit resources: %w", err)
	}

	// commit the ports
	if len(ports) > 0 {
		for port := range ports {
			err := a.ports.Allocate(allocationID, []int{port})
			if err != nil {
				return fmt.Errorf("allocate port: %w", err)
			}
		}
	}

	// commit dynamic ports
	if numDynamicPorts > 0 {
		_, err := a.ports.AllocateRandom(allocationID, numDynamicPorts)
		if err != nil {
			return fmt.Errorf("failed to allocate ports: %w", err)
		}
	}

	// store the commit
	a.lock.Lock()
	a.commits[allocationID] = expiry
	a.lock.Unlock()

	return nil
}

func (a *allocator) Uncommit(ctx context.Context, allocationID string) error {
	// Check if the allocation is committed
	if _, ok := a.commits[allocationID]; !ok {
		log.Warnf("allocation %s not committed", allocationID)
		return nil
	}

	// uncommit the resources
	err := a.resources.UncommitResources(ctx, allocationID)
	if err != nil {
		return fmt.Errorf("uncommit resources: %w", err)
	}

	// uncommit the ports
	a.ports.Release(allocationID)

	// remove the commit from the state
	a.lock.Lock()
	delete(a.commits, allocationID)
	a.lock.Unlock()

	return nil
}

func (a *allocator) mountVolumeOnHost(job jobs.Job, allocationID string) error {
	if job.Volume != nil {
		mounter, err := volume.New(a.volumeTracker, *job.Volume)
		if err != nil {
			return fmt.Errorf("create volume: %w", err)
		}

		desginationPath := filepath.Join(a.workDir, "volumes", allocationID, job.Volume.Name)
		err = createDirIfNotExists(desginationPath)
		if err != nil {
			return fmt.Errorf("mount directory: %w", err)
		}

		err = mounter.Mount(desginationPath, nil)
		if err != nil {
			return fmt.Errorf("failed to mount volume: %w", err)
		}
	}

	return nil
}

func createDirIfNotExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, 0o755) // Creates parent directories if needed
		if err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}
	return nil
}

func (a *allocator) Allocate(
	ctx context.Context,
	allocationID string,
	allocActor actor.Actor,
	job jobs.Job,
	executor types.Executor,
) (*jobs.Allocation, error) {
	// Ensure that the allocation is committed
	if _, ok := a.commits[allocationID]; !ok {
		return nil, fmt.Errorf("allocation not committed: %s", allocationID)
	}

	// Check against the actual hardware usage to ensure dms can guarantee the allocation
	hasCapacity, err := a.hardware.CheckCapacity(job.Resources)
	if err != nil {
		return nil, fmt.Errorf("check hardware capacity: %w", err)
	}

	if !hasCapacity {
		return nil, ErrNoHardwareCapacity
	}

	err = a.mountVolumeOnHost(job, allocationID)
	if err != nil {
		return nil, err
	}

	// allocate the resources
	err = a.resources.AllocateResources(ctx, allocationID)
	if err != nil {
		return nil, fmt.Errorf("allocate resources: %w", err)
	}

	allocation, err := jobs.NewAllocation(
		allocationID,
		a.fs,
		a.workDir,
		allocActor,
		jobs.AllocationDetails{Job: job, NodeID: a.hostID},
		a.network,
		executor,
	)
	if err != nil {
		return nil, fmt.Errorf("create allocation: %w", err)
	}

	// start the allocation
	err = allocation.Start()
	if err != nil {
		return nil, fmt.Errorf("start allocation: %w", err)
	}

	// delete the commit and store the allocation
	a.lock.Lock()
	delete(a.commits, allocationID)
	a.allocations[allocationID] = allocation
	a.lock.Unlock()

	return allocation, nil
}

func (a *allocator) Release(ctx context.Context, allocationID string) error {
	// Check if allocated
	allocation, ok := a.allocations[allocationID]
	if !ok {
		log.Warnf("allocation %s not found", allocationID)
		return nil
	}

	// stop and cleanup
	err := allocation.Terminate(ctx)
	if err != nil {
		return fmt.Errorf("terminate allocation: %w", err)
	}

	// deallocate the resources and ports
	a.ports.Release(allocationID)
	err = a.resources.DeallocateResources(ctx, allocationID)
	if err != nil {
		return fmt.Errorf("deallocate resources for allocation id: %s: %w", allocationID, err)
	}

	// remove the allocation
	a.lock.Lock()
	delete(a.allocations, allocationID)
	a.lock.Unlock()

	return nil
}

func (a *allocator) Stop(ctx context.Context) error {
	// stop all allocations
	for id, allocation := range a.allocations {
		err := allocation.Stop(ctx)
		if err != nil {
			log.Warnf("stop allocation %s: %v", id, err)
		}
	}

	// clear the commits
	for allocationID := range a.commits {
		err := a.Uncommit(context.Background(), allocationID)
		if err != nil {
			log.Warnf("uncommit allocation %s: %v", allocationID, err)
		}
	}

	// cancel the context to stop the allocator goroutines
	a.cancel()
	return nil
}

func (a *allocator) CheckAvailability(ports []int, numDynamicPorts int, resources types.Resources) error {
	// Check if the requested ports are already allocated
	if a.ports.isAllocated(ports) {
		return ErrPortsBusy
	}

	// Check if the requested dynamic ports are available
	if !a.ports.portsAvailable(numDynamicPorts) {
		return ErrDynamicPortsNotAvailable
	}

	// Check if the requested resources are available
	freeResources, err := a.resources.GetFreeResources(context.Background())
	if err != nil {
		return fmt.Errorf("get free resources: %w", err)
	}
	if err := freeResources.Subtract(resources); err != nil {
		return ErrResourcesNotAvailable
	}

	return nil
}

func (a *allocator) GetAllocations() map[string]*jobs.Allocation {
	a.lock.Lock()
	defer a.lock.Unlock()

	allocations := make(map[string]*jobs.Allocation, len(a.allocations))
	for k, v := range a.allocations {
		allocations[k] = v
	}

	return allocations
}

func (a *allocator) GetAllocation(allocationID string) (*jobs.Allocation, error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	allocation, ok := a.allocations[allocationID]
	if !ok {
		return nil, ErrAllocationNotFound
	}

	return allocation, nil
}

func (a *allocator) clearCommits() {
	ticker := time.NewTicker(clearCommitsFrequency)
	go func() {
		select {
		case <-ticker.C:
			for allocationID, expiry := range a.commits {
				if expiry < time.Now().Unix() {
					err := a.Uncommit(context.Background(), allocationID)
					if err != nil {
						log.Warnf("uncommit allocation %s: %v", allocationID, err)
					}
				}
			}
		case <-a.ctx.Done():
			ticker.Stop()
			return
		}
	}()
}
