package volume

import (
	"fmt"

	"gitlab.com/nunet/device-management-service/storage"
	"gitlab.com/nunet/device-management-service/storage/volume/glusterfs"
	"gitlab.com/nunet/device-management-service/storage/volume/localfs"
	"gitlab.com/nunet/device-management-service/types"
)

// New creates a volume implementation based on the provided configuration.
func New(t *storage.VoumeTracker, sc types.VolumeConfig) (types.Mounter, error) {
	switch sc.Type {
	case "glusterfs":
		return glusterfs.New(t, sc.Servers, sc.Name, sc.SSLCert, sc.SSLKey, sc.SSLCa)
	case "local":
		return localfs.New(sc.Path)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", sc.Type)
	}
}
