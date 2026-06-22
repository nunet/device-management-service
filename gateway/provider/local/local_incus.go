package local

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	logging "github.com/ipfs/go-log/v2"
	incus "github.com/lxc/incus/client"
	"github.com/lxc/incus/shared/api"
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/gateway/provider"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/types"
)

var log = logging.Logger("gateway/local/incus")

// IncusProvider
type IncusProvider struct {
	client        incus.InstanceServer
	dmsBinaryPath string
	gatewayDID    string
}

func RegisterFactory(reg *provider.FactoryRegistry) {
	reg.Register("local-incus", func(_ map[string]interface{}) (provider.Provider, error) {
		// pass cfg if needed
		return NewLocalIncusProvider(reg.GatewayDID)
	})
}

// NewLocalIncusProvider creates a new local Incus provider using the local Unix socket.
func NewLocalIncusProvider(gatewayDID string) (*IncusProvider, error) {
	c, err := incus.ConnectIncusUnix("", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to local Incus: %w", err)
	}

	return &IncusProvider{client: c, dmsBinaryPath: os.Getenv("DMS_BINARY_PATH"), gatewayDID: gatewayDID}, nil
}

// Name returns the provider identifier.
func (p *IncusProvider) Name() string {
	return "local-incus"
}

// ListPlans returns a few static plans that represent local resource profiles.
func (p *IncusProvider) ListPlans(_ context.Context) ([]provider.Plan, error) {
	plans := make([]provider.Plan, 0, 1)
	plans = append(plans, provider.Plan{
		ID:          "plan1",
		Name:        "VM1",
		Description: "vm with 8 gb ram and 4 cpu",
		CPU:         4,
		MemoryMB:    4,
		DiskGB:      10,
		Region:      "local",
		PriceUSD:    10,
	})

	return plans, nil
}

//nolint:unparam
func runCommand(ctx context.Context, name string, args ...string) (string, error) {
	log.Infof("runCommand : %s %s", name, strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, name, args...)

	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Errorf("runCommand started: error: %v %s", err, stderr.String())

		return "", fmt.Errorf("command %q failed: %w; stderr: %s", name, err, stderr.String())
	}

	return out.String(), nil
}

// ProvisionServer creates a new Incus instance (container or VM) based on the plan and image.
// TODO: proper capabilities for orchestrator instead of root anchoring
func (p *IncusProvider) ProvisionServer(ctx context.Context, _ provider.Plan, name string, imageAlias, orchestratorDID string) (*provider.Server, error) {
	// for incus if there was no image just default to the following image
	if imageAlias == "" {
		imageAlias = "ubuntu-22.04-vm"
	}

	res, err := runCommand(ctx, "incus", "launch", imageAlias, name, "--vm",
		"--config", "limits.cpu=4",
		"--config", "limits.memory=4GiB",
		"--device", "root,size=5GiB")
	if err != nil {
		return nil, fmt.Errorf("failed to launch VM: %w; stderr: %s", err, res)
	}

	var ip string
	timeout := time.After(2 * time.Minute)
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for VM %s to get an IP", name)
		case <-tick.C:
			out, err := exec.CommandContext(ctx, "incus", "list", name, "--format", "json").Output()
			if err != nil {
				continue
			}

			var vmList []struct {
				State struct {
					Network map[string]struct {
						Addresses []struct {
							Family  string `json:"family"`
							Address string `json:"address"`
						} `json:"addresses"`
					} `json:"network"`
				} `json:"state"`
			}

			if err := json.Unmarshal(out, &vmList); err != nil || len(vmList) == 0 {
				continue
			}

			found := false
			for _, iface := range vmList[0].State.Network {
				for _, addr := range iface.Addresses {
					if addr.Family == "inet" && addr.Address != "127.0.0.1" {
						ip = addr.Address
						found = true
						break
					}
				}
				if found {
					break
				}
			}

			if ip != "" {
				goto done
			}
		}
	}
