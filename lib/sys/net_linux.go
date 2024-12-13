// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build linux

package sys

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"syscall"

	"github.com/songgao/water"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

// NewTunTapInterface creates a new tun/tap interface
func NewTunTapInterface(name string, mode TunTapMode, persist bool) (*NetInterface, error) {
	var intMode water.DeviceType = water.TAP

	if mode == NetTunMode {
		intMode = water.TUN
	}

	config := water.Config{
		DeviceType: intMode,
	}
	config.Name = name

	if persist {
		config.PlatformSpecificParams.Persist = true
	}

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
	link, err := netlink.LinkByName(n.Iface.Name())
	if err != nil {
		return fmt.Errorf("error getting network interface by name: %w", err)
	}

	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("error setting network interface up: %w", err)
	}

	return nil
}

// DownNetInterface brings the network interface down
func (n *NetInterface) Down() error {
	link, err := netlink.LinkByName(n.Iface.Name())
	if err != nil {
		return fmt.Errorf("error getting network interface by name: %w", err)
	}

	if err := netlink.LinkSetDown(link); err != nil {
		return fmt.Errorf("error setting network interface down: %w", err)
	}

	return nil
}

// DeleteNetInterface deletes the network interface
func (n *NetInterface) Delete() error {
	link, err := netlink.LinkByName(n.Iface.Name())
	if err != nil {
		return fmt.Errorf("error getting network interface by name: %w", err)
	}

	if err := netlink.LinkDel(link); err != nil {
		return fmt.Errorf("error deleting network interface: %w", err)
	}

	return nil
}

// SetMTU sets the MTU of the network interface
func (n *NetInterface) SetMTU(mtu int) error {
	link, err := netlink.LinkByName(n.Iface.Name())
	if err != nil {
		return fmt.Errorf("error getting network interface by name: %w", err)
	}

	if err := netlink.LinkSetMTU(link, mtu); err != nil {
		return fmt.Errorf("error setting network interface mtu: %w", err)
	}

	return nil
}

// SetAddress sets the address of the network interface in CIDR notation
func (n *NetInterface) SetAddress(address string) error {
	addr, err := netlink.ParseAddr(address)
	if err != nil {
		return fmt.Errorf("error parsing address: %w", err)
	}

	link, err := netlink.LinkByName(n.Iface.Name())
	if err != nil {
		return fmt.Errorf("error getting network interface by name: %w", err)
	}

	if err := netlink.AddrAdd(link, addr); err != nil {
		return fmt.Errorf("error setting network interface address: %w", err)
	}

	return nil
}

// AddRouteRule adds an ip route rule to the network interface
func (n *NetInterface) AddRouteRule(src, dst, gw string) error {
	link, err := netlink.LinkByName(n.Iface.Name())
	if err != nil {
		return fmt.Errorf("error getting network interface by name: %w", err)
	}

	var gwIP net.IP
	if gw != "" {
		err = gwIP.UnmarshalText([]byte(gw))
		if err != nil {
			return fmt.Errorf("error parsing gw: %w", err)
		}
	}

	var destNet *net.IPNet
	if dst != "" {
		_, destNet, err = net.ParseCIDR(dst)
		if err != nil {
			return fmt.Errorf("error parsing dest net: %w", err)
		}
	}

	var srcIP net.IP
	if src != "" {
		err = srcIP.UnmarshalText([]byte(src))
		if err != nil {
			return fmt.Errorf("error parsing src: %w", err)
		}
	}

	err = netlink.RouteAdd(&netlink.Route{
		LinkIndex: link.Attrs().Index,
		Src:       srcIP,
		Dst:       destNet,
		Gw:        gwIP,
		Priority:  3000,
	})
	if err != nil {
		return fmt.Errorf("error adding route: %w", err)
	}

	return nil
}

// DelRoute deletes a route from the network interface
func (n *NetInterface) DelRoute(route string) error {
	link, err := netlink.LinkByName(n.Iface.Name())
	if err != nil {
		return fmt.Errorf("error getting network interface by name: %w", err)
	}

	netRoute, err := netlink.ParseIPNet(route)
	if err != nil {
		return fmt.Errorf("error parsing route: %w", err)
	}

	return netlink.RouteDel(&netlink.Route{
		LinkIndex: link.Attrs().Index,
		Dst:       netRoute,
		Priority:  3000,
	})
}

// ForwardingEnabled checks if IP forwarding is enabled
func ForwardingEnabled() (bool, error) {
	data, err := os.ReadFile("/proc/sys/net/ipv4/ip_forward")
	if err != nil {
		return false, fmt.Errorf("error reading ip_forward: %v", err)
	}
	if string(data) == "1\n" {
		return true, nil
	}

	return false, nil
}

// AddDNATRule adds a DNAT rule to iptables PRERROUTING chain
func AddDNATRule(protocol, sourceIP, sourcePort, destIP, destPort string) error {
	args := []string{
		"PREROUTING", "-t", "nat",
		"-d", sourceIP, "-p", protocol,
		"--dport", sourcePort, "-j", "DNAT",
		"--to-destination", destIP + ":" + destPort,
	}
	if !iptRuleExist(args...) {
		err := iptAppendRule(args...)
		if err != nil {
			return fmt.Errorf("error adding DNAT rule: %w", err)
		}
	}
	return nil
}

