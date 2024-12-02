// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package actor

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/dms/jobs"
	"gitlab.com/nunet/device-management-service/dms/jobs/parser"
	job_types "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/types"
)

var ErrInvalidArgument = errors.New("invalid argument")

type Command = cobra.Command

type Payload struct {
	val any
}

type behaviorConfig struct {
	Behavior    string
	Type        string
	Topic       string
	Payload     func() any
	PayloadEnc  func(payload any) (any, error)
	SetFlags    func(cmd *Command, payload any)
	PreRunE     func(cmd *Command, payload any) error
	ValidArgsFn func(cmd *Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)
	Args        cobra.PositionalArgs
	Long        string
	Short       string
}

type NewDeploymentRequestCmd struct {
	Config string
}

var behaviors = map[string]behaviorConfig{
	// /public/hello
	node.PublicHelloBehavior: {
		Type:  bInvoke,
		Short: "Broadcast a 'hello' message",
		Long: `Invoke the /public/hello behavior on an actor

This behavior broadcasts a "hello" for a polite introduction.

Examples:

  nunet actor cmd --context user /public/hello
  nunet actor cmd --context user /public/hello --dest <did/peer_id/actor_handle>`,
	},
	// /broadcast/hello
	node.BroadcastHelloBehavior: {
		Type: bBroadcast,

		Topic: node.BroadcastHelloTopic,
		Short: "Broadcast a 'hello' message to a topic",
		Long: `Invokes the /broadcast/hello behavior on an actor

This behavior sends a "hello" message to a broadcast topic for polite introduction.

Examples:

  nunet actor cmd --context user /broadcast/hello`,
	},
	// /public/status
	node.PublicStatusBehavior: {
		Type:  bInvoke,
		Short: "Retrieve actor status",
		Long: `Invokes the /public/status behavior on an actor

This behavior retrieves the status and resources information.

Examples:
  nunet actor cmd --context user /public/status # own actor status
  nunet actor cmd --context user /public/status --dest <did/peer_id/actor_handle> # status of specified destination`,
	},
	// /dms/node/peers/list
	node.PeersListBehavior: {
		Type:  bInvoke,
		Short: "List connected peers",
		Long: `Invokes the /dms/node/peers/list behavior on an actor

This behavior retrieves a list of connected peers.

Examples:
  nunet actor cmd --context user /dms/node/peers/list # own node actor peer list
  nunet actor cmd --context user /dms/node/peers/list --dest <did/peer_id/actor_handle> # specified node actor peer list`,
	},
	// /dms/node/peers/self
	node.PeerAddrInfoBehavior: {
		Type:  bInvoke,
		Short: "Get peer's ID and addresses",
		Long: `Invokes the /dms/node/peers/self behavior on an actor

This behavior retrieves information about the node itself, such as its ID or addresses.

Examples:
  nunet actor cmd --context user /dms/node/peers/self # own node actor peer ID
  nunet actor cmd --context user /dms/node/peers/self --dest <did/peer_id/actor_handle> # specified node actor peer ID`,
	},
	// /dms/node/peers/ping
	node.PeerPingBehavior: {
		Type:    bInvoke,
		Payload: func() any { return &node.PingRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.PingRequest)

			cmd.Flags().StringVarP(&p.Host, "host", "H", "", "host address to ping (required)")
			_ = cmd.MarkFlagRequired("host")
		},
		PayloadEnc: func(payload any) (any, error) {
			req, ok := payload.(*node.PingRequest)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}
			return req, nil
		},
		Short: "Ping a peer",
		Long: `Invokes the /dms/node/peers/ping behavior on an actor

This behavior establishes a ping connection with a peer.

Examples:
  nunet actor cmd --context user /dms/node/peers/ping --host <peer_id>`,
	},
	// /dms/node/peers/dht
	node.PeerDHTBehavior: {
		Type:  bInvoke,
		Short: "List peers connected to DHT",
		Long: `Invokes the /dms/node/peers/dht behavior on an actor

This behavior returns a list of peers from the  Distributed Hash Table (DHT) used for peer discovery and content routing.

Examples:
  nunet actor cmd --context user /dms/node/peers/dht`,
	},
	// /dms/node/peers/connect
	node.PeerConnectBehavior: {
		Type:    bInvoke,
		Payload: func() any { return &node.PeerConnectRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.PeerConnectRequest)

			cmd.Flags().StringVarP(&p.Address, "address", "a", "", "peer address to connect to (required)")
			_ = cmd.MarkFlagRequired("address")
		},
		PayloadEnc: func(payload any) (any, error) {
			req, ok := payload.(*node.PeerConnectRequest)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}
			return req, nil
		},
		Short: "Connect to a peer",
		Long: `Invokes the /dms/node/peers/connect behavior on an actor

This behavior initiates a connection to a specified peer.

Examples:
  nunet actor cmd --context user /dms/node/peers/connect --address /p2p/<peer_id>`,
	},
	// /dms/node/peers/score
	node.PeerScoreBehavior: {
		Type:  bInvoke,
		Short: "Retrieves gossipsub broadcast score",
		Long: `Invokes the /dms/node/peers/score behavior on an actor

This behavior retrieves a snapshot of the peer's gossipsub broadcast score.

Examples:
  nunet actor cmd --context user /dms/node/peers/score`,
	},
	// /dms/node/onboarding/onboard
	node.OnboardBehavior: {
		Type:    bInvoke,
		Payload: func() any { return &node.OnboardRequest{} },
		SetFlags: func(cmd *Command, payload any) {
			// infer the type of the payload
			p := payload.(*node.OnboardRequest)
			cmd.Flags().Float64VarP(&p.Config.OnboardedResources.RAM.Size, "ram", "R", 0, "set the amount of memory in GB to reserve for NuNet")
			cmd.Flags().Float32VarP(&p.Config.OnboardedResources.CPU.Cores, "cpu", "C", 0, "set the number of CPU cores to reserve for NuNet")
			cmd.Flags().Float64VarP(&p.Config.OnboardedResources.Disk.Size, "disk", "D", 0, "set the amount of disk size in GB to reserve for NuNet")
			cmd.Flags().StringVarP(&p.GPUs, "gpus", "G", "", "comma-separated list of GPU Index and VRAM in GB (e.g. 0:4,1:8). The gpu index can be obtained from 'nunet gpu list' command")
			cmd.Flags().BoolVarP(&p.NoGPU, "no-gpu", "N", false, "do not reserve any GPU resources")
			cmd.MarkFlagsOneRequired("ram", "cpu", "disk")
			cmd.MarkFlagsRequiredTogether("ram", "cpu", "disk")
		},
		PreRunE: onboardBehaviorPreRun,
		PayloadEnc: func(payload any) (any, error) {
			req, ok := payload.(*node.OnboardRequest)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}

			// convert RAM and Disk from GB to bytes
			req.Config.OnboardedResources.RAM.Size = types.ConvertGBToBytes(req.Config.OnboardedResources.RAM.Size)
			req.Config.OnboardedResources.Disk.Size = types.ConvertGBToBytes(req.Config.OnboardedResources.Disk.Size)
			return req, nil
		},
		Short: "Onboard a node to the network",
		Long: `Invokes the /dms/node/onboarding/onboard behavior on an actor

This behavior is used to onboard a node to the DMS, making its resources available for use.

Examples:
  nunet actor cmd --context user /dms/node/onboarding/onboard --memory 1 --cpu 2`,
	},
	// /dms/node/onboarding/offboard
	node.OffboardBehavior: {
		Type:     bInvoke,
		Payload:  func() any { return &node.OffboardRequest{} },
		SetFlags: func(_ *cobra.Command, _ any) {},
		PayloadEnc: func(payload any) (any, error) {
			req, ok := payload.(*node.OffboardRequest)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}
			return req, nil
		},
		Short: "Offboard a node from the network",
		Long: `Invokes the /dms/node/onboarding/offboard behavior on an actor

This behavior is used to offboard a node from the DMS (Device Management Service).

Examples:
  nunet actor cmd --context user /dms/node/onboarding/offboard
  nunet actor cmd --context user /dms/node/onboarding/offboard --force`,
	},
	// /dms/node/onboarding/status
	node.OnboardStatusBehavior: {
		Type:  bInvoke,
		Short: "Retrieve onboarding status of a node",
		Long: `Invokes the /dms/node/onboarding/status behavior on an actor

This behavior is used to check the onboarding status of a node.

Examples:
  nunet actor cmd --context user /dms/node/onboarding/status`,
	},

	// /dms/node/deployment/list
	node.DeploymentListBehavior: {
		Type:  bInvoke,
		Short: "List deployments",
		Long: `Invokes the /dms/node/deployment/list behavior on an actor

This behavior retrieves a list of all deployments on the node.

Examples:
  nunet actor cmd --context user /dms/node/deployment/list`,
	},

	// /dms/node/deployment/status
	node.DeploymentStatusBehavior: {
		Type:    bInvoke,
		Payload: func() any { return &node.DeploymentStatusRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.DeploymentStatusRequest)
			cmd.Flags().StringVarP(&p.ID, "id", "i", "", "deployment ID (required)")
			_ = cmd.MarkFlagRequired("id")
		},
		PayloadEnc: func(payload any) (any, error) {
			req, ok := payload.(*node.DeploymentStatusRequest)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}
			return req, nil
		},
		Short: "Get deployment status",
		Long: `Invokes the /dms/node/deployment/status behavior on an actor

This behavior retrieves the status of a specific deployment.

Examples:
  nunet actor cmd --context user /dms/node/deployment/status --id <deployment_id>`,
	},

	// /dms/node/deployment/logs
	node.DeploymentLogsBehavior: {
		Type:    bInvoke,
		Payload: func() any { return &node.DeploymentLogsRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.DeploymentLogsRequest)
			cmd.Flags().StringVarP(&p.EnsembleID, "id", "i", "", "ensemble ID (required)")
			cmd.Flags().StringVarP(&p.AllocationName, "allocation", "a", "", "allocation name (required)")
			_ = cmd.MarkFlagRequired("id")
			_ = cmd.MarkFlagRequired("allocation")
		},
		PayloadEnc: func(payload any) (any, error) {
			req, ok := payload.(*node.DeploymentLogsRequest)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}
			return req, nil
		},
		Short: "Get deployment logs",
		Long: `Invokes the /dms/node/deployment/logs behavior on an actor

This behavior retrieves the logs of a specific deployment, writing it to a file
with path returned in the response.

Examples:
  nunet actor cmd --context user /dms/node/deployment/logs --id <deployment_id> --allocation <allocation_name>`,
	},

	// /dms/node/deployment/manifest
	node.DeploymentManifestBehavior: {
		Type:    bInvoke,
		Payload: func() any { return &node.DeploymentManifestRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.DeploymentManifestRequest)
			cmd.Flags().StringVarP(&p.ID, "id", "i", "", "deployment ID (required)")
			_ = cmd.MarkFlagRequired("id")
		},
		PayloadEnc: func(payload any) (any, error) {
			req, ok := payload.(*node.DeploymentManifestRequest)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}
			return req, nil
		},
		Short: "Get deployment manifest",
		Long: `Invokes the /dms/node/deployment/manifest behavior on an actor

This behavior retrieves the manifest of a specific deployment.

Examples:
  nunet actor cmd --context user /dms/node/deployment/manifest --id <deployment_id>`,
	},

	// /dms/node/deployment/shutdown
	node.DeploymentShutdownBehavior: {
		Type:    bInvoke,
		Payload: func() any { return &node.DeploymentShutdownRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.DeploymentShutdownRequest)
			cmd.Flags().StringVarP(&p.ID, "id", "i", "", "deployment ID (required)")
			_ = cmd.MarkFlagRequired("id")
		},
		PayloadEnc: func(payload any) (any, error) {
			req, ok := payload.(*node.DeploymentShutdownRequest)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}
			return req, nil
		},
		Short: "Shutdown a deployment",
		Long: `Invokes the /dms/node/deployment/shutdown behavior on an actor

This behavior shuts down a specific deployment.

Examples:
  nunet actor cmd --context user /dms/node/deployment/shutdown --id <deployment_id>`,
	},

	// /dms/node/vm/start/custom
	node.VMStartBehavior: {
		Type:    bInvoke,
		Payload: func() any { return &vmStartOpts{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*vmStartOpts)
			cmd.Flags().StringVarP(&p.Engine.KernelImage, "kernel", "k", "", "path to kernel image file (required)")
			cmd.Flags().StringVarP(&p.Engine.RootFileSystem, "rootfs", "r", "", "path to root fs image file (required)")
			cmd.Flags().StringVarP(&p.Engine.Initrd, "initrd", "i", "", "path to initial ram disk")
			cmd.Flags().StringVarP(&p.Engine.KernelArgs, "args", "a", "", "arguments to pas to the kernel")
			cmd.Flags().Float32Var(&p.Resources.CPU.Cores, "cpu", 1, "CPU cores to allocate")
			cmd.Flags().Float64VarP(&p.Resources.RAM.Size, "ram", "m", 1, "Memory to allocate in GB")
			cmd.Flags().Float64Var(&p.Resources.Disk.Size, "disk", 0.5, "path to disk image file")
			_ = cmd.MarkFlagRequired("kernel")
			_ = cmd.MarkFlagFilename("kernel")
			_ = cmd.MarkFlagRequired("rootfs")
			_ = cmd.MarkFlagFilename("rootfs")
		},
		PayloadEnc: func(payload any) (any, error) {
			opts, ok := payload.(*vmStartOpts)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}

			return newCustomVMStartRequest(opts)
		},
		Short: "Starts a custom VM",
		Long: `Invokes the /dms/node/vm/start/custom behavior on an actor

This behavior starts a new VM with custom configurations.

Examples:
  nunet actor cmd --context user /dms/node/vm/start/custom --kernel /path/to/kernel --rootfs /path/to/rootfs --cpu 2 --memory 2048`,
	},
	// /dms/node/vm/stop
	node.VMStopBehavior: {
		Payload: func() any { return &node.VMStopRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.VMStopRequest)
			p.ExecutionType = job_types.ExecutorFirecracker
			cmd.Flags().StringVarP(&p.ExecutionID, "id", "i", "", "execution ID of the VM (required)")
			_ = cmd.MarkFlagRequired("id")
		},
		PayloadEnc: func(payload any) (any, error) {
			req, ok := payload.(*node.VMStopRequest)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}
			return req, nil
		},
		Type:  bInvoke,
		Short: "Stops a running VM",
		Long: `Invokes the /dms/node/vm/stop behavior on an actor

This behavior stops a running VM.

Examples:
  nunet actor cmd --context user /dms/node/vm/stop --id <execution_id>`,
	},
	// /dms/node/vm/list
	node.VMListBehavior: {
		Payload: func() any {
			return &node.ListVMResponse{
				ExecutionType: job_types.ExecutorFirecracker,
			}
		},
		Type:  bInvoke,
		Short: "List running VMs",
		Long: `Invokes the /dms/node/vm/list behavior on an actor

This behavior retrieves a list of virtual machines (VMs) running on the node.

Examples:
  nunet actor cmd --context user /dms/node/vm/list`,
	},
	node.NewDeploymentBehavior: {
		Type:  bInvoke,
		Short: "Create a new deployment",
		Long: `Invokes the /dms/node/deployment/new behavior on an actor

This behavior creates a new deployment.

Examples:
  nunet actor cmd --context user /dms/node/deployment/new --spec-file <path to ensemble specification file>`,
		Payload: func() any { return &NewDeploymentRequestCmd{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*NewDeploymentRequestCmd)
			cmd.Flags().StringVarP(&p.Config, "spec-file", "f", "ensemble.yaml", "path of the ensemble specification file (required)")
		},
		PayloadEnc: func(payload any) (any, error) {
			req, ok := payload.(*NewDeploymentRequestCmd)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}
			data, err := os.ReadFile(req.Config)
			if err != nil {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}

			cfg := &node.NewDeploymentRequest{}
			err = parser.Parse(parser.SpecTypeEnsembleV1, data, &cfg.Ensemble)
			if err != nil {
				return nil, err
			}

			for name, script := range cfg.Ensemble.V1.Scripts {
				fmt.Println(name, string(script))
				scriptData, err := os.ReadFile(string(script))
				if err != nil {
					return nil, fmt.Errorf("failed to read script file: %w", err)
				}
				cfg.Ensemble.V1.Scripts[name] = scriptData

			}

			for name, key := range cfg.Ensemble.V1.Keys {
				key, err := os.ReadFile(key)
				if err != nil {
					return nil, fmt.Errorf("failed to read script file: %w", err)
				}
				cfg.Ensemble.V1.Keys[name] = string(key)

			}

			return cfg, nil
		},
	},

	jobs.SubnetCreateBehavior: {
		Payload: func() any { return &jobs.SubnetCreateRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*jobs.SubnetCreateRequest)

			cmd.Flags().StringVarP(&p.SubnetID, "subnet-id", "s", "", "subnet ID (required)")
			cmd.Flags().StringToStringVarP(&p.RoutingTable, "routing-table", "r", nil, "subnet routing table (required)")
			_ = cmd.MarkFlagRequired("subnet-id")
		},
		PayloadEnc: func(payload any) (any, error) {
			req, ok := payload.(*jobs.SubnetCreateRequest)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}

			return req, nil
		},
		Type:  bInvoke,
		Short: "Create a subnet",
		Long: `Invokes the /dms/node/subnet/create behavior on an actor

This behavior creates a new subnet with the specified subnet ID, IP address, and routing table.

Examples:
  nunet actor cmd --context user /dms/node/subnet/create --subnet-id <subnet_id> --ip <ip> --routing-table <routing_table>`,
	},

	jobs.SubnetDestroyBehavior: {
		Payload: func() any { return &jobs.SubnetDestroyRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*jobs.SubnetDestroyRequest)

			cmd.Flags().StringVarP(&p.SubnetID, "subnet-id", "s", "", "subnet ID (required)")
			_ = cmd.MarkFlagRequired("subnet-id")
		},
		PayloadEnc: func(payload any) (any, error) {
			req, ok := payload.(*jobs.SubnetDestroyRequest)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}

			return req, nil
		},
		Type:  bInvoke,
		Short: "Destroy a subnet",
		Long: `Invokes the /dms/node/subnet/destroy behavior on an actor

This behavior destroys the specified subnet.

Examples:
  nunet actor cmd --context user /dms/node/subnet/destroy --subnet-id <subnet_id>`,
	},

	jobs.SubnetAddPeerBehavior: {
		Payload: func() any { return &jobs.SubnetAddPeerRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*jobs.SubnetAddPeerRequest)

			cmd.Flags().StringVarP(&p.SubnetID, "subnet-id", "s", "", "subnet ID (required)")
			cmd.Flags().StringVarP(&p.PeerID, "peer-id", "p", "", "peer ID (required)")
			cmd.Flags().StringVarP(&p.IP, "ip", "i", "", "peer IP address (required)")
			_ = cmd.MarkFlagRequired("subnet-id")
			_ = cmd.MarkFlagRequired("peer-id")
			_ = cmd.MarkFlagRequired("ip")
		},
		PayloadEnc: func(payload any) (any, error) {
			req, ok := payload.(*jobs.SubnetAddPeerRequest)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}

			return req, nil
		},
		Type:  bInvoke,
		Short: "Add a peer to a subnet",
		Long: `Invokes the /dms/node/subnet/add-peer behavior on an actor

This behavior adds a peer to the specified subnet.

Examples:
  nunet actor cmd --context user /dms/node/subnet/add-peer --subnet-id <subnet_id> --peer-id <peer_id> --ip <ip>`,
	},

	jobs.SubnetRemovePeerBehavior: {
		Payload: func() any { return &jobs.SubnetRemovePeerRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*jobs.SubnetRemovePeerRequest)

			cmd.Flags().StringVarP(&p.SubnetID, "subnet-id", "s", "", "subnet ID (required)")
			cmd.Flags().StringVarP(&p.PeerID, "peer-id", "p", "", "peer ID (required)")
			_ = cmd.MarkFlagRequired("subnet-id")
			_ = cmd.MarkFlagRequired("peer-id")
		},
		PayloadEnc: func(payload any) (any, error) {
			req, ok := payload.(*jobs.SubnetRemovePeerRequest)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}

			return req, nil
		},
		Type:  bInvoke,
		Short: "Remove a peer from a subnet",
		Long: `Invokes the /dms/node/subnet/remove-peer behavior on an actor

This behavior removes a peer from the specified subnet.

Examples:
  nunet actor cmd --context user /dms/node/subnet/remove-peer --subnet-id <subnet_id> --peer-id <peer_id>`,
	},

	jobs.SubnetAcceptPeerBehavior: {
		Payload: func() any { return &jobs.SubnetAcceptPeerRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*jobs.SubnetAcceptPeerRequest)

			cmd.Flags().StringVarP(&p.SubnetID, "subnet-id", "s", "", "subnet ID (required)")
			cmd.Flags().StringVarP(&p.PeerID, "peer-id", "p", "", "peer ID (required)")
			cmd.Flags().StringVarP(&p.IP, "ip", "i", "", "peer IP address (required)")
			_ = cmd.MarkFlagRequired("subnet-id")
			_ = cmd.MarkFlagRequired("peer-id")
			_ = cmd.MarkFlagRequired("ip")
		},
		PayloadEnc: func(payload any) (any, error) {
			req, ok := payload.(*jobs.SubnetAcceptPeerRequest)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}

			return req, nil
		},
		Type:  bInvoke,
		Short: "Accept a peer to a subnet",
		Long: `Invokes the /dms/node/subnet/accept-peer behavior on an actor

This behavior accepts a peer to the specified subnet.

Examples:
  nunet actor cmd --context user /dms/node/subnet/accept-peer --subnet-id <subnet_id> --peer-id <peer_id> --ip <ip>`,
	},

	jobs.SubnetMapPortBehavior: {
		Payload: func() any { return &jobs.SubnetMapPortRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*jobs.SubnetMapPortRequest)

			cmd.Flags().StringVarP(&p.SubnetID, "subnet-id", "i", "", "subnet-id (required)")
			cmd.Flags().StringVarP(&p.Protocol, "protocol", "p", "", "protocol (required)")
			cmd.Flags().StringVarP(&p.SourceIP, "source-ip", "s", "", "source IP address (required)")
			cmd.Flags().StringVarP(&p.SourcePort, "source-port", "o", "", "source port (required)")
			cmd.Flags().StringVarP(&p.DestIP, "dest-ip", "d", "", "destination IP address (required)")
			cmd.Flags().StringVarP(&p.DestPort, "dest-port", "n", "", "destination port (required)")
			_ = cmd.MarkFlagRequired("protocol")
			_ = cmd.MarkFlagRequired("source-ip")
			_ = cmd.MarkFlagRequired("source-port")
			_ = cmd.MarkFlagRequired("dest-ip")
			_ = cmd.MarkFlagRequired("dest-port")
		},
		PayloadEnc: func(payload any) (any, error) {
			req, ok := payload.(*jobs.SubnetMapPortRequest)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}

			return req, nil
		},
		Type:  bInvoke,
		Short: "Map a port",
		Long: `Invokes the /dms/node/subnet/map-port behavior on an actor

This behavior maps a port from the source to the destination.

Examples:
  nunet actor cmd --context user /dms/node/subnet/map-port --protocol <protocol> --source-ip <source_ip> --source-port <source_port> --dest-ip <dest_ip> --dest-port <dest_port>`,
	},

	jobs.SubnetDNSAddRecordBehavior: {
		Payload: func() any { return &jobs.SubnetDNSAddRecordRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*jobs.SubnetDNSAddRecordRequest)

			cmd.Flags().StringVarP(&p.SubnetID, "subnet-id", "s", "", "subnet ID (required)")
			cmd.Flags().StringVarP(&p.DomainName, "domain-name", "n", "", "A record name (required)")
			cmd.Flags().StringVarP(&p.IP, "ip", "i", "", "IP address (required)")
			_ = cmd.MarkFlagRequired("subnet-id")
			_ = cmd.MarkFlagRequired("name")
			_ = cmd.MarkFlagRequired("ip")
		},
		PayloadEnc: func(payload any) (any, error) {
			req, ok := payload.(*jobs.SubnetDNSAddRecordRequest)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}

			return req, nil
		},
		Type:  bInvoke,
		Short: "Add a DNS record",
		Long: `Invokes the /dms/node/subnet/dns/add-record behavior on an actor
This behavior adds a DNS record to the local resolver.

Examples:
  nunet actor cmd --context user /dms/node/subnet/dns/add-record --subnet-id <subnet_id> --name <record_name> --ip <ip>`,
	},

	jobs.SubnetUnmapPortBehavior: {
		Payload: func() any { return &jobs.SubnetUnmapPortRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*jobs.SubnetUnmapPortRequest)

			cmd.Flags().StringVarP(&p.SubnetID, "subnet-id", "i", "", "subnet-id (required)")
			cmd.Flags().StringVarP(&p.Protocol, "protocol", "p", "", "protocol (required)")
			cmd.Flags().StringVarP(&p.SourceIP, "source-ip", "s", "", "source IP address (required)")
			cmd.Flags().StringVarP(&p.SourcePort, "source-port", "o", "", "source port (required)")
			cmd.Flags().StringVarP(&p.DestIP, "dest-ip", "d", "", "destination IP address (required)")
			cmd.Flags().StringVarP(&p.DestPort, "dest-port", "n", "", "destination port (required)")
			_ = cmd.MarkFlagRequired("subnet-id")
			_ = cmd.MarkFlagRequired("protocol")
			_ = cmd.MarkFlagRequired("source-ip")
			_ = cmd.MarkFlagRequired("source-port")
			_ = cmd.MarkFlagRequired("dest-ip")
			_ = cmd.MarkFlagRequired("dest-port")
		},
		PayloadEnc: func(payload any) (any, error) {
			req, ok := payload.(*jobs.SubnetUnmapPortRequest)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}

			return req, nil
		},
		Type:  bInvoke,
		Short: "Unmap a port",
		Long: `Invokes the /dms/node/subnet/unmap-port behavior on an actor

This behavior removes a port mapping.

Examples:
  nunet actor cmd --context user /dms/node/subnet/unmap-port --subnet-id <subnet_id> --protocol <protocol> --source-ip <source_ip> --source-port <source_port> --dest-ip <dest_ip> --dest-port <dest_port>`,
	},

	jobs.SubnetDNSRemoveRecordBehavior: {
		Payload: func() any { return &jobs.SubnetDNSRemoveRecordRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*jobs.SubnetDNSRemoveRecordRequest)

			cmd.Flags().StringVarP(&p.SubnetID, "subnet-id", "s", "", "subnet ID (required)")
			cmd.Flags().StringVarP(&p.DomainName, "domain-name", "n", "", "A record name (required)")
			_ = cmd.MarkFlagRequired("subnet-id")
			_ = cmd.MarkFlagRequired("name")
		},
		PayloadEnc: func(payload any) (any, error) {
			req, ok := payload.(*jobs.SubnetDNSRemoveRecordRequest)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}

			return req, nil
		},
		Type:  bInvoke,
		Short: "Remove a DNS record",
		Long: `Invokes the /dms/node/subnet/dns/remove-record behavior on an actor

This behavior removes a DNS record from the local resolver.

Examples:

  nunet actor cmd --context user /dms/node/subnet/dns/remove-record --subnet-id <subnet_id> --name <record_name>`,
	},

	node.ResourcesAllocatedBehavior: {
		Type:  bInvoke,
		Short: "Get allocated resources",
		Long: `Invokes the /dms/node/resources/allocated behavior on an actor

This behavior retrieves the resources allocated by the node. The resources include CPU, RAM, GPU and disk space.
The returned units are in Hz for CPU clock speed, bytes for RAM, VRAM and disk space.

Examples:
	  nunet actor cmd --context user /dms/node/resources/allocated`,
	},

	node.ResourcesFreeBehavior: {
		Type:  bInvoke,
		Short: "Get free resources",
		Long: `Invokes the /dms/node/resources/free behavior on an actor

This behavior retrieves the free resources available on the node. The resources include CPU, RAM, GPU and disk space.
The returned units are in Hz for CPU clock speed, bytes for RAM, VRAM and disk space.

Examples:
	  nunet actor cmd --context user /dms/node/resources/free`,
	},

	node.ResourcesOnboardedBehavior: {
		Type:  bInvoke,
		Short: "Get onboarded resources",
		Long: `Invokes the /dms/node/resources/onboarded behavior on an actor

This behavior retrieves the resources onboarded to the node. The resources include CPU, RAM, GPU and disk space.
The returned units are in Hz for CPU clock speed, bytes for RAM, VRAM and disk space.

Examples:
	  nunet actor cmd --context user /dms/node/resources/onboarded`,
	},
	node.LoggerConfigBehavior: {
		Payload: func() any { return &node.LoggerConfigRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.LoggerConfigRequest)

			cmd.Flags().StringVarP(&p.URL, "url", "u", "", "Elasticsearch URL")
			cmd.Flags().StringVarP(&p.Level, "level", "l", "", "logging level (info, warn, debug etc.)")
			cmd.Flags().IntVarP(&p.Interval, "interval", "i", 0, "flush interval in seconds")
			cmd.MarkFlagsOneRequired("url", "level", "interval")
			cmd.Flags().StringVar(&p.APIKey, "api-key", "", "API Key for Elasticsearch and APM")
			cmd.Flags().StringVar(&p.APMURL, "apm-url", "", "APM Server URL")
			cmd.Flags().Bool("enable-elastic", false, "Enable Elasticsearch logging")

			// PreRunE function to capture if 'enable-elastic' flag was provided
			cmd.PreRunE = func(cmd *cobra.Command, _ []string) error {
				flag := cmd.Flags().Lookup("enable-elastic")
				if flag != nil && flag.Changed {
					val, err := strconv.ParseBool(flag.Value.String())
					if err != nil {
						return fmt.Errorf("invalid value for --enable-elastic: %v", err)
					}
					p.ElasticEnabled = &val
				}
				return nil
			}
		},
		PayloadEnc: func(payload any) (any, error) {
			req, ok := payload.(*node.LoggerConfigRequest)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}

			return req, nil
		},
		Type:  bInvoke,
		Short: "Adjust logger settings",
		Long: `Invokes the /dms/node/logger/config behavior on an actor

This behavior allows the user to adjust logger settings, i.e. logging level, flush interval and Elasticsearch URL.

Examples:

  nunet actor cmd --context user /dms/node/logger/config --level debug # set debug level
  nunet actor cmd --context user /dms/node/logger/config --url <elasticsearch-url> 
  nunet actor cmd --context user /dms/node/logger/config --interval 10 # flush logs each 10 seconds
  nunet actor cmd --context user /dms/node/logger/config --api-key <api-key>
  nunet actor cmd --context user /dms/node/logger/config --apm-url <apm-url>
  nunet actor cmd --context user /dms/node/logger/config --enable-elastic`,
	},
	node.HardwareSpecBehavior: {
		Type:  bInvoke,
		Short: "Get hardware specifications",
		Long: `Invokes the /dms/node/hardware/spec behavior on an actor

This behavior retrieves the hardware specifications of the system.

Examples:

	nunet actor cmd --context user /dms/node/hardware/spec`,
	},
	node.HardwareUsageBehavior: {
		Type:  bInvoke,
		Short: "Get hardware usage",
		Long: `Invokes the /dms/node/hardware/usage behavior on an actor

This behavior retrieves the hardware usage of the system.

Examples:

	nunet actor cmd --context user /dms/node/hardware/usage`,
	},
}
