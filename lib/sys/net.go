// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package sys

import (
	"net"
	"strings"

	"github.com/songgao/water"
)

// types for tun tap
type TunTapMode int

const (
	NetTunMode TunTapMode = iota
	NetTapMode
)

// // TUN is a struct containing the fields necessary
// // to configure a system TUN device. Access the
// // internal TUN device through TUN.Iface
type NetInterface struct {
	Iface *water.Interface
	Src   string
	Dst   string
}

// GetNetInterfaces gets the list of network interfaces
func GetNetInterfaces() ([]net.Interface, error) {
	return net.Interfaces()
}

// GetNetInterfaceByName gets the network interface by name
func GetNetInterfaceByName(name string) (*net.Interface, error) {
	return net.InterfaceByName(name)
}

// GetNetInterfaceByFlags gets the network interface by the flags
func GetNetInterfaceByFlags(flag net.Flags) (*net.Interface, error) {
	ifaces, err := GetNetInterfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		if iface.Flags&flag != 0 {
			return &iface, nil
		}
	}
	return nil, nil
}

func GetUsedAddresses() ([]string, error) {
	ifaces, err := GetNetInterfaces()
	if err != nil {
		return nil, err
	}

	var networks []string
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			if !strings.Contains(addr.String(), ":") {
				networks = append(networks, addr.String())
			}
		}
	}

	return networks, nil
}
