// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package containerd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	containernetns "github.com/containerd/containerd/v2/pkg/netns"
	"github.com/containernetworking/cni/libcni"
	cnitypes "github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"

	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/types"
)

type networkManager struct {
	cni          libcni.CNI
	netConfDir   string
	networkName  string
	ifName       string
	bridgeIface  string
	netNSBaseDir string
}

func newNetworkManager(cfg config.Containerd) (*networkManager, error) {
	if _, err := os.Stat(cfg.CNINetConfDir); err != nil {
		return nil, fmt.Errorf("CNI config directory %q: %w", cfg.CNINetConfDir, err)
	}
	if _, err := os.Stat(cfg.CNIPluginDir); err != nil {
		return nil, fmt.Errorf("CNI plugin directory %q: %w", cfg.CNIPluginDir, err)
	}
	if err := os.MkdirAll(cfg.NetNSBaseDir, 0o755); err != nil {
		return nil, fmt.Errorf("create netns directory %q: %w", cfg.NetNSBaseDir, err)
	}

	if _, err := libcni.LoadConfList(cfg.CNINetConfDir, cfg.CNINetworkName); err != nil {
		return nil, fmt.Errorf("load CNI network %q from %q: %w", cfg.CNINetworkName, cfg.CNINetConfDir, err)
	}

	return &networkManager{
		cni:          libcni.NewCNIConfig([]string{cfg.CNIPluginDir}, nil),
		netConfDir:   cfg.CNINetConfDir,
		networkName:  cfg.CNINetworkName,
		ifName:       DefaultCNIIfName,
		bridgeIface:  cfg.CNIBridgeIface,
		netNSBaseDir: cfg.NetNSBaseDir,
	}, nil
}

func (m *networkManager) setup(
	ctx context.Context,
	containerID string,
	ports []types.PortsToBind,
) (*networkSetup, error) {
	netConfList, err := libcni.LoadConfList(m.netConfDir, m.networkName)
	if err != nil {
		return nil, fmt.Errorf("load CNI network list: %w", err)
	}

	netNS, err := containernetns.NewNetNS(m.netNSBaseDir)
	if err != nil {
		return nil, fmt.Errorf("create network namespace: %w", err)
	}

	setup := &networkSetup{
		containerID: containerID,
		netNS:       netNS,
		netConfList: netConfList,
		runtimeConf: &libcni.RuntimeConf{
			ContainerID: containerID,
			NetNS:       netNS.GetPath(),
			IfName:      m.ifName,
		},
	}

	if len(ports) > 0 {
		setup.runtimeConf.CapabilityArgs = map[string]interface{}{
			"portMappings": toPortMappings(ports),
		}
	}

	result, err := m.cni.AddNetworkList(ctx, netConfList, setup.runtimeConf)
	if err != nil {
		_ = netNS.Remove()
		return nil, fmt.Errorf("CNI ADD for %q: %w", containerID, err)
	}

	setup.netInfo = buildNetInfo(m.ifName, m.bridgeIface, ports, result)
	return setup, nil
}

func buildNetInfo(ifName, bridgeIface string, ports []types.PortsToBind, result cnitypes.Result) types.ExecutorNetInfo {
	info := types.ExecutorNetInfo{
		InterfaceName: ifName,
		HostBridge:    bridgeIface,
		MappedPorts:   mappedPortsFromRequest(ports),
	}

	if result == nil {
		return info
	}

	cniResult, err := current.NewResultFromResult(result)
	if err != nil {
		return info
	}

	for _, ipCfg := range cniResult.IPs {
		if ipCfg.Address.IP == nil {
			continue
		}
		info.IPAddress = ipCfg.Address.IP.String()
		info.CIDR = (&ipCfg.Address).String()
		break
	}

	return info
}

func mappedPortsFromRequest(ports []types.PortsToBind) []types.ExecutorMappedPort {
	if len(ports) == 0 {
		return nil
	}

	mappings := make([]types.ExecutorMappedPort, 0, len(ports))
	for _, p := range ports {
		hostIP := strings.TrimSpace(p.IP)
		if hostIP == "" {
			hostIP = "0.0.0.0"
		}
		mappings = append(mappings, types.ExecutorMappedPort{
			HostIP:       hostIP,
			HostPort:     p.HostPort,
			ExecutorPort: p.ExecutorPort,
			Protocol:     "tcp",
		})
	}
	return mappings
}

func (m *networkManager) teardown(ctx context.Context, setup *networkSetup) error {
	if setup == nil {
		return nil
	}

	var errs []error
	if setup.netConfList != nil && setup.runtimeConf != nil {
		if err := m.cni.DelNetworkList(ctx, setup.netConfList, setup.runtimeConf); err != nil {
			errs = append(errs, fmt.Errorf("CNI DEL for %q: %w", setup.containerID, err))
		}
	}
	if setup.netNS != nil {
		if err := setup.netNS.Remove(); err != nil {
			errs = append(errs, fmt.Errorf("remove netns for %q: %w", setup.containerID, err))
		}
	}

	return errors.Join(errs...)
}

func toPortMappings(ports []types.PortsToBind) []portMapping {
	mappings := make([]portMapping, 0, len(ports))
	for _, p := range ports {
		hostIP := strings.TrimSpace(p.IP)
		if hostIP == "" {
			hostIP = "0.0.0.0"
		}
		mappings = append(mappings, portMapping{
			HostPort:      p.HostPort,
			ContainerPort: p.ExecutorPort,
			Protocol:      "tcp",
			HostIP:        hostIP,
		})
	}
	return mappings
}
