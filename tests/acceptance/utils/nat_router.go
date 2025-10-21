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

	incus "github.com/lxc/incus/client"
	"github.com/lxc/incus/shared/api"
)

// NATRouterContainer represents a container that acts as a NAT router
type NATRouterContainer struct {
	Name            string
	Client          incus.InstanceServer
	ExternalNetwork string // e.g., "incusbr0"
	InternalNetwork string // e.g., "private-net-1"
	InternalIP      string // e.g., "192.168.1.1"
	InternalSubnet  string // e.g., "192.168.1.0/24"
}

// CreateNATRouterContainer creates a container that acts as a NAT router
func CreateNATRouterContainer(c incus.InstanceServer, name, externalNet, internalNet, internalSubnet string, blockedSubnets []string) (*NATRouterContainer, error) {
	// Create the container
	req := api.InstancesPost{
		Name: name,
		InstancePut: api.InstancePut{
			Architecture: "x86_64",
			Config: map[string]string{
				"security.nesting":                     "true",
				"security.syscalls.intercept.mknod":    "true",
				"security.syscalls.intercept.setxattr": "true",
				"boot.host_shutdown_action":            "force-stop",
			},
			Devices: map[string]map[string]string{
				"eth0": {
					"type":    "nic",
					"network": externalNet,
					"name":    "eth0",
				},
				"eth1": {
					"type":         "nic",
					"network":      internalNet,
					"name":         "eth1",
					"ipv4.address": internalSubnet[:len(internalSubnet)-3] + "1", // .1 is router
				},
			},
			Ephemeral: true,
		},
		Source: api.InstanceSource{
			Type:  "image",
			Alias: DefaultImageContainer,
		},
	}

	op, err := c.CreateInstance(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create NAT router container: %w", err)
	}

	if err := op.Wait(); err != nil {
		return nil, fmt.Errorf("failed to wait for NAT router creation: %w", err)
	}

	// Start the container
	startReq := api.InstanceStatePut{
		Action:   "start",
		Timeout:  -1,
		Force:    true,
		Stateful: false,
	}

	startOp, err := c.UpdateInstanceState(name, startReq, "")
	if err != nil {
		return nil, fmt.Errorf("failed to start NAT router: %w", err)
	}

	if err := startOp.Wait(); err != nil {
		return nil, fmt.Errorf("failed to wait for NAT router start: %w", err)
	}

	// Wait for container to be ready
	err = WaitForInstanceReady(c, name, 30000000000) // 30 seconds
	if err != nil {
		return nil, fmt.Errorf("NAT router not ready: %w", err)
	}

	router := &NATRouterContainer{
		Name:            name,
		Client:          c,
		ExternalNetwork: externalNet,
		InternalNetwork: internalNet,
		InternalIP:      internalSubnet[:len(internalSubnet)-3] + "1",
		InternalSubnet:  internalSubnet,
	}

	// Configure the NAT router with blocked subnets
	if err := configureNATRouter(router, blockedSubnets); err != nil {
		return nil, err
	}

	return router, nil
}

// configureNATRouter sets up IP forwarding and NAT rules in the router container
func configureNATRouter(router *NATRouterContainer, blockedPrivateSubnets []string) error {
	commands := []string{
		// Enable IP forwarding
		"sysctl -w net.ipv4.ip_forward=1",
		"echo 'net.ipv4.ip_forward=1' >> /etc/sysctl.conf",

		// Clear any existing rules
		"iptables -F FORWARD",
		"iptables -t nat -F POSTROUTING",
	}

	// Block traffic TO other private NAT networks (critical for isolation!)
	for _, blockedSubnet := range blockedPrivateSubnets {
		blockCmd := fmt.Sprintf("iptables -A FORWARD -d %s -j DROP", blockedSubnet)
		commands = append(commands, blockCmd)
	}

	// CRITICAL: Block direct access FROM external network to internal network
	// This forces AutoNAT probes to fail (can't reach private IPs directly)
	// Only NATed traffic (ESTABLISHED connections) can come back
	commands = append(commands, []string{
		// Block NEW connections from external (eth0) to internal network
		// This makes internal hosts truly unreachable from outside
		"iptables -A FORWARD -i eth0 -o eth1 -m conntrack --ctstate NEW -j DROP",

		// Allow forwarding from internal to external (internet access)
		"iptables -A FORWARD -i eth1 -o eth0 -j ACCEPT",

		// Allow established connections back (responses to NATed requests)
		"iptables -A FORWARD -i eth0 -o eth1 -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT",

		// Configure SYMMETRICAL NAT with --random-fully
		"iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE --random-fully",

		// Set aggressive conntrack timeouts for symmetrical NAT behavior
		"sysctl -w net.netfilter.nf_conntrack_udp_timeout=30",
		"sysctl -w net.netfilter.nf_conntrack_udp_timeout_stream=60",
	}...)

	for _, cmd := range commands {
		_, err := RunCommandInInstance(router.Client, router.Name, []string{"sh", "-c", cmd})
		if err != nil {
			return fmt.Errorf("failed to configure NAT router with cmd '%s': %w", cmd, err)
		}
	}

	return nil
}

