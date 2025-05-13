// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

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

// GetDiskUsage returns the types.Disk usage
func GetDiskUsage() (types.Disk, error) {
	partitions, err := disk.PartitionsWithContext(context.Background(), false)
	if err != nil {
		return types.Disk{}, fmt.Errorf("failed to get partitions: %w", err)
	}

	var usedStorage uint64
	for p := range partitions {
		usage, err := disk.UsageWithContext(context.Background(), partitions[p].Mountpoint)
		if err != nil {
			return types.Disk{}, fmt.Errorf("failed to get disk usage: %w", err)
		}
		usedStorage += usage.Used
	}

	return types.Disk{
		Size: usedStorage,
	}, nil
}
