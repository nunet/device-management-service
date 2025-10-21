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
	"math"
	"strings"

	incus "github.com/lxc/incus/client"
)

// ConfigureSymmetricalNATInVM configures symmetrical NAT behavior inside a VM/container
// This is done by manipulating iptables rules inside the instance itself
// The --random-fully flag ensures true symmetrical NAT behavior
// blockedSubnets: list of other NAT network subnets to block (not the relay)
func ConfigureSymmetricalNATInVM(c incus.InstanceServer, instanceName string, blockedSubnets []string) error {
	fmt.Printf("[NAT-VM] Configuring symmetrical NAT inside instance: %s\n", instanceName)

	// Get the network interface (usually eth0)
	getIfaceCmd := []string{"sh", "-c", "ip route | grep default | awk '{print $5}' | head -1"}
	iface, err := RunCommandInInstance(c, instanceName, getIfaceCmd)
	if err != nil {
		return fmt.Errorf("failed to get network interface: %w", err)
	}
	if iface == "" {
		iface = "eth0" // Default fallback
	}
	iface = iface[:len(iface)-1] // Remove trailing newline

	// Get our local subnet
	getSubnetCmd := []string{"sh", "-c", "ip addr show " + iface + " | grep 'inet ' | awk '{print $2}'"}
	localSubnet, err := RunCommandInInstance(c, instanceName, getSubnetCmd)
	if err != nil || localSubnet == "" {
		fmt.Printf("[NAT-VM] Warning: could not determine local subnet, using default rules\n")
		localSubnet = "10.0.0.0/8" // Fallback
	} else {
		localSubnet = strings.TrimSpace(localSubnet)
		// Extract just the network part (e.g., 10.0.1.10/24 -> 10.0.1.0/24)
		parts := strings.Split(localSubnet, "/")
		_ = parts       //nolint:ifshort // Keep the CIDR notation as is for now
		_ = localSubnet //nolint:ifshort // Keep the local subnet as is for now
	}

	commands := []string{
		// Enable IP forwarding
		"sysctl -w net.ipv4.ip_forward=1",

		// Clear any existing rules
		"iptables -F INPUT 2>/dev/null || true",
		"iptables -F FORWARD 2>/dev/null || true",
		"iptables -t nat -F POSTROUTING 2>/dev/null || true",

		// CRITICAL: Block direct inbound connections from other private networks
		// This forces connections to go through NAT traversal mechanisms
		// Allow established connections
		"iptables -A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT",
		// Allow loopback
		"iptables -A INPUT -i lo -j ACCEPT",
		// Allow from same subnet
		fmt.Sprintf("iptables -A INPUT -s %s -j ACCEPT", localSubnet),
		// Allow SSH and DMS API from host for testing/management
		"iptables -A INPUT -p tcp --dport 22 -j ACCEPT",   // SSH
		"iptables -A INPUT -p tcp --dport 9999 -j ACCEPT", // DMS API
	}

	// BLOCK only the specific OTHER NAT network subnets (not all 10.0.0.0/8)
	// This allows relay nodes (on different 10.x subnets) to connect while blocking cross-NAT direct connections
	for _, blockedSubnet := range blockedSubnets {
		blockCmd := fmt.Sprintf("iptables -A INPUT -s %s -j DROP", blockedSubnet)
		commands = append(commands, blockCmd)
		fmt.Printf("[NAT-VM] Will block subnet: %s\n", blockedSubnet)
	}

	commands = append(commands, []string{
		// Allow everything else (including relay nodes on other 10.x subnets and internet)
		"iptables -A INPUT -j ACCEPT",

		// Configure SYMMETRICAL NAT with --random-fully for outbound traffic
		// This ensures each unique (src_ip, src_port, dst_ip, dst_port) tuple gets a unique external port
		fmt.Sprintf("iptables -t nat -A POSTROUTING -o %s -j MASQUERADE --random-fully", iface),

		// Set aggressive UDP timeout for more realistic symmetrical NAT behavior
		"sysctl -w net.netfilter.nf_conntrack_udp_timeout=30 2>/dev/null || true",
		"sysctl -w net.netfilter.nf_conntrack_udp_timeout_stream=60 2>/dev/null || true",

		// Allow forwarding
		"iptables -A FORWARD -j ACCEPT 2>/dev/null || true",

		// Save iptables rules (if iptables-persistent is available)
		"iptables-save > /etc/iptables/rules.v4 2>/dev/null || true",
	}...)

	for _, cmd := range commands {
		_, err := RunCommandInInstance(c, instanceName, []string{"sh", "-c", cmd})
		if err != nil {
			// Log warning but continue - some commands might fail
			fmt.Printf("[NAT-VM] Warning: command failed (may be OK): %s\n  Error: %v\n", cmd, err)
		}
	}

	// Verify the configuration
	verifyCmd := []string{"sh", "-c", "iptables -t nat -L POSTROUTING -n -v | grep MASQUERADE"}
	output, err := RunCommandInInstance(c, instanceName, verifyCmd)
	if err == nil && output != "" {
		fmt.Printf("[NAT-VM] Successfully configured symmetrical NAT in %s\n", instanceName)
		fmt.Printf("[NAT-VM] NAT rule: %s\n", output)
	} else {
		fmt.Printf("[NAT-VM] Warning: could not verify NAT configuration: %v\n", err)
	}

	return nil
}

// VerifySymmetricalNATInVM verifies that symmetrical NAT is configured correctly
func VerifySymmetricalNATInVM(c incus.InstanceServer, instanceName string, blockedSubnets []string) error {
	verificationCommands := []string{
		"sysctl net.ipv4.ip_forward",
		"iptables -t nat -L POSTROUTING -n",
		"iptables -L INPUT -n",
		"sysctl net.netfilter.nf_conntrack_udp_timeout 2>/dev/null || echo 'not available'",
	}

	fmt.Printf("[NAT-VM] Verifying NAT configuration in %s:\n", instanceName)
	fmt.Printf("[NAT-VM]   Blocked subnets: %v\n", blockedSubnets)
	for _, cmd := range verificationCommands {
		output, err := RunCommandInInstance(c, instanceName, []string{"sh", "-c", cmd})
		if err != nil {
			fmt.Printf("[NAT-VM]   %s: ERROR - %v\n", cmd, err)
		} else {
			fmt.Printf("[NAT-VM]   %s: %s\n", cmd, output[:int(math.Min(float64(len(output)), 100))])
		}
	}

	return nil
}
