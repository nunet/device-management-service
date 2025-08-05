package utils

import (
	"bytes"
	"fmt"
	"os"
	"strconv"

	incus "github.com/lxc/incus/client"
	"github.com/lxc/incus/shared/api"
	"gitlab.com/nunet/device-management-service/tests/acceptance/config"
	"golang.org/x/sync/errgroup"
)

const (
	DefaultImage = "ubuntu/22.04"
	LocalTarget  = "local"
)

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

func CreateInstance(c incus.InstanceServer, name, image string) error {
	req := api.InstancesPost{
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
			Type:     "image",
			Alias:    image,
			Server:   "https://images.linuxcontainers.org",
			Protocol: "simplestreams",
		},
	}

	op, err := c.CreateInstance(req)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	if err := op.Wait(); err != nil {
		return fmt.Errorf("failed to wait for container creation: %w", err)
	}

	startReq := api.InstanceStatePut{
		Action:   "start",
		Timeout:  -1,
		Force:    true,
		Stateful: false,
	}

	startOp, err := c.UpdateInstanceState(name, startReq, "")
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
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

func UploadFileToInstance(c incus.InstanceServer, containerName, localPath, remotePath string, mode int) error {
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

	if err := c.CreateInstanceFile(containerName, remotePath, args); err != nil {
		return fmt.Errorf("failed to upload file to container: %w", err)
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
				return fmt.Errorf("failed to wait for container stop: %w", err)
			}
		}
		// ephemeral instances are deleted automatically after stopped
		if instance.Ephemeral {
			return nil
		}
	}

	delOp, err := c.DeleteInstance(name)
	if err != nil {
		return fmt.Errorf("failed to delete container: %w", err)
	}

	if err := delOp.Wait(); err != nil {
		return fmt.Errorf("failed to wait for container deletion: %w", err)
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
func CreateNodes(clients []incus.InstanceServer, howMany int, image, containerNamePrefix string) ([]*Node, error) {
	nodes := make([]*Node, 0, howMany)
	g := new(errgroup.Group)

	for i := range howMany {
		idx := i
		g.Go(func() error {
			client := clients[idx%len(clients)]
			name := containerNamePrefix + "-node-" + strconv.Itoa(idx)

			err := CreateInstance(client, name, image)
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