// CreatePrivateNetwork creates a private network for nodes behind NAT
// blockedSubnets are other private networks that should not be reachable from this network
func CreatePrivateNetwork(c incus.InstanceServer, networkName, subnet string, blockedSubnets []string) error {
	// Check if network exists
	_, _, err := c.GetNetwork(networkName)
	if err == nil {
		if err := c.DeleteNetwork(networkName); err != nil {
			return fmt.Errorf("failed to delete existing network: %w", err)
		}
	}

	// Create private network WITHOUT NAT (router handles it)
	// NOTE: We cannot use raw.iptables on Incus networks (not supported)
	// Cross-NAT blocking is done via NAT router container's iptables
	// The key is ensuring all traffic goes THROUGH the NAT router (via default gateway)
	networkPost := api.NetworksPost{
		Name: networkName,
		NetworkPut: api.NetworkPut{
			Config: map[string]string{
				"ipv4.address": subnet,  // Fixed subnet
				"ipv4.nat":     "false", // NO NAT - router does this
				"ipv4.dhcp":    "true",  // DHCP for clients
				"ipv6.address": "none",  // Disable IPv6
				// ipv4.routing defaults to true - needed so NAT router can reach incusbr0
			},
		},
	}

	if err := c.CreateNetwork(networkPost); err != nil {
		return fmt.Errorf("failed to create private network: %w", err)
	}

	// Add host-level iptables rules to prevent Incus host from routing between private networks
	if len(blockedSubnets) > 0 {
		if err := AddHostCrossNATBlocking(subnet, blockedSubnets); err != nil {
			return fmt.Errorf("failed to add host firewall rules: %w", err)
		}
	}

	return nil
}

// AddHostCrossNATBlocking adds iptables rules on the Incus host to prevent routing between NAT networks
func AddHostCrossNATBlocking(sourceSubnet string, blockedSubnets []string) error {
	// Try to use iptables-legacy first (works better with capabilities)
	// Fall back to iptables if legacy not available
	iptablesCmd := "iptables-legacy"
	if _, err := exec.LookPath(iptablesCmd); err != nil {
		iptablesCmd = "iptables"
	}

	for _, blockedSubnet := range blockedSubnets {
		// Use a unique comment marker for easy cleanup
		comment := "NUNET-NAT-TEST-ISOLATION"

		// Block traffic from source subnet to blocked subnet
		rule := fmt.Sprintf("%s -I FORWARD 1 -s %s -d %s -m comment --comment '%s' -j DROP",
			iptablesCmd, sourceSubnet, blockedSubnet, comment)

		cmd := exec.Command("sh", "-c", rule)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to add iptables rule: %v, output: %s", err, string(output))
		}
	}

	return nil
}

// RemoveHostCrossNATBlocking removes all iptables rules added by AddHostCrossNATBlocking
func RemoveHostCrossNATBlocking() error {
	// Try iptables-legacy first, fall back to iptables
	iptablesCmd := "iptables-legacy"
	if _, err := exec.LookPath(iptablesCmd); err != nil {
		iptablesCmd = "iptables"
	}

	// List all rules with our comment, convert -A to -D, and delete them
	cleanupCmd := fmt.Sprintf(`%s-save | grep 'NUNET-NAT-TEST-ISOLATION' | sed 's/-A /-D /' | while read rule; do %s $rule 2>/dev/null || true; done`,
		iptablesCmd, iptablesCmd)

	cmd := exec.Command("sh", "-c", cleanupCmd)
	_ = cmd.Run() // Ignore errors - cleanup is best effort

	return nil
}

// CreateInstanceBehindNAT creates an instance on a private network with NAT router as gateway
func CreateInstanceBehindNAT(c incus.InstanceServer, instanceType, name, privateNetwork, gatewayIP string) error {
	// Create instance on private network
	if err := CreateInstanceWithNetwork(c, instanceType, name, privateNetwork); err != nil {
		return err
	}

	// Wait for instance to be ready
	if err := WaitForInstanceReady(c, name, 60000000000); err != nil { // 60 seconds
		return fmt.Errorf("instance not ready: %w", err)
	}

	// Configure default route through NAT router

	// Extract subnet from gateway IP (e.g., 172.16.1.1 -> 172.16.1.0/24)
	parts := strings.Split(gatewayIP, ".")
	if len(parts) != 4 {
		return fmt.Errorf("invalid gateway IP: %s", gatewayIP)
	}
	gatewaySubnet := fmt.Sprintf("%s.%s.%s.0/24", parts[0], parts[1], parts[2])

	commands := []string{
		// Wait for network interface to be ready
		"sleep 2",
		// Ensure we can reach the gateway (should be on same subnet via DHCP)
		fmt.Sprintf("ip route add %s dev $(ip route | grep %s | head -1 | awk '{print $3}') 2>/dev/null || true", gatewaySubnet, gatewaySubnet),
		// Ping gateway to verify reachability
		fmt.Sprintf("ping -c 1 -W 2 %s || echo 'Gateway not yet reachable'", gatewayIP),
		// Remove default route if exists
		"ip route del default 2>/dev/null || true",
		// Add route through NAT router
		fmt.Sprintf("ip route add default via %s", gatewayIP),
		// Verify route
		"ip route show",
	}

	for _, cmd := range commands {
		_, _ = RunCommandInInstance(c, name, []string{"sh", "-c", cmd})
		// Ignore errors - route setup is best effort
	}

	return nil
}

// DeleteNATRouterContainer deletes a NAT router container
func DeleteNATRouterContainer(router *NATRouterContainer) error {
	if router == nil {
		return nil
	}
	return DeleteInstance(router.Client, router.Name)
}
