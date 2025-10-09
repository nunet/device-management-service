// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package utils

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	incus "github.com/lxc/incus/client"
	"github.com/lxc/incus/shared/api"
	"gitlab.com/nunet/device-management-service/tests/acceptance/config"
	"golang.org/x/sync/errgroup"
)

const (
	DefaultImageContainer = "ubuntu-acc-test-container"
	DefaultImageVM        = "ubuntu-acc-test-vm"
	DefaultVMPrefix       = "test"
	LocalTarget           = "local"
	ContainerType         = "container"
	VMType                = "vm"
)

func getInstanceType() string {
	typ := os.Getenv("INSTANCE_TYPE")
	if typ != ContainerType {
		return VMType
	}
	return ContainerType
}

func WaitForInstanceReady(c incus.InstanceServer, name string, timeout time.Duration) error {
	if getInstanceType() != VMType {
		// Containers don't need additional check
		return nil
	}

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		inst, _, err := c.GetInstance(name)
		if err != nil {
			return fmt.Errorf("failed to get instance: %w", err)
		}

		// Wait for instance running
		if inst.Status != "Running" {
			// Wait and retry
			time.Sleep(1 * time.Second)
			continue
		}

		// Try pushing a small test file to verify if the incus agent is responsive
		args := incus.InstanceFileArgs{
			Content:   bytes.NewReader([]byte("ping")),
			Mode:      0o644,
			Type:      "file",
			WriteMode: "overwrite",
		}

		err = c.CreateInstanceFile(name, "/tmp/_agent_check.txt", args)
		if err == nil {
			return nil // incus agent is ready
		}

		if !strings.Contains(err.Error(), "VM agent isn't currently running") &&
			!strings.Contains(err.Error(), "Instance is not running") {
			return fmt.Errorf("unexpected error while checking agent status: %w", err)
		}

		// Wait and retry
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("VM agent did not respond within %s", timeout)
}

func ConnectIncus(target, clientCert, clientKey, serverCert string) (incus.InstanceServer, error) {
	if target == LocalTarget {
		return incus.ConnectIncusUnix("", nil)
	}

	connectionArgs := &incus.ConnectionArgs{
		TLSClientCert:      clientCert,
		TLSClientKey:       clientKey,
		TLSServerCert:      serverCert,
		InsecureSkipVerify: true, // for dev
	}

	return incus.ConnectIncus(target, connectionArgs)
}

func CreateInstance(c incus.InstanceServer, instanceType, name string) error {
	var req api.InstancesPost
	switch instanceType {
	case ContainerType:
		req = api.InstancesPost{
			Name: name,
			InstancePut: api.InstancePut{
				Architecture: "x86_64",
				Config: map[string]string{
					"security.nesting":                     "true", // allows Docker
					"security.syscalls.intercept.mknod":    "true",
					"security.syscalls.intercept.setxattr": "true",
					"boot.host_shutdown_action":            "force-stop",
				},
				Devices:   map[string]map[string]string{},
				Ephemeral: true,
			},
			Source: api.InstanceSource{
				Type:  "image",
				Alias: DefaultImageContainer,
			},
		}
	default:
		// VM instance type
		req = api.InstancesPost{
			Name: name,
			InstancePut: api.InstancePut{
				Architecture: "x86_64",
				Config: map[string]string{
					"boot.host_shutdown_action": "force-stop",
					"limits.cpu":                "4",
					"limits.memory":             "2GiB",
				},
				Devices: map[string]map[string]string{
					"root": {
						"type": "disk",
						"path": "/",
						"pool": "default", // use "incus storage list" to verify if the default pool exists
					},
				},
				Ephemeral: true,
			},
			Source: api.InstanceSource{
				Type:  "image",
				Alias: DefaultImageVM,
			},
			Type: "virtual-machine",
		}
	}
	op, err := c.CreateInstance(req)
	if err != nil {
		return fmt.Errorf("failed to create instance: %w", err)
	}

	if err := op.Wait(); err != nil {
		return fmt.Errorf("failed to wait for instance creation: %w", err)
	}

	startReq := api.InstanceStatePut{
		Action:   "start",
		Timeout:  -1,
		Force:    true,
		Stateful: false,
	}

	startOp, err := c.UpdateInstanceState(name, startReq, "")
	if err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}
	return startOp.Wait()
}

func findVMNetworkMainInterface(c incus.InstanceServer, name string) (string, error) {
	// Find main network interface (exclude 'lo')
	ifaceCmd := []string{"sh", "-c", "ip route | grep default | awk '{print $5}'"}
	ifaceOut, err := RunCommandInInstance(c, name, ifaceCmd)
	if err != nil {
		return "", fmt.Errorf("failed to detect network interface: %v", err)
	}
	iface := strings.TrimSpace(ifaceOut)
	return iface, nil
}

