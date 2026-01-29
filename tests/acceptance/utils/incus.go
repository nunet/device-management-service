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
	"io"
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

type NetFwdParams struct {
	HostIface    string
	HostIP       string
	HostPort     string
	InstanceIP   string
	InstancePort string
	Protocol     string
}

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
		// TODO Environment
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

func DownloadFile(c incus.InstanceServer, instance, instancePath, hostPath string) error {
	r, _, err := c.GetInstanceFile(instance, instancePath)
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read file content: %w", err)
	}

	if err := os.WriteFile(hostPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
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

// CreateInstances creates `howMany` instances on a given Incus server (unix or remote URL).
func CreateInstances(clients []incus.InstanceServer, howMany int, namePrefix string) ([]*Instance, error) {
	nodes := make([]*Instance, 0, howMany)
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

			nodes = append(nodes, &Instance{
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
func GetNode(clients []incus.InstanceServer, name string) (*Instance, error) {
	for _, client := range clients {
		_, _, err := client.GetInstance(name)
		if err == nil {
			node := &Instance{
				Name:     name,
				Client:   client,
				Contexts: make(map[string]*Context),
			}
			return node, nil
		}
	}
	return nil, fmt.Errorf("failed to find instance %s on any provided Incus server", name)
}

func getOrCreateNetworkForward(c incus.InstanceServer, networkName, listenAddress string) (*api.NetworkForward, error) {
	forward, _, err := c.GetNetworkForward(networkName, listenAddress)
	if forward != nil && err == nil {
		// Forward already exists
		return forward, nil
	}

	forwardPost := api.NetworkForwardsPost{
		ListenAddress: listenAddress,
	}

	err = c.CreateNetworkForward(networkName, forwardPost)
	if err != nil {
		return nil, fmt.Errorf("failed to create network forward on %s: %w", networkName, err)
	}

	forward, _, err = c.GetNetworkForward(networkName, listenAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get network forward on %s after creation: %w", networkName, err)
	}

	return forward, nil
}

func NetworkForwardPort(c incus.InstanceServer, params NetFwdParams) error {
	// Update the forward to add target address and port
	forward, err := getOrCreateNetworkForward(c, params.HostIface, params.HostIP)
	if err != nil {
		return fmt.Errorf("failed to get network forward on %s: %w", params.HostIface, err)
	}

	if forward.Ports == nil {
		forward.Ports = []api.NetworkForwardPort{}
	}

	forward.Ports = append(forward.Ports,
		api.NetworkForwardPort{
			ListenPort:    params.HostPort,
			TargetAddress: params.InstanceIP,
			TargetPort:    params.InstancePort,
			Protocol:      params.Protocol,
			Description:   "NuNet AccTest: Forward libp2p nat listening port to instance addr",
		},
	)

	netForwardPut := api.NetworkForwardPut{
		Ports: forward.Ports,
	}

	return c.UpdateNetworkForward(params.HostIface, params.HostIP, netForwardPut, "")
}

func CleanNetworkForward(c incus.InstanceServer) error {
	nets, err := c.GetNetworks()
	if err != nil {
		return fmt.Errorf("failed to get networks: %w", err)
	}
	for _, net := range nets {
		forwards, err := c.GetNetworkForwards(net.Name)
		if err != nil {
			// network can not have a forward
			continue
		}
		var keepFwds []api.NetworkForwardPort
		for _, fwd := range forwards {
			for _, port := range fwd.Ports {
				if !strings.Contains(port.Description, "NuNet AccTest") {
					keepFwds = append(keepFwds, port)
				}
			}
			if len(keepFwds) > 0 {
				// Update forward to keep only non-AccTest ports
				netForwardPut := api.NetworkForwardPut{
					Ports: keepFwds,
				}
				err := c.UpdateNetworkForward(net.Name, fwd.ListenAddress, netForwardPut, "")
				if err != nil {
					return fmt.Errorf("failed to update network forward on %s: %w", net.Name, err)
				}
			} else {
				// Delete the forward entirely
				err := c.DeleteNetworkForward(net.Name, fwd.ListenAddress)
				if err != nil {
					return fmt.Errorf("failed to delete network forward on %s: %w", net.Name, err)
				}
			}
		}
	}
	return nil
}

// GetInstanceType returns the instance type from environment
// Exported version for use in other packages
func GetInstanceType() string {
	return getInstanceType()
}
