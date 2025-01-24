// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package node

import (
	"fmt"
	"sync"

	"gitlab.com/nunet/device-management-service/network/utils"
)

// PortAllocator keeps track of port allocations and manages state.
type PortAllocator struct {
	config PortConfig

	mx          sync.Mutex
	allocations map[string][]int
	reserved    map[int]struct{}
}

// NewPortAllocator initializes a new PortAllocator with a PortConfig.
func NewPortAllocator(config PortConfig) *PortAllocator {
	return &PortAllocator{
		config:      config,
		allocations: make(map[string][]int),
		reserved:    make(map[int]struct{}),
	}
}

// AllocatePorts allocates the requested ports, associating them with an allocation.
// If it's not possible to allocate one of the ports, an error is returned and no ports are allocated.
func (pa *PortAllocator) AllocatePorts(allocationID string, ports []int) error {
	pa.mx.Lock()
	defer pa.mx.Unlock()

	if len(ports) == 0 {
		return fmt.Errorf("cannot allocate 0 ports")
	}

	for _, port := range ports {
		if err := pa.allocate(port); err != nil {
			pa.releasePorts(ports)
			return fmt.Errorf("cannot allocate port %d: %w", port, err)
		}
	}

	pa.allocations[allocationID] = ports
	return nil
}

// AllocateRandom allocates the requested number of ports and associates them with the allocation ID.
func (pa *PortAllocator) AllocateRandom(allocationID string, numPorts int) ([]int, error) {
	pa.mx.Lock()
	defer pa.mx.Unlock()

	if numPorts == 0 {
		return nil, fmt.Errorf("cannot allocate 0 ports")
	}

	portsToAllocate := pa.getAvailablePorts(numPorts)

	if len(portsToAllocate) != numPorts {
		pa.releasePorts(portsToAllocate)
		return nil, fmt.Errorf("failed to allocate %d ports", numPorts)
	}

	// allocate them
	for _, port := range portsToAllocate {
		if err := pa.allocate(port); err != nil {
			pa.releasePorts(portsToAllocate)
			return nil, fmt.Errorf("failed to allocate port %d: %w", port, err)
		}
	}

	pa.allocations[allocationID] = portsToAllocate
	return portsToAllocate, nil
}

func (pa *PortAllocator) getAvailablePorts(numPorts int) []int {
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

func (pa *PortAllocator) allocate(port int) error {
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

// Release releases the ports associated with the allocation ID.
func (pa *PortAllocator) Release(allocationID string) {
	pa.mx.Lock()
	defer pa.mx.Unlock()

	allocated, ok := pa.allocations[allocationID]
	if !ok {
		return
	}

	pa.releasePorts(allocated)
	delete(pa.allocations, allocationID)
}

func (pa *PortAllocator) releasePorts(ports []int) {
	for _, p := range ports {
		if _, ok := pa.reserved[p]; !ok {
			continue
		}
		delete(pa.reserved, p)
	}
}

// GetAllocation returns the allocated ports for a specific allocation ID.
func (pa *PortAllocator) GetAllocation(allocationID string) ([]int, error) {
	ports, exists := pa.allocations[allocationID]
	if !exists {
		return nil, fmt.Errorf("port allocation ID not found: %s", allocationID)
	}
	return ports, nil
}

// Allocated checks if the given ports are already allocated.
func (pa *PortAllocator) Allocated(ports []int) bool {
	for _, port := range ports {
		if _, reserved := pa.reserved[port]; reserved {
			return true
		}
	}

	return false
}

func (pa *PortAllocator) PortsAvailable(numPorts int) bool {
	pa.mx.Lock()
	defer pa.mx.Unlock()

	ports := pa.getAvailablePorts(numPorts)

	return len(ports) == numPorts
}