// ConfigureVMNetworkingForQUIC applies network optimizations for QUIC connections in VMs
// This addresses the root causes of QUIC connection failures in VM environments:
// - Increases UDP timeouts for QUIC's connectionless nature
// - Optimizes network buffers for QUIC's multiplexed streams
// - Disables network interface optimizations that interfere with QUIC
// - Configures firewall rules for QUIC traffic
func ConfigureVMNetworkingForQUIC(c incus.InstanceServer, name string) error {
	iface, err := findVMNetworkMainInterface(c, name)
	if err != nil {
		return err
	}

	// Network optimizations for QUIC in VM environments
	networkConfig := []string{
		// Enable IP forwarding for better packet routing
		"echo 'net.ipv4.ip_forward = 1' >> /etc/sysctl.conf",
		"echo 'net.ipv4.conf.all.accept_redirects = 0' >> /etc/sysctl.conf",

		// Optimize network buffer sizes for QUIC
		"echo 'net.core.rmem_max = 134217728' >> /etc/sysctl.conf",
		"echo 'net.core.wmem_max = 134217728' >> /etc/sysctl.conf",
		"echo 'net.core.rmem_default = 262144' >> /etc/sysctl.conf",
		"echo 'net.core.wmem_default = 262144' >> /etc/sysctl.conf",

		// Optimize for QUIC's UDP-based transport
		"echo 'net.ipv4.udp_mem = 102400 873800 16777216' >> /etc/sysctl.conf",
		"echo 'net.ipv4.udp_rmem_min = 8192' >> /etc/sysctl.conf",
		"echo 'net.ipv4.udp_wmem_min = 8192' >> /etc/sysctl.conf",

		// Apply sysctl settings immediately
		"sysctl -p",

		// Configure iptables rules for QUIC traffic (port 9000 and common QUIC ports)
		"iptables -A INPUT -p udp --dport 9000 -j ACCEPT",
		"iptables -A INPUT -p udp --dport 4800 -j ACCEPT", // Common QUIC port from logs
		"iptables -A INPUT -p udp --dport 3091 -j ACCEPT", // Another QUIC port from logs
		"iptables -A FORWARD -p udp --dport 9000 -j ACCEPT",
		"iptables -A FORWARD -p udp --dport 4800 -j ACCEPT",
		"iptables -A FORWARD -p udp --dport 3091 -j ACCEPT",

		// Disable network interface optimizations that can interfere with QUIC
		fmt.Sprintf("ethtool -K %s gro off || true", iface), // Disable generic receive offload
		fmt.Sprintf("ethtool -K %s tso off || true", iface), // Disable TCP segmentation offload
		fmt.Sprintf("ethtool -K %s gso off || true", iface), // Disable generic segmentation offload
		fmt.Sprintf("ethtool -K %s ufo off || true", iface), // Disable UDP fragmentation offload

		// Set network interface to use optimal settings for QUIC
		fmt.Sprintf("ethtool -G %s rx 1024 tx 1024 || true", iface), // Increase ring buffer sizes
	}

	for _, cmd := range networkConfig {
		// Add a small delay between commands to avoid overwhelming the VM agent
		time.Sleep(100 * time.Millisecond)

		_, err := RunCommandInInstance(c, name, []string{"sh", "-c", cmd})
		if err != nil {
			// Log warning but don't fail - some commands might not work in all VM environments
			fmt.Printf("Warning: failed to apply network config '%s' to VM %s: %v\n", cmd, name, err)
			return err
		}
	}

	return nil
}

// VerifyVMNetworkingForQUIC verifies that the QUIC networking optimizations were applied correctly
func VerifyVMNetworkingForQUIC(c incus.InstanceServer, name string) error {
	iface, err := findVMNetworkMainInterface(c, name)
	if err != nil {
		return err
	}

	verificationCommands := []string{
		"sysctl net.ipv4.ip_forward",
		"sysctl net.netfilter.nf_conntrack_udp_timeout",
		"sysctl net.core.rmem_max",
		"iptables -L INPUT | grep 9000",
		fmt.Sprintf("ethtool -k %s | grep -E '(gro|tso|gso|ufo)'", iface),
	}

	for _, cmd := range verificationCommands {
		// Add a small delay between verification commands
		time.Sleep(50 * time.Millisecond)

		_, err := RunCommandInInstance(c, name, []string{"sh", "-c", cmd})
		if err != nil {
			fmt.Printf("Warning: failed to verify config '%s' in VM %s: %v\n", cmd, name, err)
			return err
		}
	}

	return nil
}

