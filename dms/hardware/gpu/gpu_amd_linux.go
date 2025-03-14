// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build linux && (amd64 || 386)

package gpu

import (
	"fmt"
	"sync"

	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

var ErrNotInitialised = fmt.Errorf("GPU manager not initialised")

// gpuConnectors is a struct that holds the GPU connectors for different vendors.
type gpuConnectors struct {
	nvidia types.GPUConnector
	amd    types.GPUConnector
	intel  types.GPUConnector
}

// gpuManager implements the types.GPUManager interface.
type gpuManager struct {
	initialised bool

	gpuIndexCache map[string]int // UUID to gpuCache index map
	gpuCache      types.GPUs     // Cache of GPUs

	connectors gpuConnectors
	lock       sync.RWMutex
}

var _ types.GPUManager = &gpuManager{}

// NewGPUManager creates a new GPU manager.
func NewGPUManager() types.GPUManager {
	nvidiaConnector, err := newNVIDIAGPUConnector()
	if err != nil {
		log.Warnw("create NVIDIA GPU connector",
			"labels", []string{string(observability.LabelNode)},
			"error", err)
	}
	amdConnector, err := newAMDGPUConnector()
	if err != nil {
		log.Warnw("create AMD GPU connector",
			"labels", []string{string(observability.LabelNode)},
			"error", err)
	}
	intelConnector, err := newIntelGPUConnector()
	if err != nil {
		log.Warnw("create Intel GPU connector",
			"labels", []string{string(observability.LabelNode)},
			"error", err)
	}
	connector := gpuConnectors{
		nvidia: nvidiaConnector,
		amd:    amdConnector,
		intel:  intelConnector,
	}

	return &gpuManager{
		initialised:   true,
		gpuIndexCache: make(map[string]int),
		connectors:    connector,
	}
}

// getGPUs is a helper function to get the GPUs from the device managers
func (g *gpuManager) getGPUs() (types.GPUs, error) {
	var gpus []types.GPU
	if g.connectors.nvidia != nil {
		nvidiaGPUs, err := g.connectors.nvidia.GetGPUs()
		if err != nil {
			return nil, fmt.Errorf("get NVIDIA GPUs: %w", err)
		}
		gpus = append(gpus, nvidiaGPUs...)
	}

	if g.connectors.amd != nil {
		amdGPUs, err := g.connectors.amd.GetGPUs()
		if err != nil {
			return nil, fmt.Errorf("get AMD GPUs: %w", err)
		}
		gpus = append(gpus, amdGPUs...)
	}

	if g.connectors.intel != nil {
		intelGPUs, err := g.connectors.intel.GetGPUs()
		if err != nil {
			return nil, fmt.Errorf("get Intel GPUs: %w", err)
		}
		gpus = append(gpus, intelGPUs...)
	}

	return gpus, nil
}

// GetGPUs returns the GPUs in the system
func (g *gpuManager) GetGPUs() (types.GPUs, error) {
	if !g.initialised {
		return nil, ErrNotInitialised
	}

	// Check if the GPUs are cached
	g.lock.RLock()
	if len(g.gpuCache) > 0 {
		g.lock.RUnlock()
		return g.gpuCache.Copy(), nil
	}
	g.lock.RUnlock()

	g.lock.Lock()
	defer g.lock.Unlock()

	gpus, err := g.getGPUs()
	if err != nil {
		g.lock.Unlock()
		return nil, fmt.Errorf("get GPUs: %w", err)
	}

	// Assign index to GPUs
	// Note: The index is internal to dms and is not the same as the device index
	gpus = assignIndexToGPUs(gpus)

	// Cache the GPUs
	g.gpuCache = gpus
	for i, gpu := range gpus {
		g.gpuIndexCache[gpu.UUID] = i
	}

	return g.gpuCache.Copy(), nil
}

// getGPUUsage a helper function to get the GPU usage based on the vendor
func (g *gpuManager) getGPUUsage(uuid string, vendor types.GPUVendor) (float64, error) {
	switch vendor {
	case types.GPUVendorNvidia:
		usage, err := g.connectors.nvidia.GetGPUUsage(uuid)
		if err != nil {
			return 0, fmt.Errorf("get nvidia gpu usage: %w", err)
		}
		return usage, nil
	case types.GPUVendorAMDATI:
		usage, err := g.connectors.amd.GetGPUUsage(uuid)
		if err != nil {
			return 0, fmt.Errorf("get amd gpu usage: %w", err)
		}
		return usage, nil
	case types.GPUVendorIntel:
		usage, err := g.connectors.intel.GetGPUUsage(uuid)
		if err != nil {
			return 0, fmt.Errorf("get intel gpu usage: %w", err)
		}
		return usage, nil

	default:
		return 0, fmt.Errorf("unsupported vendor")
	}
}

// GetGPUUsage returns the GPU usage based on the specified uuid.
// if uuid is empty, it returns the usage of all GPUs.
func (g *gpuManager) GetGPUUsage(uuid ...string) (types.GPUs, error) {
	if !g.initialised {
		return nil, ErrNotInitialised
	}

	// Check if there are gpus
	gpuCache, err := g.GetGPUs()
	if err != nil {
		return nil, fmt.Errorf("get gpus: %w", err)
	}
	if len(gpuCache) == 0 {
		return nil, nil
	}

	// Get the GPUs based on the UUID
	var gpus []types.GPU
	if len(uuid) == 0 {
		// copy the GPU cache
		gpus = gpuCache
	} else {
		// Get the GPUs based on the UUID
		for _, u := range uuid {
			g.lock.RLock()
			if index, ok := g.gpuIndexCache[u]; ok {
				gpus = append(gpus, g.gpuCache[index])
			}
			g.lock.RUnlock()
		}
	}

	if len(gpus) == 0 {
		return nil, fmt.Errorf("no GPUs found for the specified parameters")
	}

	for i, gpu := range gpus {
		// Get the GPU usage based on the vendor
		usage, err := g.getGPUUsage(gpu.UUID, gpu.Vendor)
		if err != nil {
			return nil, fmt.Errorf("get gpu usage: %w", err)
		}
		gpus[i].VRAM = usage
	}

	return gpus, nil
}

// Shutdown shuts down the GPU manager
// TODO: this results in a permanent shutdown of the GPU manager and we don't have a way to restart it yet
func (g *gpuManager) Shutdown() error {
	if !g.initialised {
		return ErrNotInitialised
	}

	g.lock.Lock()
	defer g.lock.Unlock()

	if g.connectors.nvidia != nil {
		if err := g.connectors.nvidia.Shutdown(); err != nil {
			log.Errorw("shutdown nvml",
				"labels", []string{string(observability.LabelNode)},
				"error", err)
		}
	}

	if g.connectors.amd != nil {
		if err := g.connectors.amd.Shutdown(); err != nil {
			log.Errorw("shutdown amdsmi",
				"labels", []string{string(observability.LabelNode)},
				"error", err)
		}
	}

	if g.connectors.intel != nil {
		if err := g.connectors.intel.Shutdown(); err != nil {
			log.Errorw("shutdowm xpum",
				"labels", []string{string(observability.LabelNode)},
				"error", err)
		}
	}

	g.connectors = gpuConnectors{}
	g.initialised = false
	// Clear the cache
	g.gpuCache = nil
	g.gpuIndexCache = nil

	return nil
}
