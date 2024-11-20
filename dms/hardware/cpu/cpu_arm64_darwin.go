// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build darwin && arm64

package cpu

import (
	"github.com/shoenig/go-m1cpu"
	"gitlab.com/nunet/device-management-service/types"
)

// getCPU returns the  types.CPU information for the system
func getCPU() (types.CPU, error) {
	var (
		totalCompute float64
		totalCores   uint64
	)
	eCompute := float64(m1cpu.ECoreCount()) * m1cpu.ECoreGHz() * 1000000
	pCompute := float64(m1cpu.PCoreCount()) * m1cpu.PCoreGHz() * 1000000
	totalCompute = eCompute + pCompute
	totalCores = uint64(m1cpu.ECoreCount() + m1cpu.PCoreCount())

	return types.CPU{
		Cores:      float32(totalCores),
		ClockSpeed: totalCompute / float64(m1cpu.ECoreCount()+m1cpu.PCoreCount()),
	}, nil
}
