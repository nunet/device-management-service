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

// AllocatePorts allocates the requested number of ports and associates them with the allocation ID.
func (pa *PortAllocator) Allocate(allocationID string, numPorts int) ([]int, error) {
	pa.mx.Lock()
	defer pa.mx.Unlock()

	var allocated []int
	for i := 0; i < numPorts; i++ {
		port, err := pa.allocate()
		if err != nil {
			return nil, err
		}
		allocated = append(allocated, port)
	}

	for _, p := range allocated {
		pa.reserved[p] = struct{}{}
	}
	pa.allocs[allocationID] = allocated

	return allocated, nil
}

func (pa *PortAllocator) allocate() (int, error) {
	for i := pa.config.AvailableRangeFrom; i <= pa.config.AvailableRangeTo; i++ {
		_, reserved := pa.reserved[i]
		if reserved {
			continue
		}
		pa.reserved[i] = struct{}{}
		return i, nil
	}

	return 0, fmt.Errorf("no available ports")
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

// GetAllocations returns the allocated ports for a specific allocation ID.
func (pa *PortAllocator) GetAllocation(allocationID string) ([]int, error) {
	ports, exists := pa.allocs[allocationID]
	if !exists {
		return nil, fmt.Errorf("port allocation ID not found: %s", allocationID)
	}
	return ports, nil
}
