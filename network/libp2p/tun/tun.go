// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package tun

import (
	"net"
	"strings"

	"github.com/songgao/water"
)

// TUN is a struct containing the fields necessary
// to configure a system TUN device. Access the
// internal TUN device through TUN.Iface
type TUN struct {
	Iface *water.Interface
	MTU   int
	Src   string
	Dst   string
}

// Apply configures the specified options for a TUN device.
func (t *TUN) Apply(opts ...Option) error {
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(t); err != nil {
			return err
		}
	}
	return nil
}

// Name returns the name of the TUN device.
func (t *TUN) Name() string {
	return t.Iface.Name()
}

func LocalInterfaces() ([]net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	return ifaces, nil
}

func JoinedNetworks() ([]string, error) {
	ifaces, err := LocalInterfaces()
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
