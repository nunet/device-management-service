// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package containerd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

const (
	networkConfBaseDir  = "/tmp/nunet/net-conf-"
	defaultNameserver   = "1.1.1.1"
	defaultDNSSearch    = "internal"
	defaultDNSOptions   = "ndots:1 timeout:2 attempts:1"
	containerResolvConf = "/etc/resolv.conf"
	containerHostsFile  = "/etc/hosts"
)

func prepNetworkConf(executionID, gatewayIP, containerIP string) ([]specs.Mount, error) {
	hostDir := networkConfHostDir(executionID)
	if err := removeNetworkConfDir(executionID); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(hostDir, 0o700); err != nil {
		return nil, fmt.Errorf("create network config directory: %w", err)
	}

	if err := mountNetworkConfDir(hostDir); err != nil {
		_ = os.RemoveAll(hostDir)
		return nil, fmt.Errorf("prepare network config directory: %w", err)
	}

	resolvPath := filepath.Join(hostDir, "resolv.conf")
	hostsPath := filepath.Join(hostDir, "hosts")
	if err := os.WriteFile(resolvPath, []byte(buildResolvConf(gatewayIP)), 0o644); err != nil {
		_ = removeNetworkConfDir(executionID)
		return nil, fmt.Errorf("write resolv.conf: %w", err)
	}
	if err := os.WriteFile(hostsPath, []byte(buildHosts(executionID, containerIP)), 0o644); err != nil {
		_ = removeNetworkConfDir(executionID)
		return nil, fmt.Errorf("write hosts: %w", err)
	}

	return []specs.Mount{
		{
			Type:        "bind",
			Source:      resolvPath,
			Destination: containerResolvConf,
			Options:     []string{"bind", "ro", "nosuid", "nodev"},
		},
		{
			Type:        "bind",
			Source:      hostsPath,
			Destination: containerHostsFile,
			Options:     []string{"bind", "ro", "nosuid", "nodev"},
		},
	}, nil
}

func buildResolvConf(gatewayIP string) string {
	var b strings.Builder
	if gateway := strings.TrimSpace(gatewayIP); gateway != "" {
		fmt.Fprintf(&b, "nameserver %s\n", gateway)
	}
	fmt.Fprintf(&b, "nameserver %s\n", defaultNameserver)
	fmt.Fprintf(&b, "search %s\n", defaultDNSSearch)
	fmt.Fprintf(&b, "options %s\n", defaultDNSOptions)
	return b.String()
}

func buildHosts(hostname, containerIP string) string {
	var b strings.Builder
	b.WriteString("127.0.0.1\tlocalhost\n")
	b.WriteString("::1\tlocalhost ip6-localhost ip6-loopback\n")
	if ip := strings.TrimSpace(containerIP); ip != "" {
		fmt.Fprintf(&b, "%s\t%s\n", ip, hostname)
	}
	return b.String()
}

func networkConfHostDir(id string) string {
	return networkConfBaseDir + id
}

func removeNetworkConfDir(id string) error {
	hostDir := networkConfHostDir(id)
	if _, err := os.Stat(hostDir); err == nil {
		if err := unmountNetworkConfDir(hostDir); err != nil {
			log.Warnw("failed to unmount network config directory", "dir", hostDir, "error", err)
		}
	}
	return os.RemoveAll(hostDir)
}

func removeAllNetworkConfDirs() {
	matches, err := filepath.Glob(networkConfBaseDir + "*")
	if err != nil {
		log.Warnw("failed to find network config directories", "error", err)
		return
	}

	for _, dir := range matches {
		id := strings.TrimPrefix(dir, networkConfBaseDir)
		if err := removeNetworkConfDir(id); err != nil {
			log.Warnw("failed to remove network config directory", "dir", dir, "error", err)
		}
	}
}
