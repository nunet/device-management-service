package types

// Mounter is responsible for mounting and unmounting a volume.
type Mounter interface {
	Mount(targetPath string, options map[string]string) error
	Unmount(targetPath string) error
}

type VolumeConfig struct {
	// The type of storage backend, e.g., "glusterfs" or "local".
	Type string `json:"type"`

	//  GlusterFS:
	Servers []string `json:"servers,omitempty"` // List of GlusterFS server addresses.
	Name    string   `json:"name,omitempty"`    // Name of the GlusterFS volume.

	// Local
	Path string `json:"path,omitempty"` // The path on the local filesystem.
}
