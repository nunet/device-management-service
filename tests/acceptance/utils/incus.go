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
