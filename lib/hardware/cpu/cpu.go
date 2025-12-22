// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cpu

import (
	"fmt"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"

	"gitlab.com/nunet/device-management-service/types"
)

var (
	cpuCache     *types.CPU
	cpuCacheOnce sync.Once
	cpuCacheErr  error
)

const (
	sampleInterval = 1 * time.Second
	alpha          = 0.4 // smoothing factor (should be tweaked)
)

// monitor CPU usage using an exponential moving average to help with
// smoothing out short-term fluctuations
type Monitor struct {
	mu       sync.RWMutex
	avgUsage float64
	ticker   *time.Ticker
	done     chan struct{}
	running  bool
}

func NewCPUMonitor() *Monitor {
	monitor := &Monitor{
		avgUsage: 0.0,
		done:     make(chan struct{}),
	}
	monitor.Start()

	return monitor
}

func (c *Monitor) Start() {
	if c.running {
		return
	}

	c.running = true
	c.ticker = time.NewTicker(sampleInterval)
	go func() {
		for {
			select {
			case <-c.ticker.C:
				cpuPercent, err := cpu.Percent(sampleInterval, false)
				if err != nil {
					continue
				}
				if len(cpuPercent) > 0 {
					c.mu.Lock()
					c.avgUsage = alpha*cpuPercent[0] + (1.0-alpha)*c.avgUsage
					c.mu.Unlock()
				}
			case <-c.done:
				return
			}
		}
	}()
}

func (c *Monitor) Stop() {
	if !c.running {
		return
	}
	c.running = false
	c.ticker.Stop()
	close(c.done)
}

func (c *Monitor) GetAvgCPUUsage() (types.CPU, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cpuInfo, err := GetCPU()
	if err != nil {
		return types.CPU{}, fmt.Errorf("get CPU info: %s", err)
	}

	usedCores := float64(cpuInfo.Cores) * c.avgUsage / 100
	cpuInfo.Cores = float32(usedCores)
	return cpuInfo, nil
}

// GetCPU returns the CPU information for the system
func GetCPU() (types.CPU, error) {
	cpuCacheOnce.Do(func() {
		cpuInfo, err := getCPU()
		if err != nil {
			cpuCacheErr = fmt.Errorf("get CPU info: %w", err)
			return
		}
		cpuCache = &cpuInfo
	})

	if cpuCacheErr != nil {
		return types.CPU{}, cpuCacheErr
	}

	if cpuCache == nil {
		return types.CPU{}, fmt.Errorf("cpu info is nil")
	}

	return *cpuCache, nil
}

// GetUsage returns the CPU usage for the system at the current moment
func GetUsage() (types.CPU, error) {
	cpuUsage, err := cpu.Percent(time.Second, false)
	if err != nil {
		return types.CPU{}, fmt.Errorf("failed to get CPU usage: %s", err)
	}

	cpuInfo, err := GetCPU()
	if err != nil {
		return types.CPU{}, fmt.Errorf("get CPU info: %s", err)
	}

	// Calculate the used cores
	usedCores := float64(cpuInfo.Cores) * cpuUsage[0] / 100
	cpuInfo.Cores = float32(usedCores)
	return cpuInfo, nil
}
