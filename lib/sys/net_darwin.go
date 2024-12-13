// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build darwin

//nolint:all // This is a stub file for darwin
package sys

import (
	"fmt"

	"github.com/songgao/water"
)

// NewTunTapInterface creates a new tun/tap interface
// persist is not used on darwin
func NewTunTapInterface(name string, mode TunTapMode, persist bool) (*NetInterface, error) {
	var intMode water.DeviceType = water.TAP

	if mode == NetTunMode {
		intMode = water.TUN
	}

	config := water.Config{
		DeviceType: intMode,
	}
	config.Name = name

	iface, err := water.New(config)
	if err != nil {
		return nil, fmt.Errorf("error creating interface: %w", err)
	}

	return &NetInterface{
		Iface: iface,
	}, nil
}

// UpNetInterface brings the network interface up
func (n *NetInterface) Up() error {
	return fmt.Errorf("up interface not available on darwin")
}

// DownNetInterface brings the network interface down
func (n *NetInterface) Down() error {
	return fmt.Errorf("down interface not available on darwin")
}

// DeleteNetInterface deletes the network interface
func (n *NetInterface) Delete() error {
	return fmt.Errorf("delete interface not available on darwin")
}

// SetMTU sets the MTU of the network interface
func (n *NetInterface) SetMTU(mtu int) error {
	return fmt.Errorf("set mtu not available on darwin")
}

// SetAddress sets the address of the network interface
func (n *NetInterface) SetAddress(address string) error {
	return fmt.Errorf("set interface address not available on darwin")
}

// AddRouteRule adds a route rule to the network interface
func (n *NetInterface) AddRouteRule(src, dst, gw string) error { 
 	return fmt.Errorf("add route not available on darwin")
}

// DelRoute deletes a route from the network interface
func (n *NetInterface) DelRoute(route string) error {
	return fmt.Errorf("delete route not available on darwin")
}

// ForwardingEnabled checks if IP forwarding is enabled
func ForwardingEnabled() (bool, error) {
	return false, fmt.Errorf("forwarding check not available on darwin")
}

// AddDNATRule adds a DNAT rule to iptables PRERROUTING chain
func AddDNATRule(protocol, sourceIP, sourcePort, destIP, destPort string) error {
	return fmt.Errorf("add dnat rule not available on darwin")
}

// DelDNATRule deletes a DNAT rule to iptables PRERROUTING chain if it exists
func DelDNATRule(protocol, sourceIP, sourcePort, destIP, destPort string) error {
	return fmt.Errorf("del dnat rule not available on darwin")
}

// AddForwardRule adds an ip:port FORWARD rule to iptables
func AddForwardRule(protocol, destIP, destPort string) error {
	return fmt.Errorf("add forward rule not available on darwin")
}

// DelForwardRule deletes an ip:port FORWARD rule if it exists
func DelForwardRule(protocol, destIP, destPort string) error {
	return fmt.Errorf("del forward rule not available on darwin")
}

// AddForwardIntRule adds a FORWARD between interfaces rule to iptables
func AddForwardIntRule(inInt, outInt string) error {
	return fmt.Errorf("add forward interface rule not available on darwin")
}

// DelForwardIntRule deletes a FORWARD between interfaces rule if it exists
func DelForwardIntRule(inInt, outInt string) error {
	return fmt.Errorf("del forward interface rule not available on darwin")
}

// AddMasqueradeRule adds a MASQUERADE rule to iptables POSTROUTING chain
func AddMasqueradeRule() error {
	return fmt.Errorf("add masquerade rule not available on darwin")
}

// DelMasqueradeRule deletes a MASQUERADE rule from the POSTROUTING chain if it exists
func DelMasqueradeRule() error {
	return fmt.Errorf("del masquerade rule not available on darwin")
}

// AddRelEstRule adds a RELATED,ESTABLISHED rule to specified chain of iptables
func AddRelEstRule(chain string) error {
	return fmt.Errorf("add related,established rule not available on darwin")
}

// DelRelEstRule deletes a RELATED,ESTABLISHED rule from the specified chain if it exists
func DelRelEstRule(chain string) error {
	return fmt.Errorf("del related,established rule not available on darwin")
}