done:
	server := &provider.Server{
		ID:     name,
		Name:   name,
		Status: "RUNNING",
		IP:     ip,
	}

	time.Sleep(5 * time.Second)

	res, err = runCommand(ctx, "incus", "file", "push", p.dmsBinaryPath, name+"/home/ubuntu/dms")
	if err != nil {
		return nil, fmt.Errorf("failed to copy file into VM: %w %s", err, res)
	}

	res, err = runCommand(ctx, "incus", "exec", name, "--", "bash", "-c", `
  set -eux
  # Disable systemd-resolved and set custom DNS
  #systemctl stop systemd-resolved
  #systemctl disable systemd-resolved
  rm -f /etc/resolv.conf
  echo -e "nameserver 1.1.1.1\nnameserver 8.8.8.8" | tee /etc/resolv.conf
  apt install -y ca-certificates curl
  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
  chmod a+r /etc/apt/keyrings/docker.asc
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "${UBUNTU_CODENAME:-$VERSION_CODENAME}") stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
  apt update
  apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
  apt install -y openssh-server
  systemctl enable ssh
  systemctl start ssh
  # Allow password authentication in sshd_config
  sed -i 's/^#\?PasswordAuthentication .*/PasswordAuthentication yes/' /etc/ssh/sshd_config
  sed -i 's/^#\?PermitRootLogin .*/PermitRootLogin yes/' /etc/ssh/sshd_config
  systemctl restart ssh
  # Set a root password
  echo "root:root" | chpasswd
  DMS_PASSPHRASE=pass /home/ubuntu/dms key new dms
  setcap cap_net_admin,cap_sys_admin+ep /home/ubuntu/dms
  DMS_PASSPHRASE=pass /home/ubuntu/dms cap new dms
  DMS_PASSPHRASE=pass /home/ubuntu/dms cap anchor --context dms --root `+p.gatewayDID+`
  DMS_PASSPHRASE=pass /home/ubuntu/dms cap anchor --context dms --root `+orchestratorDID+`
  /home/ubuntu/dms config set p2p.listen_address '["/ip4/0.0.0.0/tcp/9001", "/ip4/0.0.0.0/udp/9001/quic-v1"]'
  GOLOG_LOG_LEVEL=debug,pubsub=error,observability=error DMS_PASSPHRASE=pass /home/ubuntu/dms run --context dms > logfile.log 2>&1 &
  sleep 7
  DMS_PASSPHRASE=pass /home/ubuntu/dms actor cmd --context dms /dms/node/onboarding/onboard --no-gpu --ram 3 GB --cpu 2 --disk 2GiB
`)
	if err != nil {
		return nil, fmt.Errorf("failed to install dms and requirements: %w %s", err, res)
	}

	time.Sleep(10 * time.Second)

	// connect to gateway, orchestrator and bootstrap peers to give identify a head start before the deployment
	gatewayHandle, err := actor.HandleFromDID(p.gatewayDID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse gateway DID handle: %w", err)
	}
	orchHandle, err := actor.HandleFromDID(orchestratorDID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse orchestrator DID handle: %w", err)
	}

	// Connect to gateway
	res, err = runCommand(ctx, "incus", "exec", name, "--", "bash", "-c", `
	  set -eux
	  DMS_PASSPHRASE=pass /home/ubuntu/dms actor cmd --context dms /dms/node/peers/connect --address /p2p/`+gatewayHandle.Address.HostID+`
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to execute self with json output: %w %s", err, res)
	}

	// connect to orchestrator
	res, err = runCommand(ctx, "incus", "exec", name, "--", "bash", "-c", `
	  set -eux
	  DMS_PASSPHRASE=pass /home/ubuntu/dms actor cmd --context dms /dms/node/peers/connect --address /p2p/`+orchHandle.Address.HostID+`
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to execute self with json output: %w %s", err, res)
	}

	// connect to bootstrap nodes
	for _, addr := range config.DefaultConfig.BootstrapPeers {
		res, err = runCommand(ctx, "incus", "exec", name, "--", "bash", "-c", `
	  set -eux
	  DMS_PASSPHRASE=pass /home/ubuntu/dms actor cmd --context dms /dms/node/peers/connect --address `+addr[31:])
		if err != nil {
			return nil, fmt.Errorf("failed to execute self with json output: %w %s", err, res)
		}
	}

	// give identify some time to finish obtaining observed addr
	time.Sleep(60 * time.Second)

	res, err = runCommand(ctx, "incus", "exec", name, "--", "bash", "-c", `
	  set -eux
	  DMS_PASSPHRASE=pass /home/ubuntu/dms actor cmd --context dms /dms/node/peers/self
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to execute self : %w %s", err, res)
	}

	var self self
	err = json.Unmarshal([]byte(res), &self)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal self payload: %w %s", err, res)
	}

	server.PeerID = self.ID

	ips := strings.Split(self.ListenAddr, ",")
	if len(ips) == 0 {
		return nil, fmt.Errorf("failed to get listen addr: %w %s", err, res)
	}
	server.ListenAddr = ips[0]

	return server, nil
}

type self struct {
	ID         string `json:"id"`
	ListenAddr string `json:"listen_addr"`
}

// DeleteServer removes an Incus instance by name.
func (p *IncusProvider) DeleteServer(_ context.Context, serverID string) error {
	op, err := p.client.DeleteInstance(serverID)
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}
	return op.Wait()
}

// RestartServer restarts an Incus instance.
func (p *IncusProvider) RestartServer(_ context.Context, serverID string) error {
	op, err := p.client.UpdateInstanceState(serverID, api.InstanceStatePut{
		Action:  "restart",
		Timeout: -1,
	}, "")
	if err != nil {
		return fmt.Errorf("failed to restart instance: %w", err)
	}
	return op.Wait()
}

// GetServerStatus retrieves instance state and metadata.
func (p *IncusProvider) GetServerStatus(_ context.Context, serverID string) (*provider.Server, error) {
	inst, etag, err := p.client.GetInstance(serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	state, _, err := p.client.GetInstanceState(serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance state: %w", err)
	}

	server := &provider.Server{
		ID:     inst.Name,
		Name:   inst.Name,
		Status: state.Status,

		Metadata: map[string]string{
			"etag": etag,
			"pid":  fmt.Sprintf("%d", state.Pid),
		},
	}
	return server, nil
}

// SelectMatchingPlan selects a plan matching target resource requirements.
func (p *IncusProvider) SelectMatchingPlan(_ []provider.Plan, _ types.Resources) (*provider.Plan, error) {
	plns, _ := p.ListPlans(context.Background())
	return &plns[0], nil
}
