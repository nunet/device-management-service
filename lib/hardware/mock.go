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
