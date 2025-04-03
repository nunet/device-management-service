package types

// Mounter is responsible for mounting and unmounting a volume.
type Mounter interface {
	Mount(targetPath string, options map[string]string) error
	Unmount(targetPath string) error
}

type VolumeConfig struct {
	// The type of storage backend, e.g., "glusterfs" or "local".
	Type string `json:"type"`

	Name             string   `json:"name"`
	Servers          []string `json:"servers"`
	ClientPrivateKey string   `json:"client_private_key"`
	ClientPEM        string   `json:"client_pem"`
	ClientCA         string   `json:"client_ca"`

	// Local
	Path string `json:"path,omitempty"` // The path on the local filesystem.
}
