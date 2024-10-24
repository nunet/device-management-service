// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package hardware

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultHardwareManager_GetMachineResources(t *testing.T) {
	t.Parallel()

	hm := NewHardwareManager()
	machineResources, err := hm.GetMachineResources()
	require.NoError(t, err)
	require.NotZero(t, machineResources.CPU.Cores)
	require.NotZero(t, machineResources.CPU.ClockSpeed)
	require.NotZero(t, machineResources.RAM.Size)
	require.NotZero(t, machineResources.Disk.Size)
}

func TestDefaultHardwareManager_GetFreeResources(t *testing.T) {
	t.Parallel()

	hm := NewHardwareManager()
	freeResources, err := hm.GetFreeResources()
	require.NoError(t, err)
	require.NotZero(t, freeResources.CPU.Cores)
	require.NotZero(t, freeResources.CPU.ClockSpeed)
	require.NotZero(t, freeResources.RAM.Size)
	require.NotZero(t, freeResources.Disk.Size)
}

func TestDefaultHardwareManager_GetUsage(t *testing.T) {
	t.Parallel()

	hm := NewHardwareManager()
	usage, err := hm.GetUsage()
	require.NoError(t, err)
	require.NotZero(t, usage.CPU.Cores)
	require.NotZero(t, usage.CPU.ClockSpeed)
	require.NotZero(t, usage.RAM.Size)
	require.NotZero(t, usage.Disk.Size)
}
