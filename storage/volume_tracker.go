package storage

import (
	"errors"
	"sync"
)

type VoumeTracker struct {
	mu     sync.RWMutex
	mounts map[string]TrackedVolume
}

type TrackedVolume struct {
	AllocationID string
	ContainerID  string
}

func NewVolumeTracker() *VoumeTracker {
	return &VoumeTracker{
		mounts: map[string]TrackedVolume{},
	}
}

func (v *VoumeTracker) TrackMount(targetPath, allocationID, containerID string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.mounts[targetPath] = TrackedVolume{
		AllocationID: allocationID,
		ContainerID:  containerID,
	}
}

func (v *VoumeTracker) UntrackMount(targetPath string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	delete(v.mounts, targetPath)
}

func (v *VoumeTracker) IsMounted(targetPath string) bool {
	v.mu.RLock()
	defer v.mu.RUnlock()

	_, ok := v.mounts[targetPath]
	return ok
}

func (v *VoumeTracker) GetMountInfo(targetPath string) (TrackedVolume, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	info, ok := v.mounts[targetPath]
	if !ok {
		return TrackedVolume{}, errors.New("failed to find volume")
	}

	return info, nil
}
