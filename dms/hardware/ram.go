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
		Size: float64(v.Total),
	}, nil
}

// GetRAMUsage returns the RAM usage
func GetRAMUsage() (types.RAM, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return types.RAM{}, fmt.Errorf("failed to get total memory: %s", err)
	}

	return types.RAM{
		Size: float64(v.Used),
	}, nil
}
