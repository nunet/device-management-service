// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package utils

import (
	"fmt"
	"os/exec"
	"strings"
)

// NATRouter represents a network namespace configured as a NAT router
type NATRouter struct {
	Name           string
	NetworkName    string
	Namespace      string
	InternalSubnet string
	VethIn         string
	VethOut        string
	RouterIP       string
	HostIP         string
}

// CreateSymmetricalNATRouter creates a network namespace that acts as a symmetrical NAT router
// for the specified Incus network
func CreateSymmetricalNATRouter(networkName string) (*NATRouter, error) {
	// Generate names
	nsName := fmt.Sprintf("nat-router-%s", networkName)
	vethIn := fmt.Sprintf("veth-%s-in", networkName)
	vethOut := fmt.Sprintf("veth-%s-out", networkName)

	fmt.Printf("[NAT-NS] Creating symmetrical NAT router in namespace: %s\n", nsName)

	// Get the Incus network subnet
	subnet, err := getIncusNetworkSubnet(networkName)
	if err != nil {
		return nil, fmt.Errorf("failed to get network subnet: %w", err)
	}

	// Parse subnet to get router IP (usually .1)
	routerIP, err := getRouterIPFromSubnet(subnet)
	if err != nil {
		return nil, fmt.Errorf("failed to determine router IP: %w", err)
	}

	// Generate a unique host IP for the veth pair
	hostIP := generateHostIP(networkName)

	router := &NATRouter{
		Name:           nsName,
		NetworkName:    networkName,
		Namespace:      nsName,
		InternalSubnet: subnet,
		VethIn:         vethIn,
		VethOut:        vethOut,
		RouterIP:       routerIP,
		HostIP:         hostIP,
	}

	// Step 1: Create network namespace
	if err := createNetworkNamespace(nsName); err != nil {
		return nil, err
	}

	// Step 2: Create veth pair
	if err := createVethPair(vethIn, vethOut); err != nil {
		deleteNetworkNamespace(nsName) //nolint:errcheck // Cleanup
		return nil, err
	}

	// Step 3: Move veth-in to namespace
	if err := moveVethToNamespace(vethIn, nsName); err != nil {
		deleteVethPair(vethOut)        //nolint:errcheck // Cleanup
		deleteNetworkNamespace(nsName) //nolint:errcheck // Cleanup
		return nil, err
	}

	// Step 4: Configure veth-in inside namespace (connects to Incus bridge)
	if err := configureVethInNamespace(nsName, vethIn, routerIP, subnet); err != nil {
		deleteVethPair(vethOut)        //nolint:errcheck // Cleanup
		deleteNetworkNamespace(nsName) //nolint:errcheck // Cleanup
		return nil, err
	}

	// Step 5: Configure veth-out on host
	if err := configureVethOnHost(vethOut, hostIP); err != nil {
		deleteVethPair(vethOut)        //nolint:errcheck // Cleanup
		deleteNetworkNamespace(nsName) //nolint:errcheck // Cleanup
		return nil, err
	}

	// Step 6: Connect veth-in to Incus bridge
	bridgeName := networkName // Incus usually names bridge same as network
	if err := connectVethToBridge(vethIn, bridgeName, nsName); err != nil {
		// Non-fatal - log warning
		fmt.Printf("[NAT-NS] Warning: failed to connect veth to bridge: %v\n", err)
	}

	// Step 7: Enable IP forwarding in namespace
	if err := enableForwardingInNamespace(nsName); err != nil {
		deleteVethPair(vethOut)        //nolint:errcheck // Cleanup
		deleteNetworkNamespace(nsName) //nolint:errcheck // Cleanup
		return nil, err
	}

	// Step 8: Configure symmetrical NAT in namespace
	if err := configureSymmetricalNATInNamespace(nsName, vethOut); err != nil {
		deleteVethPair(vethOut)        //nolint:errcheck // Cleanup
		deleteNetworkNamespace(nsName) //nolint:errcheck // Cleanup
		return nil, err
	}

	// Step 9: Configure routing in namespace
	if err := configureRoutingInNamespace(nsName, hostIP); err != nil {
		deleteVethPair(vethOut)        //nolint:errcheck // Cleanup
		deleteNetworkNamespace(nsName) //nolint:errcheck // Cleanup
		return nil, err
	}

	fmt.Printf("[NAT-NS] Successfully created symmetrical NAT router: %s\n", nsName)
	fmt.Printf("[NAT-NS]   Internal subnet: %s\n", subnet)
	fmt.Printf("[NAT-NS]   Router IP: %s\n", routerIP)
	fmt.Printf("[NAT-NS]   External IP: %s\n", hostIP)

	return router, nil
}

