// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package hardware

import (
	"fmt"

	"github.com/shirou/gopsutil/v4/mem"

	"gitlab.com/nunet/device-management-service/types"
)

// GetRAM returns the types.RAM information for the system
func GetRAM() (types.RAM, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return types.RAM{}, fmt.Errorf("failed to get total memory: %s", err)
	}

	return types.RAM{
		Size: v.Total,
	}, nil
}

// GetRAMUsage returns the RAM usage
func GetRAMUsage() (types.RAM, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return types.RAM{}, fmt.Errorf("failed to get total memory: %s", err)
	}

	return types.RAM{
		Size: v.Used,
	}, nil
}
