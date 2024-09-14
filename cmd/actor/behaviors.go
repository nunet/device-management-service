package actor

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/executor/firecracker"
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
	PayloadEnc  func(cmd *Command, payload any) (any, error)
	SetFlags    func(cmd *Command, payload any)
	ValidArgsFn func(cmd *Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)
	Args        cobra.PositionalArgs
}

var behaviors = map[string]behaviorConfig{
	// /public/hello
	node.PublicHelloBehavior: {
		Type: bInvoke,
	},
	// /broadcast/hello
	node.BroadcastHelloBehavior: {
		Type:  bBroadcast,
		Topic: node.BroadcastHelloTopic,
	},
	// /public/status
	node.PublicStatusBehavior: {
		Type: bInvoke,
	},

	// /dms/node/peers/list
	node.PeersListBehavior: {
		Type: bInvoke,
	},
	// /dms/node/peers/self
	node.PeerAddrInfoBehavior: {
		Type: bInvoke,
	},
	// /dms/node/peers/ping
	node.PeerPingBehavior: {
		Type:    bInvoke,
		Payload: func() any { return &node.PingRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.PingRequest)

			cmd.Flags().StringVarP(&p.Host, "host", "H", "", "Host address to ping")
			_ = cmd.MarkFlagRequired("host")
		},
		PayloadEnc: func(_ *Command, payload any) (any, error) {
			req, ok := payload.(*node.PingRequest)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}
			return req, nil
		},
	},
	// /dms/node/peers/dht
	node.PeerDHTBehavior: {
		Type: bInvoke,
	},
	// /dms/node/peers/connect
	node.PeerConnectBehavior: {
		Type:    bInvoke,
		Payload: func() any { return &node.PeerConnectRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.PeerConnectRequest)

			cmd.Flags().StringVarP(&p.Address, "address", "a", "", "Peer address to connect to")
			_ = cmd.MarkFlagRequired("address")
		},
		PayloadEnc: func(_ *Command, payload any) (any, error) {
			req, ok := payload.(*node.PeerConnectRequest)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}
			return req, nil
		},
	},
	// /dms/node/peers/score
	node.PeerScoreBehavior: {
		Type: bInvoke,
	},
	// /dms/node/onboarding/onboard
	node.OnboardBehaviour: {
		Type:    bInvoke,
		Payload: func() any { return &node.OnboardRequest{} },
		SetFlags: func(cmd *Command, payload any) {
			// infer the type of the payload
			p := payload.(*node.OnboardRequest)
			cmd.Flags().Uint64VarP(&p.Config.Memory, "memory", "m", 0, "set value for memory usage")
			cmd.Flags().Int64VarP(&p.Config.CPU, "cpu", "z", 0, "set value for CPU usage")
			cmd.Flags().StringVarP(&p.Config.Channel, "nunet-channel", "n", "", "set channel")
			cmd.Flags().StringVarP(&p.Config.PaymentAddress, "wallet", "w", "", "set wallet address")
			cmd.Flags().Float64VarP(&p.Config.NTXPricePerMinute, "ntx-price", "x", 0, "price in NTX per minute for onboarded compute resource")
			cmd.Flags().BoolVarP(&p.Config.IsAvailable, "available", "a", false, "unavailable for job deployment (default: false)")
			cmd.Flags().BoolVarP(&p.Config.ServerMode, "local-enable", "l", true, "set server mode (enable for local)")
			cmd.MarkFlagsRequiredTogether("memory", "cpu", "nunet-channel", "wallet")
		},
		PayloadEnc: func(_ *Command, payload any) (any, error) {
			req, ok := payload.(*node.OnboardRequest)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}
			return req, nil
		},
	},
	// /dms/node/onboarding/offboard
	node.OffboardBehaviour: {
		Type:    bInvoke,
		Payload: func() any { return &node.OffboardRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.OffboardRequest)

			cmd.Flags().BoolVarP(&p.Force, "force", "f", false, "force offboard")
		},
		PayloadEnc: func(_ *Command, payload any) (any, error) {
			req, ok := payload.(*node.OffboardRequest)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}
			return req, nil
		},
	},
	// /dms/node/onboarding/status
	node.OnboardStatusBehaviour: {
		Type: bInvoke,
	},
	// /dms/node/onboarding/resource
	node.OnboardResourceBehaviour: {
		Type: bInvoke,
	},
	// /dms/node/vm/start/custom
	node.CustomVMStart: {
		Type:    bInvoke,
		Payload: func() any { return &vmStartOpts{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*vmStartOpts)
			cmd.Flags().StringVarP(&p.Engine.KernelImage, "kernel", "k", "", "path to kernel image file")
			cmd.Flags().StringVarP(&p.Engine.RootFileSystem, "rootfs", "r", "", "path to root fs image file")
			cmd.Flags().StringVarP(&p.Engine.Initrd, "initrd", "i", "", "path to initial ram disk")
			cmd.Flags().StringVarP(&p.Engine.KernelArgs, "args", "a", "", "arguments to pas to the kernel")
			cmd.Flags().Float32VarP(&p.Resources.CPU.Cores, "cpu", "z", 1, "CPU cores to allocate")
			cmd.Flags().Uint64VarP(&p.Resources.RAM.Size, "memory", "m", 1024, "Memory to allocate")
			_ = cmd.MarkFlagRequired("kernel")
			_ = cmd.MarkFlagFilename("kernel")
			_ = cmd.MarkFlagRequired("rootfs")
			_ = cmd.MarkFlagFilename("rootfs")
		},
		PayloadEnc: func(_ *Command, payload any) (any, error) {
			opts, ok := payload.(*vmStartOpts)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}
			engine := firecracker.NewFirecrackerEngineBuilder(opts.Engine.RootFileSystem)
			engine = engine.WithKernelImage(opts.Engine.KernelImage)
			engine = engine.WithKernelArgs(opts.Engine.KernelArgs)
			engine = engine.WithInitrd(opts.Engine.Initrd)
			es := engine.Build()
			req := node.CustomVMStartRequest{
				Execution: types.ExecutionRequest{
					ExecutionID: uuid.New().String(),
					EngineSpec:  es,
					Resources:   &opts.Resources,
				},
			}
			return req, nil
		},
	},
	// /dms/node/vm/stop
	node.VMStop: {
		Payload: func() any { return &node.VMStopRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.VMStopRequest)

			cmd.Flags().StringVarP(&p.ExecutionID, "id", "i", "", "execution id of the vm")
			_ = cmd.MarkFlagRequired("host")
		},
		PayloadEnc: func(_ *Command, payload any) (any, error) {
			req, ok := payload.(*node.VMStopRequest)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}
			return req, nil
		},
		Type: bInvoke,
	},
	// /dms/node/vm/list
	node.VMList: {
		Type: bInvoke,
	},
}

type vmStartOpts struct {
	Engine    firecracker.EngineSpec
	Resources types.Resources
}