// DeleteSymmetricalNATRouter removes the NAT router namespace and associated resources
func DeleteSymmetricalNATRouter(router *NATRouter) error {
	if router == nil {
		return nil
	}

	fmt.Printf("[NAT-NS] Deleting NAT router: %s\n", router.Name)

	// Delete veth pair (this also removes it from namespace)
	if err := deleteVethPair(router.VethOut); err != nil {
		fmt.Printf("[NAT-NS] Warning: failed to delete veth pair: %v\n", err)
	}

	// Delete namespace
	if err := deleteNetworkNamespace(router.Namespace); err != nil {
		fmt.Printf("[NAT-NS] Warning: failed to delete namespace: %v\n", err)
	}

	fmt.Printf("[NAT-NS] NAT router deleted: %s\n", router.Name)
	return nil
}

// Helper functions

func getIncusNetworkSubnet(networkName string) (string, error) {
	// Use incus command to get network info
	cmd := exec.Command("incus", "network", "get", networkName, "ipv4.address")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get network subnet: %w, output: %s", err, string(out))
	}
	subnet := strings.TrimSpace(string(out))
	if subnet == "" {
		return "", fmt.Errorf("network %s has no ipv4.address configured", networkName)
	}
	return subnet, nil
}

func getRouterIPFromSubnet(subnet string) (string, error) {
	// Extract IP from CIDR (e.g., "10.0.1.1/24" -> "10.0.1.1")
	parts := strings.Split(subnet, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid subnet format: %s", subnet)
	}
	return parts[0], nil
}

func generateHostIP(networkName string) string {
	// Generate a unique host IP based on network name
	// Use 192.168.100.x range for NAT router connections
	hash := 0
	for _, c := range networkName {
		hash += int(c)
	}
	octet := (hash % 200) + 10 // Range: 192.168.100.10 - 192.168.100.209
	return fmt.Sprintf("192.168.100.%d/30", octet)
}

func createNetworkNamespace(name string) error {
	cmd := exec.Command("ip", "netns", "add", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create namespace %s: %w, output: %s", name, err, string(out))
	}
	fmt.Printf("[NAT-NS] Created namespace: %s\n", name)
	return nil
}

func deleteNetworkNamespace(name string) error {
	cmd := exec.Command("ip", "netns", "del", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete namespace %s: %w, output: %s", name, err, string(out))
	}
	return nil
}

func createVethPair(vethIn, vethOut string) error {
	cmd := exec.Command("ip", "link", "add", vethIn, "type", "veth", "peer", "name", vethOut)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create veth pair: %w, output: %s", err, string(out))
	}
	fmt.Printf("[NAT-NS] Created veth pair: %s <-> %s\n", vethIn, vethOut)
	return nil
}

func deleteVethPair(vethName string) error {
	cmd := exec.Command("ip", "link", "del", vethName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete veth: %w, output: %s", err, string(out))
	}
	return nil
}

func moveVethToNamespace(veth, namespace string) error {
	cmd := exec.Command("ip", "link", "set", veth, "netns", namespace)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to move veth to namespace: %w, output: %s", err, string(out))
	}
	fmt.Printf("[NAT-NS] Moved %s to namespace %s\n", veth, namespace)
	return nil
}

