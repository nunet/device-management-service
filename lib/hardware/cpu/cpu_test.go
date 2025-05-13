// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cpu_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/lib/hardware/cpu"
)

// TestGetCPU tests the GetCPU function retrieves valid CPU information
func TestGetCPU(t *testing.T) {
	t.Parallel()

	cpuInfo, err := cpu.GetCPU()
	require.NoError(t, err)
	require.Greater(t, cpuInfo.Cores, float32(0))
	require.Greater(t, cpuInfo.ClockSpeed, float64(0))
}

// TestGetCPUUsage tests the GetUsage function retrieves valid CPU usage
func TestGetCPUUsage(t *testing.T) {
	t.Parallel()

	usage, err := cpu.GetUsage()
	require.NoError(t, err)
	require.Greater(t, usage.Cores, float32(0))
	require.Greater(t, usage.ClockSpeed, float64(0))
}

// TestUsedCoresLimit tests that used cores are not more than available cores
func TestUsedCoresLimit(t *testing.T) {
	t.Parallel()

	usedCPU, err := cpu.GetUsage()
	require.NoError(t, err)

	cpuInfo, err := cpu.GetCPU()
	require.NoError(t, err)

	require.True(t, usedCPU.Cores <= cpuInfo.Cores)
}

func TestConcurrency(t *testing.T) {
	t.Parallel()
	wg := sync.WaitGroup{}
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = cpu.GetCPU()
			_, _ = cpu.GetUsage()
		}()
	}
	wg.Wait()
}
