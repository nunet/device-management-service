package storage

import "sync"

type VoumeTracker struct {
	mu     sync.RWMutex
	mounts map[string]bool
}

func NewVolumeTracker() *VoumeTracker {
	return &VoumeTracker{
		mounts: make(map[string]bool),
	}
}

func (v *VoumeTracker) Mount(targetPath string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.mounts[targetPath] = true
}

func (v *VoumeTracker) Unmount(targetPath string) {
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
