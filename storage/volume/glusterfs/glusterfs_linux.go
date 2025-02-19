// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build linux
// +build linux

package glusterfs

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"syscall"

	"gitlab.com/nunet/device-management-service/storage"
	"gitlab.com/nunet/device-management-service/types"
	"golang.org/x/sys/unix"
)

// Commander is an interface that wraps the CombinedOutput method.
type Commander interface {
	CombinedOutput() ([]byte, error)
}

// ExecFunc defines the function signature for command execution.
// XXX: would be good to standardize this and lift up to the lib/sys package and
// use for the localfs too.
type ExecFunc func(name string, args ...string) Commander

// execCommand is the function used to run external commands. It defaults to exec.Command.
var execCommand ExecFunc = func(name string, args ...string) Commander {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		AmbientCaps: []uintptr{
			unix.CAP_SYS_ADMIN,
		},
	}
	return cmd
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
