// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package storage

import (
	"errors"
	"sync"
)

type VolumeTracker struct {
	mu     sync.RWMutex
	mounts map[string]TrackedVolume
}

type TrackedVolume struct {
	AllocationID string
	ContainerID  string
}

func NewVolumeTracker() *VolumeTracker {
	return &VolumeTracker{
		mounts: map[string]TrackedVolume{},
	}
}

func (v *VolumeTracker) TrackMount(targetPath, allocationID, containerID string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.mounts[targetPath] = TrackedVolume{
		AllocationID: allocationID,
		ContainerID:  containerID,
	}
}

func (v *VolumeTracker) UntrackMount(targetPath string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	delete(v.mounts, targetPath)
}

func (v *VolumeTracker) IsMounted(targetPath string) bool {
	v.mu.RLock()
	defer v.mu.RUnlock()

	_, ok := v.mounts[targetPath]
	return ok
}

func (v *VolumeTracker) GetMountInfo(targetPath string) (TrackedVolume, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	info, ok := v.mounts[targetPath]
	if !ok {
		return TrackedVolume{}, errors.New("failed to find volume")
	}

	return info, nil
}
