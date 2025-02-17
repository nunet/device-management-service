package glusterfs

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"gitlab.com/nunet/device-management-service/storage"
	"gitlab.com/nunet/device-management-service/types"
)

// Commander is an interface that wraps the CombinedOutput method.
type Commander interface {
	CombinedOutput() ([]byte, error)
}

// ExecFunc defines the function signature for command execution.
type ExecFunc func(name string, args ...string) Commander

// execCommand is the function used to run external commands. It defaults to exec.Command.
var execCommand ExecFunc = func(name string, args ...string) Commander {
	return exec.Command(name, args...)
}

// GlusterFS holds the configuration needed to mount a GlusterFS volume.
type GlusterFS struct {
	servers []string
	name    string
	sslCert string
	sslKey  string
	sslCA   string

	mu sync.Mutex

	tracker *storage.VoumeTracker
}

var _ types.Mounter = (*GlusterFS)(nil)

// New creates a new GlusterFS mounter with the provided configuration.
// The servers slice and volume are required; SSL certificate parameters are optional.
func New(t *storage.VoumeTracker, servers []string, name, sslCert, sslKey, sslCA string) (*GlusterFS, error) {
	if len(servers) == 0 {
		return nil, fmt.Errorf("no GlusterFS servers provided")
	}

	if name == "" {
		return nil, fmt.Errorf("no volume provided")
	}

	return &GlusterFS{
		servers: servers,
		name:    name,
		sslCert: sslCert,
		sslKey:  sslKey,
		sslCA:   sslCA,
		tracker: t,
	}, nil
}

// Mount mounts the GlusterFS volume to the provided targetPath.
// Additional mount options can be passed in the options map.
func (g *GlusterFS) Mount(targetPath string, options map[string]string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.tracker.IsMounted(targetPath) {
		return fmt.Errorf("%s is already mounted", targetPath)
	}

	if targetPath == "" {
		return fmt.Errorf("target path cannot be empty")
	}

	serverList := strings.Join(g.servers, ",")
	source := fmt.Sprintf("%s:%s", serverList, g.name)

	opts := make([]string, 0)
	for k, v := range options {
		opts = append(opts, fmt.Sprintf("%s=%s", k, v))
	}
	if g.sslCert != "" {
		opts = append(opts, fmt.Sprintf("ssl_cert=%s", g.sslCert))
	}
	if g.sslKey != "" {
		opts = append(opts, fmt.Sprintf("ssl_key=%s", g.sslKey))
	}
	if g.sslCA != "" {
		opts = append(opts, fmt.Sprintf("ssl_ca=%s", g.sslCA))
	}

	// build the arguments for the mount command:
	//   mount -t glusterfs -o <options> server1,server2:volume /target/path
	args := []string{"-t", "glusterfs"}
	if len(opts) > 0 {
		args = append(args, "-o", strings.Join(opts, ","))
	}
	args = append(args, source, targetPath)

	cmd := execCommand("mount", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to mount GlusterFS volume: %w, output: %s", err, output)
	}

	g.tracker.Mount(targetPath)

	return nil
}

// Unmount unmounts the GlusterFS volume from the provided targetPath.
func (g *GlusterFS) Unmount(targetPath string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.tracker.IsMounted(targetPath) {
		return fmt.Errorf("%s is not mounted", targetPath)
	}

	if targetPath == "" {
		return fmt.Errorf("target path cannot be empty")
	}

	cmd := execCommand("umount", targetPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to unmount GlusterFS volume: %w, output: %s", err, output)
	}

	g.tracker.Unmount(targetPath)

	return nil
}
