package actor

import (
	"errors"
	"fmt"
	"strconv"

	"gitlab.com/nunet/device-management-service/dms/hardware"
	"gitlab.com/nunet/device-management-service/types"

	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/dms/jobs"
	"gitlab.com/nunet/device-management-service/dms/node"
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
		Type:    bInvoke,
		Payload: func() any { return &node.OffboardRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.OffboardRequest)

			cmd.Flags().BoolVarP(&p.Force, "force", "f", false, "force offboard")
		},
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
			p.ExecutionType = jobs.ExecutorFirecracker
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
				ExecutionType: jobs.ExecutorFirecracker,
			}
		},
		Type:  bInvoke,
		Short: "List running VMs",
		Long: `Invokes the /dms/node/vm/list behavior on an actor

This behavior retrieves a list of virtual machines (VMs) running on the node.

Examples:
  nunet actor cmd --context user /dms/node/vm/list`,
	},

	node.SubnetCreateBehavior: {
		Payload: func() any { return &node.SubnetCreateRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.SubnetCreateRequest)

			cmd.Flags().StringVarP(&p.SubnetID, "subnet-id", "s", "", "subnet ID (required)")
			cmd.Flags().StringToStringVarP(&p.RoutingTable, "routing-table", "r", nil, "subnet routing table (required)")
			_ = cmd.MarkFlagRequired("subnet-id")
		},
		PayloadEnc: func(payload any) (any, error) {
			req, ok := payload.(*node.SubnetCreateRequest)
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

	node.SubnetAddPeerBehavior: {
		Payload: func() any { return &node.SubnetAddPeerRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.SubnetAddPeerRequest)

			cmd.Flags().StringVarP(&p.SubnetID, "subnet-id", "s", "", "subnet ID (required)")
			cmd.Flags().StringVarP(&p.PeerID, "peer-id", "p", "", "peer ID (required)")
			cmd.Flags().StringVarP(&p.IP, "ip", "i", "", "peer IP address (required)")
			_ = cmd.MarkFlagRequired("subnet-id")
			_ = cmd.MarkFlagRequired("peer-id")
			_ = cmd.MarkFlagRequired("ip")
		},
		PayloadEnc: func(payload any) (any, error) {
			req, ok := payload.(*node.SubnetAddPeerRequest)
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

	node.SubnetAcceptPeerBehavior: {
		Payload: func() any { return &node.SubnetAcceptPeerRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.SubnetAcceptPeerRequest)

			cmd.Flags().StringVarP(&p.SubnetID, "subnet-id", "s", "", "subnet ID (required)")
			cmd.Flags().StringVarP(&p.PeerID, "peer-id", "p", "", "peer ID (required)")
			cmd.Flags().StringVarP(&p.IP, "ip", "i", "", "peer IP address (required)")
			_ = cmd.MarkFlagRequired("subnet-id")
			_ = cmd.MarkFlagRequired("peer-id")
			_ = cmd.MarkFlagRequired("ip")
		},
		PayloadEnc: func(payload any) (any, error) {
			req, ok := payload.(*node.SubnetAcceptPeerRequest)
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

	node.SubnetMapPortBehavior: {
		Payload: func() any { return &node.SubnetMapPortRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.SubnetMapPortRequest)

			cmd.Flags().StringVarP(&p.Protocol, "protocol", "p", "", "protocol (required)")
			cmd.Flags().StringVarP(&p.SourceIP, "source-ip", "s", "", "source IP address (required)")
			cmd.Flags().StringVarP(&p.SourcePort, "source-port", "o", "", "source port (required)")
			cmd.Flags().StringVarP(&p.DestIP, "dest-ip", "i", "", "destination IP address (required)")
			cmd.Flags().StringVarP(&p.DestPort, "dest-port", "n", "", "destination port (required)")
			_ = cmd.MarkFlagRequired("protocol")
			_ = cmd.MarkFlagRequired("source-ip")
			_ = cmd.MarkFlagRequired("source-port")
			_ = cmd.MarkFlagRequired("dest-ip")
			_ = cmd.MarkFlagRequired("dest-port")
		},
		PayloadEnc: func(payload any) (any, error) {
			req, ok := payload.(*node.SubnetMapPortRequest)
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

	node.SubnetDNSAddRecordBehavior: {
		Payload: func() any { return &node.SubnetDNSAddRecordRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.SubnetDNSAddRecordRequest)

			cmd.Flags().StringVarP(&p.SubnetID, "subnet-id", "s", "", "subnet ID (required)")
			cmd.Flags().StringVarP(&p.DomainName, "domain-name", "n", "", "A record name (required)")
			cmd.Flags().StringVarP(&p.IP, "ip", "i", "", "IP address (required)")
			_ = cmd.MarkFlagRequired("subnet-id")
			_ = cmd.MarkFlagRequired("name")
			_ = cmd.MarkFlagRequired("ip")
		},
		PayloadEnc: func(payload any) (any, error) {
			req, ok := payload.(*node.SubnetDNSAddRecordRequest)
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
}

func onboardBehaviorPreRun(_ *Command, payload any) error {
	p, ok := payload.(*node.OnboardRequest)
	if !ok {
		return ErrInvalidArgument
	}

	// TODO: we need to have single instance of hardware manager
	// Should we do an api call here?
	hardwareManager := hardware.NewHardwareManager()
	machineResources, err := hardwareManager.GetMachineResources()
	if err != nil {
		return fmt.Errorf("could not get machine resources: %w", err)
	}

	p.Config.OnboardedResources.CPU.ClockSpeed = machineResources.CPU.ClockSpeed
	if len(machineResources.GPUs) != 0 {
		var (
			gpuMap         = make(map[string]types.GPU)
			gpuPromptItems []*selectPromptItem
		)
		for _, gpu := range machineResources.GPUs {
			gpuMap[gpu.Model] = gpu
			gpuPromptItems = append(gpuPromptItems, &selectPromptItem{
				Label: gpu.Model,
			})
		}

		res, err := selectPrompt("Select GPU", gpuPromptItems)
		if err != nil {
			return fmt.Errorf("could not select GPU: %w", err)
		}

		vramValidator := func(input string) error {
			if _, err := strconv.ParseFloat(input, 64); err != nil {
				return fmt.Errorf("invalid input: %w", err)
			}
			return nil
		}
		for _, gpuName := range res {
			input, err := prompt("Enter VRAM in GB", vramValidator)
			if err != nil {
				return fmt.Errorf("could not prompt for VRAM: %w", err)
			}

			vram, err := strconv.ParseFloat(input, 64)
			if err != nil {
				return fmt.Errorf("could not parse VRAM: %w", err)
			}

			gpu := gpuMap[gpuName]
			gpu.VRAM = types.ConvertGBToBytes(vram)
			p.Config.OnboardedResources.GPUs = append(p.Config.OnboardedResources.GPUs, gpu)
		}
	} else {
		fmt.Println("No GPUs found. Skipping GPU selection.")
	}

	p.Config.OnboardedResources.CPU.ClockSpeed = machineResources.CPU.ClockSpeed
	return nil
}
