// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build amd64 && darwin

package cpu

import (
	"fmt"

	"github.com/shirou/gopsutil/v4/cpu"

	"gitlab.com/nunet/device-management-service/types"
)

// GetCPU returns the types.CPU information for the system
func GetCPU() (types.CPU, error) {
	cpus, err := cpu.Info()
	if err != nil {
		return types.CPU{}, fmt.Errorf("failed to get CPU info: %w", err)
	}

	var (
		totalCompute float64
		totalCores   uint64
	)
	for c := range cpus {
		totalCompute += float64(cpus[c].Cores) * cpus[c].Mhz
		totalCores += uint64(cpus[c].Cores)
	}

	cpuInfo := types.CPU{
		Cores:      float32(totalCores),
		ClockSpeed: float64(cpus[0].Mhz) * 1000000,
	}
	return cpuInfo, nil
}
