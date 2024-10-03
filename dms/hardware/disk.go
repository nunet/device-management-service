package hardware

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/v4/disk"

	"gitlab.com/nunet/device-management-service/types"
)

// GetDisk returns the types.Disk for the system
func GetDisk() (types.Disk, error) {
	partitions, err := disk.PartitionsWithContext(context.Background(), false)
	if err != nil {
		return types.Disk{}, fmt.Errorf("failed to get partitions: %w", err)
	}

	var totalStorage uint64
	for p := range partitions {
		usage, err := disk.UsageWithContext(context.Background(), partitions[p].Mountpoint)
		if err != nil {
			return types.Disk{}, fmt.Errorf("failed to get disk usage: %w", err)
		}
		totalStorage += usage.Total
	}

	return types.Disk{
		Size: totalStorage,
	}, nil
}