func configureVethInNamespace(namespace, veth, _, subnet string) error {
	commands := [][]string{
		{"ip", "netns", "exec", namespace, "ip", "addr", "add", subnet, "dev", veth},
		{"ip", "netns", "exec", namespace, "ip", "link", "set", veth, "up"},
		{"ip", "netns", "exec", namespace, "ip", "link", "set", "lo", "up"},
	}

	for _, cmdArgs := range commands {
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to configure veth in namespace: %w, cmd: %v, output: %s",
				err, cmdArgs, string(out))
		}
	}

	fmt.Printf("[NAT-NS] Configured %s in namespace with IP %s\n", veth, subnet)
	return nil
}

func configureVethOnHost(veth, ipAddr string) error {
	commands := [][]string{
		{"ip", "addr", "add", ipAddr, "dev", veth},
		{"ip", "link", "set", veth, "up"},
	}

	for _, cmdArgs := range commands {
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to configure veth on host: %w, cmd: %v, output: %s",
				err, cmdArgs, string(out))
		}
	}

	fmt.Printf("[NAT-NS] Configured %s on host with IP %s\n", veth, ipAddr)
	return nil
}

func connectVethToBridge(veth, bridge, namespace string) error {
	// The veth-in is inside the namespace, we need to attach it to the Incus bridge
	// This is done by setting the master of the veth to the bridge
	// Since veth is in namespace, we run from within namespace
	cmd := exec.Command("ip", "netns", "exec", namespace, "ip", "link", "set", veth, "master", bridge)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Try alternative: attach from host side before moving to namespace
		// This is expected to fail in some cases, it's OK
		return fmt.Errorf("info: could not attach to bridge (this may be OK): %w, output: %s", err, string(out))
	}

	fmt.Printf("[NAT-NS] Connected %s to bridge %s\n", veth, bridge)
	return nil
}

func enableForwardingInNamespace(namespace string) error {
	cmd := exec.Command("ip", "netns", "exec", namespace, "sysctl", "-w", "net.ipv4.ip_forward=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to enable forwarding: %w, output: %s", err, string(out))
	}
	fmt.Printf("[NAT-NS] Enabled IP forwarding in namespace %s\n", namespace)
	return nil
}

func configureSymmetricalNATInNamespace(namespace, outInterface string) error {
	// Configure SYMMETRICAL NAT with --random-fully flag
	// This ensures each unique (src_ip, src_port, dst_ip, dst_port) tuple gets unique external port
	commands := [][]string{
		// Add MASQUERADE with --random-fully for symmetrical behavior
		{
			"ip", "netns", "exec", namespace, "iptables", "-t", "nat", "-A", "POSTROUTING",
			"-o", outInterface, "-j", "MASQUERADE", "--random-fully",
		},

		// Configure aggressive conntrack timeouts for more realistic symmetrical NAT
		{
			"ip", "netns", "exec", namespace, "sysctl", "-w",
			"net.netfilter.nf_conntrack_udp_timeout=30",
		},
		{
			"ip", "netns", "exec", namespace, "sysctl", "-w",
			"net.netfilter.nf_conntrack_udp_timeout_stream=60",
		},

		// Allow forwarding
		{"ip", "netns", "exec", namespace, "iptables", "-A", "FORWARD", "-j", "ACCEPT"},
	}

	for _, cmdArgs := range commands {
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			// Some commands might fail in containers without full privileges
			fmt.Printf("[NAT-NS] Warning: command failed (may be OK): %v\n  Output: %s\n",
				cmdArgs, string(out))
			return err
		}
	}

	fmt.Printf("[NAT-NS] Configured symmetrical NAT in namespace %s\n", namespace)
	return nil
}

func configureRoutingInNamespace(namespace, hostIP string) error {
	// Extract just the IP without CIDR
	hostIPOnly := strings.Split(hostIP, "/")[0]

	// Add default route through the host
	cmd := exec.Command("ip", "netns", "exec", namespace, "ip", "route", "add",
		"default", "via", hostIPOnly)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to configure routing: %w, output: %s", err, string(out))
	}

	fmt.Printf("[NAT-NS] Configured default route in namespace %s via %s\n", namespace, hostIPOnly)
	return nil
}

// CheckNamespaceExists checks if a network namespace exists
func CheckNamespaceExists(name string) bool {
	cmd := exec.Command("ip", "netns", "list")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), name)
}
