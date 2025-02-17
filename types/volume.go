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

	// TODO: move to an appropriate place
	SSLCert string `json:"ssl_cert,omitempty"` // Optional path to the SSL certificate.
	SSLKey  string `json:"ssl_key,omitempty"`  // Optional path to the SSL key.
	SSLCa   string `json:"ssl_ca,omitempty"`   // Optional path to the CA certificate.

	// Local
	Path string `json:"path,omitempty"` // The path on the local filesystem.
}
