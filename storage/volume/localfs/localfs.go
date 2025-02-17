package localfs

import (
	"fmt"
	"os/exec"

	"gitlab.com/nunet/device-management-service/types"
)

type LocalFS struct {
	path string
}

var _ types.Mounter = (*LocalFS)(nil)

// New creates a new LocalFS storage instance using the provided path.
func New(path string) (*LocalFS, error) {
	if path == "" {
		return nil, fmt.Errorf("local filesystem path cannot be empty")
	}
	return &LocalFS{path: path}, nil
}

// Mount for LocalFS might perform a bind mount or simply check that the path exists.
func (l *LocalFS) Mount(targetPath string, _ map[string]string) error {
	if targetPath == "" {
		return fmt.Errorf("target path cannot be empty")
	}

	// perform a bind mount: mount --bind <l.path> <targetPath>
	cmd := exec.Command("mount", "--bind", l.path, targetPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to mount local filesystem: %w, output: %s", err, output)
	}
	return nil
}

func (l *LocalFS) Unmount(targetPath string) error {
	if targetPath == "" {
		return fmt.Errorf("target path cannot be empty")
	}
	cmd := exec.Command("umount", targetPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to unmount local filesystem: %w, output: %s", err, output)
	}
	return nil
}
