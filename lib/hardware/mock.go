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

	"gitlab.com/nunet/device-management-service/types"
)

type mockHardwareManager struct {
	machineResources types.MachineResources
	freeResources    types.Resources
	usedResources    types.Resources
}

var _ types.HardwareManager = (*mockHardwareManager)(nil)

func NewMockHardwareManager(
	machineResources types.MachineResources,
	freeResources types.Resources,
	usedResources types.Resources,
) types.HardwareManager {
	return &mockHardwareManager{
		machineResources: machineResources,
		freeResources:    freeResources,
		usedResources:    usedResources,
	}
}

func (m *mockHardwareManager) GetMachineResources() (types.MachineResources, error) {
	return m.machineResources, nil
}

func (m *mockHardwareManager) GetFreeResources() (types.Resources, error) {
	return m.freeResources, nil
}

func (m *mockHardwareManager) GetUsage() (types.Resources, error) {
	return m.usedResources, nil
}

func (m *mockHardwareManager) CheckCapacity(resources types.Resources) (bool, error) {
	if err := m.freeResources.Subtract(resources); err != nil {
		return false, fmt.Errorf("%w: %w", types.ErrNoFreeResources, err)
	}

	return true, nil
}

func (m *mockHardwareManager) Shutdown() error { return nil }
