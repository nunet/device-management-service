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

// NetInterface defines the interface for network interfaces (TUN/TAP)
type NetInterface interface {
	Name() string
	Write([]byte) (int, error)
	Read([]byte) (int, error)
	Up() error
	Down() error
	Delete() error
	SetAddress(string) error
	SetMTU(int) error
}

type netiface struct {
	iface *water.Interface
}

func (n *netiface) Name() string {
	return n.iface.Name()
}

func (n *netiface) Read(packet []byte) (int, error) {
	return n.iface.Read(packet)
}

func (n *netiface) Write(packet []byte) (int, error) {
	return n.iface.Write(packet)
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