func RunCommandInInstance(c incus.InstanceServer, name string, command []string) (string, error) {
	execReq := api.InstanceExecPost{
		Command:     command,
		WaitForWS:   true,
		Interactive: false,
	}

	var stdoutBuf, stderrBuf bytes.Buffer

	execArgs := incus.InstanceExecArgs{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
	}

	op, err := c.ExecInstance(name, execReq, &execArgs)
	if err != nil {
		return "", fmt.Errorf("failed to execute command: %w", err)
	}

	if err := op.Wait(); err != nil {
		return "", fmt.Errorf("operation wait failed: %w", err)
	}

	opAPI := op.Get()
	if code, ok := opAPI.Metadata["return"].(float64); ok && code != 0 {
		return stdoutBuf.String(), fmt.Errorf("command exited with error: %s", stderrBuf.String())
	}

	return stdoutBuf.String(), nil
}

func RunBackgroundCommandInInstance(c incus.InstanceServer, name string, command []string) error {
	execReq := api.InstanceExecPost{
		Command:     command,
		WaitForWS:   true,
		Interactive: false,
	}

	var stdoutBuf, stderrBuf bytes.Buffer

	execArgs := incus.InstanceExecArgs{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
	}

	op, err := c.ExecInstance(name, execReq, &execArgs)
	if err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	opAPI := op.Get()
	if code, ok := opAPI.Metadata["return"].(float64); ok && code != 0 {
		return fmt.Errorf("command exited with code %d", int(code))
	}

	return nil
}

func UploadFileToInstance(c incus.InstanceServer, name, localPath, remotePath string, mode int) error {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	reader := bytes.NewReader(data)

	args := incus.InstanceFileArgs{
		Content:   reader,
		Mode:      mode,
		Type:      "file",
		WriteMode: "overwrite",
	}

	if err := c.CreateInstanceFile(name, remotePath, args); err != nil {
		return fmt.Errorf("failed to upload file to instance: %w", err)
	}

	return nil
}

func DeleteInstance(c incus.InstanceServer, name string) error {
	instance, _, err := c.GetInstance(name)
	if err != nil {
		return fmt.Errorf("failed to get instance %s state: %w", name, err)
	}

	if instance.Status != "Stopped" {
		stopOp, err := c.UpdateInstanceState(name, api.InstanceStatePut{
			Action:  "stop",
			Force:   true,
			Timeout: 10,
		}, "")
		if err == nil {
			if err := stopOp.Wait(); err != nil {
				return fmt.Errorf("failed to wait for instance stop: %w", err)
			}
		}
		// ephemeral instances are deleted automatically after stopped
		if instance.Ephemeral {
			return nil
		}
	}

	delOp, err := c.DeleteInstance(name)
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	if err := delOp.Wait(); err != nil {
		return fmt.Errorf("failed to wait for instance deletion: %w", err)
	}

	return nil
}

func ListInstances(c incus.InstanceServer) ([]api.Instance, error) {
	return c.GetInstances(api.InstanceTypeAny)
}

// ConnectToClients return all Incus clients from configuration
func ConnectToClients(config *config.Config) ([]incus.InstanceServer, error) {
	clients := []incus.InstanceServer{}
	for _, host := range config.IncusHosts {
		target := LocalTarget
		if host.Host != LocalTarget {
			target = fmt.Sprintf("https://%s:%d", host.Host, host.Port)
		}
		client, err := ConnectIncus(target, host.ClientCert, host.ClientKey, host.ServerCert)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to host %s: %w", host.Host, err)
		}
		clients = append(clients, client)
	}
	return clients, nil
}

// CreateNodes creates `howMany` instances on a given Incus server (unix or remote URL).
func CreateNodes(clients []incus.InstanceServer, howMany int, namePrefix string) ([]*Node, error) {
	nodes := make([]*Node, 0, howMany)
	g := new(errgroup.Group)

	for i := range howMany {
		idx := i
		g.Go(func() error {
			client := clients[idx%len(clients)]
			name := namePrefix + "-node-" + strconv.Itoa(idx)

			err := CreateInstance(client, getInstanceType(), name)
			if err != nil {
				return fmt.Errorf("failed to create instance %s: %w", name, err)
			}

			nodes = append(nodes, &Node{
				Name:     name,
				Client:   client,
				Contexts: make(map[string]*Context),
			})
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		for i, node := range nodes {
			if node != nil {
				client := clients[i%len(clients)]
				_ = DeleteInstance(client, node.Name)
			}
		}
		return nil, err
	}

	return nodes, nil
}

// GetNode gets an instance already created on a given Incus server (unix or remote URL).
func GetNode(clients []incus.InstanceServer, name string) (*Node, error) {
	for _, client := range clients {
		_, _, err := client.GetInstance(name)
		if err == nil {
			node := &Node{
				Name:     name,
				Client:   client,
				Contexts: make(map[string]*Context),
			}
			return node, nil
		}
	}
	return nil, fmt.Errorf("failed to find instance %s on any provided Incus server", name)
}
