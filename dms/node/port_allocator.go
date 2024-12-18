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
)

// PortAllocator keeps track of port allocations and manages state.
type PortAllocator struct {
	config PortConfig

	mx       sync.Mutex
	allocs   map[string][]int
	reserved map[int]struct{}
}

// NewPortAllocator initializes a new PortAllocator with a PortConfig.
func NewPortAllocator(config PortConfig) *PortAllocator {
	return &PortAllocator{
		config:   config,
		allocs:   make(map[string][]int),
		reserved: make(map[int]struct{}),
	}
}

// AllocatePorts allocates the requested ports, associating them with an allocation.
// If it's not possible to allocate one of the ports, an error is returned and no ports are allocated.
func (pa *PortAllocator) AllocatePorts(allocationID string, ports []int) error {
	pa.mx.Lock()
	defer pa.mx.Unlock()

	for _, port := range ports {
		if err := pa.allocate(port); err != nil {
			pa.releasePorts(ports)
			return fmt.Errorf("cannot allocate port %d: %w", port, err)
		}
	}

	pa.allocs[allocationID] = ports
	return nil
}

// AllocateRandom allocates the requested number of ports and associates them with the allocation ID.
func (pa *PortAllocator) AllocateRandom(allocationID string, numPorts int) ([]int, error) {
	pa.mx.Lock()
	defer pa.mx.Unlock()

	var portsToAllocate []int
	portNum := pa.config.AvailableRangeFrom

	// select random ports
	for range numPorts {
		for ; portNum <= pa.config.AvailableRangeTo; portNum++ {
			if _, reserved := pa.reserved[portNum]; !reserved {
				portsToAllocate = append(portsToAllocate, portNum)
				portNum++
				break
			}
		}
	}

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

	pa.allocs[allocationID] = portsToAllocate
	return portsToAllocate, nil
}

func (pa *PortAllocator) allocate(port int) error {
	if port < pa.config.AvailableRangeFrom || port > pa.config.AvailableRangeTo {
		return fmt.Errorf("port %d is outside allowed range [%d-%d]",
			port, pa.config.AvailableRangeFrom, pa.config.AvailableRangeTo)
	}

	if _, reserved := pa.reserved[port]; reserved {
		return fmt.Errorf("port %d is already reserved", port)
	}

	// TODO: Add system port availability check here
	// Example: Check if port is in use on the machine

	pa.reserved[port] = struct{}{}
	return nil
}

func (pa *PortAllocator) Release(allocationID string) {
	pa.mx.Lock()
	defer pa.mx.Unlock()

	allocated, ok := pa.allocs[allocationID]
	if !ok {
		return
	}

	for _, p := range allocated {
		delete(pa.reserved, p)
	}
	delete(pa.allocs, allocationID)
}

func (pa *PortAllocator) releasePorts(ports []int) {
	for _, p := range ports {
		if _, ok := pa.reserved[p]; !ok {
			continue
		}
		delete(pa.reserved, p)
	}
}

// GetAllocations returns the allocated ports for a specific allocation ID.
func (pa *PortAllocator) GetAllocation(allocationID string) ([]int, error) {
	ports, exists := pa.allocs[allocationID]
	if !exists {
		return nil, fmt.Errorf("port allocation ID not found: %s", allocationID)
	}
	return ports, nil
}
