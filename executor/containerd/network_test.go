// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build linux

package containerd

import (
	"encoding/json"
	"testing"

	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/types"
)

func TestBuildNetInfo(t *testing.T) {
	t.Parallel()

	t.Run("with nil CNI result", func(t *testing.T) {
		t.Parallel()
		ports := []types.PortsToBind{
			{IP: "127.0.0.1", HostPort: 8080, ExecutorPort: 3000},
		}

		info := buildNetInfo(DefaultCNIIfName, ports, nil)
		require.Equal(t, DefaultCNIIfName, info.InterfaceName)
		require.Equal(t, DefaultCNIBridgeIface, info.HostBridge)
		require.Len(t, info.MappedPorts, 1)
		require.Equal(t, "127.0.0.1", info.MappedPorts[0].HostIP)
		require.Equal(t, 8080, info.MappedPorts[0].HostPort)
		require.Equal(t, 3000, info.MappedPorts[0].ExecutorPort)
	})

	t.Run("with CNI IP result", func(t *testing.T) {
		t.Parallel()
		ports := []types.PortsToBind{
			{IP: "127.0.0.1", HostPort: 8080, ExecutorPort: 3000},
		}

		raw := `{
			"cniVersion": "1.0.0",
			"ips": [{"address": "10.22.0.5/16"}]
		}`
		var result current.Result
		require.NoError(t, json.Unmarshal([]byte(raw), &result))

		info := buildNetInfo(DefaultCNIIfName, ports, &result)
		require.Equal(t, "10.22.0.5", info.IPAddress)
		require.Equal(t, "10.22.0.5/16", info.CIDR)
	})

	t.Run("with CNI result without IPs", func(t *testing.T) {
		t.Parallel()
		ports := []types.PortsToBind{
			{IP: "", HostPort: 8080, ExecutorPort: 3000},
		}

		raw := `{
			"cniVersion": "1.0.0"
		}`
		var result current.Result
		require.NoError(t, json.Unmarshal([]byte(raw), &result))

		info := buildNetInfo(DefaultCNIIfName, ports, &result)
		require.Equal(t, DefaultCNIIfName, info.InterfaceName)
		require.Equal(t, DefaultCNIBridgeIface, info.HostBridge)
		require.Equal(t, "0.0.0.0", info.MappedPorts[0].HostIP)
		require.Empty(t, info.IPAddress)
		require.Empty(t, info.CIDR)
	})
}

func TestMappedPortsFromRequest(t *testing.T) {
	t.Parallel()

	t.Run("empty input returns nil", func(t *testing.T) {
		t.Parallel()
		require.Nil(t, mappedPortsFromRequest(nil))
		require.Nil(t, mappedPortsFromRequest([]types.PortsToBind{}))
	})

	t.Run("normalizes host ip and protocol", func(t *testing.T) {
		t.Parallel()
		ports := []types.PortsToBind{
			{IP: "  ", HostPort: 8080, ExecutorPort: 3000},
			{IP: "127.0.0.1", HostPort: 8443, ExecutorPort: 3443},
		}

		mappings := mappedPortsFromRequest(ports)
		require.Len(t, mappings, 2)
		require.Equal(t, "0.0.0.0", mappings[0].HostIP)
		require.Equal(t, 8080, mappings[0].HostPort)
		require.Equal(t, 3000, mappings[0].ExecutorPort)
		require.Equal(t, "tcp", mappings[0].Protocol)

		require.Equal(t, "127.0.0.1", mappings[1].HostIP)
		require.Equal(t, 8443, mappings[1].HostPort)
		require.Equal(t, 3443, mappings[1].ExecutorPort)
		require.Equal(t, "tcp", mappings[1].Protocol)
	})
}

func TestToPortMappings(t *testing.T) {
	t.Parallel()

	t.Run("empty input returns empty slice", func(t *testing.T) {
		t.Parallel()
		require.Empty(t, toPortMappings(nil))
		require.Empty(t, toPortMappings([]types.PortsToBind{}))
	})

	t.Run("maps values and defaults host ip", func(t *testing.T) {
		t.Parallel()
		ports := []types.PortsToBind{
			{IP: "", HostPort: 9000, ExecutorPort: 90},
			{IP: "10.1.1.10", HostPort: 9443, ExecutorPort: 443},
		}

		mappings := toPortMappings(ports)
		require.Len(t, mappings, 2)

		require.Equal(t, 9000, mappings[0].HostPort)
		require.Equal(t, 90, mappings[0].ContainerPort)
		require.Equal(t, "tcp", mappings[0].Protocol)
		require.Equal(t, "0.0.0.0", mappings[0].HostIP)

		require.Equal(t, 9443, mappings[1].HostPort)
		require.Equal(t, 443, mappings[1].ContainerPort)
		require.Equal(t, "tcp", mappings[1].Protocol)
		require.Equal(t, "10.1.1.10", mappings[1].HostIP)
	})
}