// DelDNATRule deletes a DNAT rule to iptables PRERROUTING chain if it exists
func DelDNATRule(protocol, sourceIP, sourcePort, destIP, destPort string) error {
	args := []string{
		"PREROUTING", "-t", "nat",
		"-d", sourceIP, "-p", protocol,
		"--dport", sourcePort, "-j", "DNAT",
		"--to-destination", destIP + ":" + destPort,
	}
	if iptRuleExist(args...) {
		err := iptDeleteRule(args...)
		if err != nil {
			return fmt.Errorf("error deleting DNAT rule: %w", err)
		}
	}
	return nil
}

// AddForwardRule adds an ip:port FORWARD rule to iptables
func AddForwardRule(protocol, destIP, destPort string) error {
	args := []string{
		"FORWARD", "-t", "filter",
		"-p", protocol, "-d", destIP,
		"--dport", destPort, "-j", "ACCEPT",
	}
	if !iptRuleExist(args...) {
		err := iptAppendRule(args...)
		if err != nil {
			return fmt.Errorf("error adding forward rule: %w", err)
		}
	}
	return nil
}

// DelForwardRule deletes an ip:port FORWARD rule if it exists
func DelForwardRule(protocol, destIP, destPort string) error {
	args := []string{
		"FORWARD", "-t", "filter",
		"-p", protocol, "-d", destIP,
		"--dport", destPort, "-j", "ACCEPT",
	}
	if iptRuleExist(args...) {
		err := iptDeleteRule(args...)
		if err != nil {
			return fmt.Errorf("error adding froward rule: %w", err)
		}
	}
	return nil
}

// AddForwardIntRule adds a FORWARD between interfaces rule to iptables
func AddForwardIntRule(inInt, outInt string) error {
	args := []string{
		"FORWARD", "-t", "filter",
		"-i", inInt, "-o", outInt, "-j", "ACCEPT",
	}
	if !iptRuleExist(args...) {
		err := iptAppendRule(args...)
		if err != nil {
			return fmt.Errorf("error adding interface forward rule: %w", err)
		}
	}
	return nil
}

// DelForwardIntRule deletes a FORWARD between interfaces rule if it exists
func DelForwardIntRule(inInt, outInt string) error {
	args := []string{
		"FORWARD", "-t", "filter",
		"-i", inInt, "-o", outInt, "-j", "ACCEPT",
	}
	if iptRuleExist(args...) {
		err := iptDeleteRule(args...)
		if err != nil {
			return fmt.Errorf("error deleting interface forward rule: %w", err)
		}
	}
	return nil
}

// AddMasqueradeRule adds a MASQUERADE rule to iptables POSTROUTING chain
func AddMasqueradeRule() error {
	args := []string{"POSTROUTING", "-t", "nat", "-j", "MASQUERADE"}

	if !iptRuleExist(args...) {
		err := iptAppendRule(args...)
		if err != nil {
			return fmt.Errorf("error adding rule: %w", err)
		}
	}
	return nil
}

// DelMasqueradeRule deletes a MASQUERADE rule from the POSTROUTING chain if it exists
func DelMasqueradeRule() error {
	args := []string{"POSTROUTING", "-t", "nat", "-j", "MASQUERADE"}

	if iptRuleExist(args...) {
		err := iptDeleteRule(args...)
		if err != nil {
			return fmt.Errorf("error deleting masquerade rule: %w", err)
		}
	}
	return nil
}

// AddRelEstRule adds a RELATED,ESTABLISHED rule to specified chain of iptables
func AddRelEstRule(chain string) error {
	args := []string{
		chain, "-t", "filter",
		"-m", "conntrack", "--ctstate",
		"RELATED,ESTABLISHED", "-j", "ACCEPT",
	}

	if !iptRuleExist(args...) {
		err := iptAppendRule(args...)
		if err != nil {
			return fmt.Errorf("error adding related,established rule: %w", err)
		}
	}
	return nil
}

// DelRelEstRule deletes a RELATED,ESTABLISHED rule from the specified chain if it exists
func DelRelEstRule(chain string) error {
	args := []string{
		chain, "-t", "filter",
		"-m", "conntrack", "--ctstate",
		"RELATED,ESTABLISHED", "-j", "ACCEPT",
	}

	if iptRuleExist(args...) {
		err := iptDeleteRule(args...)
		if err != nil {
			return fmt.Errorf("error deleting related,established rule: %w", err)
		}
	}
	return nil
}

// iptDeleteRule deletes the specified rule from the iptables chain
func iptDeleteRule(rule ...string) error {
	out, err := execCmd("iptables", append([]string{"-D"}, rule...))
	if err != nil {
		return fmt.Errorf("error deleting rule: %w, output: %s", err, out)
	}
	return nil
}

func iptAppendRule(rule ...string) error {
	out, err := execCmd("iptables", append([]string{"-A"}, rule...))
	if err != nil {
		return fmt.Errorf("error appending rule: %w, output: %s", err, out)
	}
	return nil
}

func iptRuleExist(rule ...string) bool {
	_, err := execCmd("iptables", append([]string{"-C"}, rule...))
	return err == nil
}

func execCmd(command string, args []string) (string, error) {
	cmd := exec.Command(command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		AmbientCaps: []uintptr{
			unix.CAP_NET_ADMIN,
		},
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("failed to execute command: %q: %w", command, err)
	}
	return string(output), nil
}
