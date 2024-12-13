// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cpu

import (
	"testing"

	"gitlab.com/nunet/device-management-service/types"

	"github.com/stretchr/testify/require"
)

func TestGetCPU(t *testing.T) {
	cpu, err := GetCPU()
	require.NoError(t, err)
	require.Greater(t, cpu.Cores, float32(0))
	require.Greater(t, cpu.ClockSpeed, float64(0))
}

func TestGetCPUUsage(t *testing.T) {
	cpu, err := GetUsage()
	require.NoError(t, err)
	require.Greater(t, cpu.Cores, float32(0))
	require.Greater(t, cpu.ClockSpeed, float64(0))
}

func TestCPUCache(t *testing.T) {
	cpuCache = &types.CPU{
		Cores:      4,
		ClockSpeed: 2.5,
	}
	cpu, err := GetCPU()
	require.NoError(t, err)
	require.Equal(t, cpuCache, &cpu)

	// Ensure that the value is returned safely
	cpu.Cores = 0
	cpu.ClockSpeed = 0

	actualCPU, err := GetCPU()
	require.NoError(t, err)
	require.Equal(t, cpuCache, &actualCPU)
}
