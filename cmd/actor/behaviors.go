// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package actor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/client"
	"gitlab.com/nunet/device-management-service/cmd/cli"
	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/dms/orchestrator"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/utils/convert"
)

type BehaviorAction string

const (
	bBroadcast BehaviorAction = "broadcast"
	bInvoke    BehaviorAction = "invoke"
	bSend      BehaviorAction = "send"
)

var ErrInvalidArgument = errors.New("invalid argument")

type Command = cobra.Command

type behaviorConfig struct {
	Payload     func() any
	Behavior    string
	Action      BehaviorAction
	SetFlags    func(cmd *Command, payload any)
	RunFn       func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error)
	PreRunFn    func(cmd *Command, opts actorCmdOptions) error
	ValidArgsFn func(cmd *Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)
	Args        cobra.PositionalArgs
	Long        string
	Short       string
}

func (b *behaviorConfig) Run(ctx context.Context, dmsCli *cli.DmsCLI, opts actorCmdOptions, streams cli.Streams) error {
	// Create security context first
	sctx, err := utils.NewSecurityContext(dmsCli, opts.Context)
	if err != nil {
		return fmt.Errorf("could not create security context: %w", err)
	}

	// Now call newClient with the correct arguments
	dmsClient, err := dmsCli.NewClient(sctx)
	if err != nil {
		return fmt.Errorf("could not create client: %w", err)
	}
	res, err := b.RunFn(ctx, dmsCli, dmsClient, opts)
	if err != nil {
		return fmt.Errorf("could not run behavior: %w", err)
	}
	return displayResponse(streams.Out, res)
}

type NewDeploymentRequestCmd struct {
	Config string
}

type CapAnchorRequestCmd struct {
	Root    bool
	Require bool
	Provide bool
	Revoke  bool
	Data    string
}

type CreateVolumeRequestCmd struct {
	ClientPEMFile string
	VolumeName    string
	CAOutputDir   string
}

var registeredBehaviors = map[string]*behaviorConfig{
	// /dms/volume/create
	behaviors.VolumeCreateBehavior: {
		Payload: func() any { return &CreateVolumeRequestCmd{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*CreateVolumeRequestCmd)
			cmd.Flags().StringVarP(&p.VolumeName, "name", "n", "", "name (required)")
			cmd.Flags().StringVarP(&p.ClientPEMFile, "client-pem-file", "p", "", "client-pem-file (required)")
			cmd.Flags().StringVarP(&p.CAOutputDir, "ca-output-dir", "", "", "ca-output-dir (required)")

			_ = cmd.MarkFlagRequired("name")
			_ = cmd.MarkFlagRequired("client-pem-file")
			_ = cmd.MarkFlagRequired("ca-output-dir")
		},
		RunFn: func(ctx context.Context, dmsCli *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*CreateVolumeRequestCmd)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}

			afs := afero.Afero{Fs: dmsCli.FS()}
			data, err := afs.ReadFile(req.ClientPEMFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}

			// validate client pem

			cfg := &node.CreateVolumeRequest{
				Name:      req.VolumeName,
				ClientPEM: string(data),
			}

			resp, err := dmsClient.CreateVolume(ctx, *cfg, opts.MsgOpts...)
			if err != nil {
				return resp, err
			}

			err = afs.WriteFile(filepath.Join(req.CAOutputDir, "glusterfs.ca"), []byte(resp.CAData), 0o775)
			if err != nil {
				return resp, err
			}

			return resp, nil
		},
		Action: bInvoke,
		Short:  "Send a create volume message",
		Long: `Invoke the /dms/volume/create behavior on an actor
	
	This behavior calls the actors create volume behaviour.
	
	Examples:
	
	  nunet actor cmd --context user /dms/volume/create --name <volname> --client-pem-file <filename>`,
	},
	// /dms/volume/delete
	behaviors.VolumeDeleteBehavior: {
		Payload: func() any { return &node.DeleteVolumeRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.DeleteVolumeRequest)

			cmd.Flags().StringVarP(&p.Name, "name", "n", "", "name (required)")

			_ = cmd.MarkFlagRequired("name")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*node.DeleteVolumeRequest)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}

			return dmsClient.DeleteVolume(ctx, *req, opts.MsgOpts...)
		},
		Action: bInvoke,
		Short:  "Send a delete volume message",
		Long: `Invoke the /dms/volume/delete behavior on an actor
		
		This behavior calls the actors delete volume behaviour.
		
		Examples:
		
		  nunet actor cmd --context user /dms/volume/delete --name <volname>`,
	},
	// /dms/volume/start
	behaviors.VolumeStartBehavior: {
		Payload: func() any { return &node.StartVolumeRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.StartVolumeRequest)

			cmd.Flags().StringVarP(&p.Name, "name", "n", "", "name (required)")

			_ = cmd.MarkFlagRequired("name")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*node.StartVolumeRequest)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}

			return dmsClient.StartVolume(ctx, *req, opts.MsgOpts...)
		},
		Action: bInvoke,
		Short:  "Send a start volume message",
		Long: `Invoke the /dms/volume/start behavior on an actor
		
		This behavior calls the actors start volume behaviour.
		
		Examples:
		
		  nunet actor cmd --context user /dms/volume/start --name <volname>`,
	},
	// /public/hello
	behaviors.PublicHelloBehavior: {
		Action: bInvoke,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.Hello(ctx, opts.MsgOpts...)
		},
		Short: "Invoke a 'hello' message",
		Long: `Invoke the /public/hello behavior on an actor

This behavior invokes a "hello" for a polite introduction.

Examples:

  nunet actor cmd --context user /public/hello
  nunet actor cmd --context user /public/hello --dest <did/peer_id/actor_handle>`,
	},
	// /broadcast/hello
	behaviors.BroadcastHelloBehavior: {
		Action: bBroadcast,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.BroadcastHello(ctx, opts.MsgOpts...)
		},
		Short: "Broadcast a 'hello' message to a topic",
		Long: `Invokes the /broadcast/hello behavior on an actor

This behavior sends a "hello" message to a broadcast topic for polite introduction.

Examples:

  nunet actor cmd --context user /broadcast/hello`,
	},
	// /public/status
	behaviors.PublicStatusBehavior: {
		Action: bInvoke,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.Status(ctx, opts.MsgOpts...)
		},
		Short: "Retrieve actor status",
		Long: `Invokes the /public/status behavior on an actor

This behavior retrieves the status and resources information.

Examples:
  nunet actor cmd --context user /public/status # own actor status
  nunet actor cmd --context user /public/status --dest <did/peer_id/actor_handle> # status of specified destination`,
	},
	// /dms/node/status
	behaviors.StatusDiscoveryBehavior: {
		Action: bInvoke,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.Discovery(ctx, opts.MsgOpts...)
		},
		Short: "Invoke a 'status discovery' message",
		Long: `Invoke the /dms/node/status behavior on an actor
	
	This behavior invokes a "status discovery" behavior for fleet discovery.
	
	Examples:
	
	  nunet actor cmd --context user /dms/node/status
	  nunet actor cmd --context user /dms/node/status --dest <did/peer_id/actor_handle>`,
	},
	// /broadcast/dms/status
	behaviors.BroadcastStatusDiscoveryBehavior: {
		Action: bBroadcast,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.DiscoveryBroadcast(ctx, opts.MsgOpts...)
		},
		Short: "Broadcast a 'status discovery' message to a topic",
		Long: `Broadcast the /broadcast/dms/status behavior to nodes in the network
	
	This behavior broadcasts a "status discovery" message to topic /nunet/status for fleet discovery.
	
	Examples:
	
	  nunet actor cmd --context user /broadcast/dms/status`,
	},
	// /dms/node/peers/list
	behaviors.PeersListBehavior: {
		Action: bInvoke,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.PeersList(ctx, opts.MsgOpts...)
		},
		Short: "List connected peers",
		Long: `Invokes the /dms/node/peers/list behavior on an actor

This behavior retrieves a list of connected peers.

Examples:
  nunet actor cmd --context user /dms/node/peers/list # own node actor peer list
  nunet actor cmd --context user /dms/node/peers/list --dest <did/peer_id/actor_handle> # specified node actor peer list`,
	},
	// /dms/node/peers/self
	behaviors.PeerAddrInfoBehavior: {
		Action: bInvoke,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.PeersSelf(ctx, opts.MsgOpts...)
		},
		Short: "Get peer's ID and addresses",
		Long: `Invokes the /dms/node/peers/self behavior on an actor

This behavior retrieves information about the node itself, such as its ID or addresses.

Examples:
  nunet actor cmd --context user /dms/node/peers/self # own node actor peer ID
  nunet actor cmd --context user /dms/node/peers/self --dest <did/peer_id/actor_handle> # specified node actor peer ID`,
	},
	// /dms/node/peers/ping
	behaviors.PeerPingBehavior: {
		Action:  bInvoke,
		Payload: func() any { return &node.PingRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.PingRequest)

			cmd.Flags().StringVarP(&p.Host, "host", "H", "", "host address to ping (required)")
			_ = cmd.MarkFlagRequired("host")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*node.PingRequest)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}
			return dmsClient.PeerPing(ctx, *req, opts.MsgOpts...)
		},
		Short: "Ping a peer",
		Long: `Invokes the /dms/node/peers/ping behavior on an actor

This behavior establishes a ping connection with a peer.

Examples:
  nunet actor cmd --context user /dms/node/peers/ping --host <peer_id>`,
	},
	// /dms/node/peers/dht
	behaviors.PeerDHTBehavior: {
		Action: bInvoke,
		// TODO: Check the actual implementation?
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.PeersListFromDHT(ctx, opts.MsgOpts...)
		},
		Short: "List peers connected to DHT",
		Long: `Invokes the /dms/node/peers/dht behavior on an actor

This behavior returns a list of peers from the  Distributed Hash Table (DHT) used for peer discovery and content routing.

Examples:
  nunet actor cmd --context user /dms/node/peers/dht`,
	},
	// /dms/node/peers/connect
	behaviors.PeerConnectBehavior: {
		Action:  bInvoke,
		Payload: func() any { return &node.PeerConnectRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.PeerConnectRequest)

			cmd.Flags().StringVarP(&p.Address, "address", "a", "", "peer address to connect to (required)")
			_ = cmd.MarkFlagRequired("address")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*node.PeerConnectRequest)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}
			return dmsClient.PeerConnect(ctx, *req, opts.MsgOpts...)
		},
		Short: "Connect to a peer",
		Long: `Invokes the /dms/node/peers/connect behavior on an actor

This behavior initiates a connection to a specified peer.

Examples:
  nunet actor cmd --context user /dms/node/peers/connect --address /p2p/<peer_id>`,
	},
	// /dms/node/peers/score
	behaviors.PeerScoreBehavior: {
		Action: bInvoke,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.PeerScore(ctx, opts.MsgOpts...)
		},
		Short: "Retrieves gossipsub broadcast score",
		Long: `Invokes the /dms/node/peers/score behavior on an actor

This behavior retrieves a snapshot of the peer's gossipsub broadcast score.

Examples:
  nunet actor cmd --context user /dms/node/peers/score`,
	},
	// /dms/node/onboarding/onboard
	behaviors.OnboardBehavior: {
		Action:  bInvoke,
		Payload: func() any { return &OnboardingInput{} },
		SetFlags: func(cmd *Command, payload any) {
			// infer the type of the payload
			p := payload.(*OnboardingInput)
			cmd.Flags().StringVarP(&p.RAMSize, "ram", "R", "0GiB", "set the amount of memory to reserve for NuNet (defaults to GiB)")
			cmd.Flags().Float32VarP(&p.CPUCores, "cpu", "C", 0, "set the number of CPU cores to reserve for NuNet")
			cmd.Flags().StringVarP(&p.DiskSize, "disk", "D", "0GiB", "set the amount of disk size to reserve for NuNet (defaults to GiB)")
			cmd.Flags().StringVarP(&p.GPUsStr, "gpus", "G", "", "comma-separated list of GPU Index and VRAM in GiB (e.g. 0:4,1:8). The gpu index can be obtained from 'nunet gpu list' command. Unit can be specified for the VRAM but defaults to GiB")
			cmd.Flags().BoolVarP(&p.NoGPU, "no-gpu", "N", false, "do not reserve any GPU resources")
			cmd.MarkFlagsOneRequired("ram", "cpu", "disk")
			cmd.MarkFlagsRequiredTogether("ram", "cpu", "disk")
		},
		PreRunFn: onboardBehaviorPreRun,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			p, ok := opts.Payload.(*OnboardingInput)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}

			req := node.OnboardRequest{}
			req.Config.OnboardedResources.CPU.Cores = p.CPUCores
			req.NoGPU = p.NoGPU
			req.Config.OnboardedResources.GPUs = p.GPUs

			var err error
			// convert RAM and Disk from specified unit to bytes if specified otherwise, default to GiB
			req.Config.OnboardedResources.RAM.Size, err = convert.ParseBytesWithDefaultUnit(p.RAMSize, "GiB")
			if err != nil {
				return nil, fmt.Errorf("failed to decode RAM size. Expected Unit in GiB")
			}
			req.Config.OnboardedResources.Disk.Size, err = convert.ParseBytesWithDefaultUnit(p.DiskSize, "GiB")
			if err != nil {
				return nil, fmt.Errorf("failed to decode Disk size. Expected Unit in GiB")
			}

			return dmsClient.Onboard(ctx, req, opts.MsgOpts...)
		},
		Short: "Onboard a node to the network",
		Long: `Invokes the /dms/node/onboarding/onboard behavior on an actor

This behavior is used to onboard a node to the DMS, making its resources available for use.

Examples:
  nunet actor cmd --context user /dms/node/onboarding/onboard --disk 1 --ram 1 --cpu 2`,
	},
	// /dms/node/onboarding/offboard
	behaviors.OffboardBehavior: {
		Action: bInvoke,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req := node.OffboardRequest{}
			return dmsClient.Offboard(ctx, req, opts.MsgOpts...)
		},
		Short: "Offboard a node from the network",
		Long: `Invokes the /dms/node/onboarding/offboard behavior on an actor

This behavior is used to offboard a node from the DMS (Device Management Service).

Examples:
  nunet actor cmd --context user /dms/node/onboarding/offboard
  nunet actor cmd --context user /dms/node/onboarding/offboard --force`,
		// TODO: there is no flag set for --force
	},
	// /dms/node/onboarding/status
	behaviors.OnboardStatusBehavior: {
		Action: bInvoke,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.OnboardStatus(ctx, opts.MsgOpts...)
		},
		Short: "Retrieve onboarding status of a node",
		Long: `Invokes the /dms/node/onboarding/status behavior on an actor

This behavior is used to check the onboarding status of a node.

Examples:
  nunet actor cmd --context user /dms/node/onboarding/status`,
	},

	// /dms/node/deployment/list
	behaviors.DeploymentListBehavior: {
		Action:  bInvoke,
		Payload: func() any { return &node.DeploymentListRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.DeploymentListRequest)

			cmd.Flags().StringToStringVarP(&p.Metadata, "filter", "f", nil, "metadata filter to filter deployments (optional)")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*node.DeploymentListRequest)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}
			return dmsClient.DeploymentList(ctx, *req, opts.MsgOpts...)
		},
		Short: "List deployments",
		Long: `Invokes the /dms/node/deployment/list behavior on an actor

This behavior retrieves a list of all deployments on the node.

Examples:
  nunet actor cmd --context user /dms/node/deployment/list --filter "<metadata_key>=<metadata_value>"`,
	},

	// /dms/node/deployment/status
	behaviors.DeploymentStatusBehavior: {
		Action:  bInvoke,
		Payload: func() any { return &node.DeploymentStatusRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.DeploymentStatusRequest)
			cmd.Flags().StringVarP(&p.ID, "id", "i", "", "deployment ID (required)")
			_ = cmd.MarkFlagRequired("id")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*node.DeploymentStatusRequest)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}
			return dmsClient.DeploymentStatus(ctx, *req, opts.MsgOpts...)
		},
		Short: "Get deployment status",
		Long: `Invokes the /dms/node/deployment/status behavior on an actor

This behavior retrieves the status of a specific deployment.

Examples:
  nunet actor cmd --context user /dms/node/deployment/status --id <deployment_id>`,
	},

	// /dms/node/deployment/logs
	behaviors.DeploymentLogsBehavior: {
		Action:  bInvoke,
		Payload: func() any { return &node.DeploymentLogsRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.DeploymentLogsRequest)
			cmd.Flags().StringVarP(&p.EnsembleID, "id", "i", "", "ensemble ID (required)")
			cmd.Flags().StringVarP(&p.AllocationName, "allocation", "a", "", "allocation name (required)")
			_ = cmd.MarkFlagRequired("id")
			_ = cmd.MarkFlagRequired("allocation")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*node.DeploymentLogsRequest)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}
			return dmsClient.DeploymentLogs(ctx, *req, opts.MsgOpts...)
		},
		Short: "Get deployment logs",
		Long: `Invokes the /dms/node/deployment/logs behavior on an actor

This behavior retrieves the logs of a specific deployment, writing it to a file
with path returned in the response.

Examples:
  nunet actor cmd --context user /dms/node/deployment/logs --id <deployment_id> --allocation <allocation_name>`,
	},

	// /dms/node/deployment/manifest
	behaviors.DeploymentManifestBehavior: {
		Action:  bInvoke,
		Payload: func() any { return &node.DeploymentManifestRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.DeploymentManifestRequest)
			cmd.Flags().StringVarP(&p.ID, "id", "i", "", "deployment ID (required)")
			_ = cmd.MarkFlagRequired("id")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*node.DeploymentManifestRequest)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}
			return dmsClient.DeploymentManifest(ctx, *req, opts.MsgOpts...)
		},
		Short: "Get deployment manifest",
		Long: `Invokes the /dms/node/deployment/manifest behavior on an actor

This behavior retrieves the manifest of a specific deployment.

Examples:
  nunet actor cmd --context user /dms/node/deployment/manifest --id <deployment_id>`,
	},

	// /dms/node/deployment/shutdown
	behaviors.DeploymentShutdownBehavior: {
		Action:  bInvoke,
		Payload: func() any { return &node.DeploymentShutdownRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.DeploymentShutdownRequest)
			cmd.Flags().StringVarP(&p.ID, "id", "i", "", "deployment ID (required)")
			_ = cmd.MarkFlagRequired("id")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*node.DeploymentShutdownRequest)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}
			return dmsClient.DeploymentShutdown(ctx, *req, opts.MsgOpts...)
		},
		Short: "Shutdown a deployment",
		Long: `Invokes the /dms/node/deployment/shutdown behavior on an actor

This behavior shuts down a specific deployment.

Examples:
  nunet actor cmd --context user /dms/node/deployment/shutdown --id <deployment_id>`,
	},

	behaviors.NewDeploymentBehavior: {
		Action: bInvoke,
		Short:  "Create a new deployment",
		Long: `Invokes the /dms/node/deployment/new behavior on an actor

This behavior creates a new deployment.

Examples:
  nunet actor cmd --context user /dms/node/deployment/new --spec-file <path to ensemble specification file>`,
		Payload: func() any { return &NewDeploymentRequestCmd{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*NewDeploymentRequestCmd)
			cmd.Flags().StringVarP(&p.Config, "spec-file", "f", "ensemble.yaml", "path of the ensemble specification file (required)")
		},
		RunFn: func(ctx context.Context, dmsCli *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*NewDeploymentRequestCmd)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}

			cfg, err := ProcessEnsembleYaml(afero.Afero{Fs: dmsCli.FS()}, req.Config)
			if err != nil {
				return nil, fmt.Errorf("failed to process config file: %w", err)
			}

			return dmsClient.DeploymentNew(ctx, *cfg, opts.MsgOpts...)
		},
	},

	behaviors.SubnetCreateBehavior.Static: {
		Payload: func() any { return &orchestrator.SubnetCreateRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*orchestrator.SubnetCreateRequest)

			cmd.Flags().StringVarP(&p.SubnetID, "subnet-id", "s", "", "subnet ID (required)")
			cmd.Flags().StringToStringVarP(&p.RoutingTable, "routing-table", "r", nil, "subnet routing table (required)")
			_ = cmd.MarkFlagRequired("subnet-id")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*orchestrator.SubnetCreateRequest)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}
			return dmsClient.SubnetCreate(ctx, *req, opts.MsgOpts...)
		},
		Action: bInvoke,
		Short:  "Create a subnet",
		Long: `Invokes the /dms/node/subnet/create behavior on an actor

This behavior creates a new subnet with the specified subnet ID, IP address, and routing table.

Examples:
  nunet actor cmd --context user /dms/node/subnet/create --subnet-id <subnet_id> --ip <ip> --routing-table <routing_table>`,
	},

	behaviors.SubnetDestroyBehavior.Static: {
		Payload: func() any { return &orchestrator.SubnetDestroyRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*orchestrator.SubnetDestroyRequest)

			cmd.Flags().StringVarP(&p.SubnetID, "subnet-id", "s", "", "subnet ID (required)")
			_ = cmd.MarkFlagRequired("subnet-id")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*orchestrator.SubnetDestroyRequest)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}
			return dmsClient.SubnetDestroy(ctx, *req, opts.MsgOpts...)
		},
		Action: bInvoke,
		Short:  "Destroy a subnet",
		Long: `Invokes the /dms/node/subnet/destroy behavior on an actor

This behavior destroys the specified subnet.

Examples:
  nunet actor cmd --context user /dms/node/subnet/destroy --subnet-id <subnet_id>`,
	},

	behaviors.SubnetAddPeerBehavior: {
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
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*jobs.SubnetAddPeerRequest)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}
			return dmsClient.SubnetAddPeer(ctx, *req, opts.MsgOpts...)
		},
		Action: bInvoke,
		Short:  "Add a peer to a subnet",
		Long: `Invokes the /dms/node/subnet/add-peer behavior on an actor

This behavior adds a peer to the specified subnet.

Examples:
  nunet actor cmd --context user /dms/node/subnet/add-peer --subnet-id <subnet_id> --peer-id <peer_id> --ip <ip>`,
	},

	behaviors.SubnetRemovePeerBehavior: {
		Payload: func() any { return &jobs.SubnetRemovePeerRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*jobs.SubnetRemovePeerRequest)

			cmd.Flags().StringVarP(&p.SubnetID, "subnet-id", "s", "", "subnet ID (required)")
			cmd.Flags().StringVarP(&p.PeerID, "peer-id", "p", "", "peer ID (required)")
			_ = cmd.MarkFlagRequired("subnet-id")
			_ = cmd.MarkFlagRequired("peer-id")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*jobs.SubnetRemovePeerRequest)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}
			return dmsClient.SubnetRemovePeer(ctx, *req, opts.MsgOpts...)
		},
		Action: bInvoke,
		Short:  "Remove a peer from a subnet",
		Long: `Invokes the /dms/node/subnet/remove-peer behavior on an actor

This behavior removes a peer from the specified subnet.

Examples:
  nunet actor cmd --context user /dms/node/subnet/remove-peer --subnet-id <subnet_id> --peer-id <peer_id>`,
	},

	behaviors.SubnetAcceptPeerBehavior: {
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
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*jobs.SubnetAcceptPeerRequest)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}
			return dmsClient.SubnetAcceptPeer(ctx, *req, opts.MsgOpts...)
		},
		Action: bInvoke,
		Short:  "Accept a peer to a subnet",
		Long: `Invokes the /dms/node/subnet/accept-peer behavior on an actor

This behavior accepts a peer to the specified subnet.

Examples:
  nunet actor cmd --context user /dms/node/subnet/accept-peer --subnet-id <subnet_id> --peer-id <peer_id> --ip <ip>`,
	},

	behaviors.SubnetMapPortBehavior: {
		Payload: func() any { return &jobs.SubnetMapPortRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*jobs.SubnetMapPortRequest)

			cmd.Flags().StringVarP(&p.SubnetID, "subnet-id", "i", "", "subnet-id (required)")
			cmd.Flags().StringVarP(&p.Protocol, "protocol", "p", "", "protocol (required)")
			cmd.Flags().StringVarP(&p.SourceIP, "source-ip", "s", "", "source IP address (required)")
			cmd.Flags().StringVarP(&p.SourcePort, "source-port", "o", "", "source port (required)")
			cmd.Flags().StringVarP(&p.DestIP, "dest-ip", "D", "", "destination IP address (required)")
			cmd.Flags().StringVarP(&p.DestPort, "dest-port", "n", "", "destination port (required)")
			_ = cmd.MarkFlagRequired("protocol")
			_ = cmd.MarkFlagRequired("source-ip")
			_ = cmd.MarkFlagRequired("source-port")
			_ = cmd.MarkFlagRequired("dest-ip")
			_ = cmd.MarkFlagRequired("dest-port")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*jobs.SubnetMapPortRequest)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}
			return dmsClient.SubnetMapPort(ctx, *req, opts.MsgOpts...)
		},
		Action: bInvoke,
		Short:  "Map a port",
		Long: `Invokes the /dms/node/subnet/map-port behavior on an actor

This behavior maps a port from the source to the destination.

Examples:
  nunet actor cmd --context user /dms/node/subnet/map-port --protocol <protocol> --source-ip <source_ip> --source-port <source_port> --dest-ip <dest_ip> --dest-port <dest_port>`,
	},

	behaviors.SubnetDNSAddRecordsBehavior: {
		Payload: func() any { return &jobs.SubnetDNSAddRecordsRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*jobs.SubnetDNSAddRecordsRequest)

			cmd.Flags().StringVarP(&p.SubnetID, "subnet-id", "s", "", "subnet ID (required)")
			cmd.Flags().StringToStringVarP(&p.Records, "records", "r", nil, "A record name (required)")
			_ = cmd.MarkFlagRequired("subnet-id")
			_ = cmd.MarkFlagRequired("records")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*jobs.SubnetDNSAddRecordsRequest)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}
			return dmsClient.SubnetDNSAddRecords(ctx, *req, opts.MsgOpts...)
		},
		Action: bInvoke,
		Short:  "Add a DNS record",
		Long: `Invokes the /dms/node/subnet/dns/add-record behavior on an actor
This behavior adds a DNS record to the local resolver.

Examples:
  nunet actor cmd --context user /dms/node/subnet/dns/add-record --subnet-id <subnet_id> --name <record_name> --ip <ip>`,
	},

	behaviors.SubnetUnmapPortBehavior: {
		Payload: func() any { return &jobs.SubnetUnmapPortRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*jobs.SubnetUnmapPortRequest)

			cmd.Flags().StringVarP(&p.SubnetID, "subnet-id", "i", "", "subnet-id (required)")
			cmd.Flags().StringVarP(&p.Protocol, "protocol", "p", "", "protocol (required)")
			cmd.Flags().StringVarP(&p.SourceIP, "source-ip", "s", "", "source IP address (required)")
			cmd.Flags().StringVarP(&p.SourcePort, "source-port", "o", "", "source port (required)")
			cmd.Flags().StringVarP(&p.DestIP, "dest-ip", "D", "", "destination IP address (required)")
			cmd.Flags().StringVarP(&p.DestPort, "dest-port", "n", "", "destination port (required)")
			_ = cmd.MarkFlagRequired("subnet-id")
			_ = cmd.MarkFlagRequired("protocol")
			_ = cmd.MarkFlagRequired("source-ip")
			_ = cmd.MarkFlagRequired("source-port")
			_ = cmd.MarkFlagRequired("dest-ip")
			_ = cmd.MarkFlagRequired("dest-port")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*jobs.SubnetUnmapPortRequest)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}
			return dmsClient.SubnetUnmapPort(ctx, *req, opts.MsgOpts...)
		},
		Action: bInvoke,
		Short:  "Unmap a port",
		Long: `Invokes the /dms/node/subnet/unmap-port behavior on an actor

This behavior removes a port mapping.

Examples:
  nunet actor cmd --context user /dms/node/subnet/unmap-port --subnet-id <subnet_id> --protocol <protocol> --source-ip <source_ip> --source-port <source_port> --dest-ip <dest_ip> --dest-port <dest_port>`,
	},

	behaviors.SubnetDNSRemoveRecordBehavior: {
		Payload: func() any { return &jobs.SubnetDNSRemoveRecordRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*jobs.SubnetDNSRemoveRecordRequest)

			cmd.Flags().StringVarP(&p.SubnetID, "subnet-id", "s", "", "subnet ID (required)")
			cmd.Flags().StringVarP(&p.DomainName, "domain-name", "n", "", "A record name (required)")
			_ = cmd.MarkFlagRequired("subnet-id")
			_ = cmd.MarkFlagRequired("name")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*jobs.SubnetDNSRemoveRecordRequest)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}

			return dmsClient.SubnetDNSRemoveRecord(ctx, *req, opts.MsgOpts...)
		},
		Action: bInvoke,
		Short:  "Remove a DNS record",
		Long: `Invokes the /dms/node/subnet/dns/remove-record behavior on an actor

This behavior removes a DNS record from the local resolver.

Examples:

  nunet actor cmd --context user /dms/node/subnet/dns/remove-record --subnet-id <subnet_id> --name <record_name>`,
	},

	behaviors.ResourcesAllocatedBehavior: {
		Action: bInvoke,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.ResourcesAllocated(ctx, opts.MsgOpts...)
		},
		Short: "Get allocated resources",
		Long: `Invokes the /dms/node/resources/allocated behavior on an actor

This behavior retrieves the resources allocated by the node. The resources include CPU, RAM, GPU and disk space.
The returned units are in Hz for CPU clock speed, bytes for RAM, VRAM and disk space.

Examples:
	  nunet actor cmd --context user /dms/node/resources/allocated`,
	},

	behaviors.ResourcesFreeBehavior: {
		Action: bInvoke,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.ResourcesFree(ctx, opts.MsgOpts...)
		},
		Short: "Get free resources",
		Long: `Invokes the /dms/node/resources/free behavior on an actor

This behavior retrieves the free resources available on the node. The resources include CPU, RAM, GPU and disk space.
The returned units are in Hz for CPU clock speed, bytes for RAM, VRAM and disk space.

Examples:
	  nunet actor cmd --context user /dms/node/resources/free`,
	},

	behaviors.ResourcesOnboardedBehavior: {
		Action: bInvoke,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.ResourcesOnboarded(ctx, opts.MsgOpts...)
		},
		Short: "Get onboarded resources",
		Long: `Invokes the /dms/node/resources/onboarded behavior on an actor

This behavior retrieves the resources onboarded to the node. The resources include CPU, RAM, GPU and disk space.
The returned units are in Hz for CPU clock speed, bytes for RAM, VRAM and disk space.

Examples:
	  nunet actor cmd --context user /dms/node/resources/onboarded`,
	},
	behaviors.AllocationsListBehavior: {
		Action: bInvoke,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.AllocationsList(ctx, opts.MsgOpts...)
		},
		Short: "List allocations",
		Long: `Invokes the /dms/node/allocations/list behavior on an actor

This behavior retrieves information about all running allocations within your onboarded DMS.
The information includes allocation ID, status, executor type, container ID, resources, and port mappings.

Examples:
	  nunet actor cmd --context user /dms/node/allocations/list`,
	},
	behaviors.LoggerConfigBehavior: {
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
		},
		PreRunFn: func(cmd *cobra.Command, opts actorCmdOptions) error {
			p, ok := opts.Payload.(*node.LoggerConfigRequest)
			if !ok {
				return fmt.Errorf("failed to decode payload")
			}
			flag := cmd.Flags().Lookup("enable-elastic")
			if flag != nil && flag.Changed {
				val, err := strconv.ParseBool(flag.Value.String())
				if err != nil {
					return fmt.Errorf("invalid value for --enable-elastic: %v", err)
				}
				p.ElasticEnabled = &val
			}
			return nil
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*node.LoggerConfigRequest)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}
			return dmsClient.LoggerConfig(ctx, *req, opts.MsgOpts...)
		},
		Action: bInvoke,
		Short:  "Adjust logger settings",
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
	behaviors.HardwareSpecBehavior: {
		Action: bInvoke,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.HardwareSpec(ctx, opts.MsgOpts...)
		},
		Short: "Get hardware specifications",
		Long: `Invokes the /dms/node/hardware/spec behavior on an actor

This behavior retrieves the hardware specifications of the system.

Examples:

	nunet actor cmd --context user /dms/node/hardware/spec`,
	},
	behaviors.HardwareUsageBehavior: {
		Action: bInvoke,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.HardwareUsage(ctx, opts.MsgOpts...)
		},
		Short: "Get hardware usage",
		Long: `Invokes the /dms/node/hardware/usage behavior on an actor

This behavior retrieves the hardware usage of the system.

Examples:

	nunet actor cmd --context user /dms/node/hardware/usage`,
	},
	behaviors.CapListBehavior: {
		Action:  bInvoke,
		Payload: func() any { return &node.CapListRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.CapListRequest)
			cmd.Flags().StringVarP(&p.Context, "context", "c", "", "context name")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*node.CapListRequest)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}
			return dmsClient.CapList(ctx, *req, opts.MsgOpts...)
		},
		Short: "List capabilities",
		Long: `Invokes the /dms/cap/list behavior on an actor

This behavior retrieves a list of capabilities available on the node.

Examples:
  nunet actor cmd --context user /dms/cap/list`,
	},
	behaviors.CapAnchorBehavior: {
		Action:  bInvoke,
		Payload: func() any { return &CapAnchorRequestCmd{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*CapAnchorRequestCmd)
			cmd.Flags().BoolVarP(&p.Root, "root", "", false, "add root anchor")
			cmd.Flags().BoolVarP(&p.Require, "require", "", false, "add require anchor")
			cmd.Flags().BoolVarP(&p.Provide, "provide", "", false, "add provide anchor")
			cmd.Flags().BoolVarP(&p.Revoke, "revoke", "", false, "add revoke anchor")
			cmd.MarkFlagsOneRequired("root", "require", "provide", "revoke")
			cmd.MarkFlagsMutuallyExclusive("root", "require", "provide", "revoke")
		},
		Args: cobra.ExactArgs(1),
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*CapAnchorRequestCmd)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}

			if len(opts.Args) == 1 {
				req.Data = opts.Args[0]
			}

			request := &node.CapAnchorRequest{
				Require: ucan.TokenList{
					Tokens: []*ucan.Token{},
				},
				Provide: ucan.TokenList{
					Tokens: []*ucan.Token{},
				},
				Revoke: ucan.TokenList{
					Tokens: []*ucan.Token{},
				},
				Root: []did.DID{},
			}
			switch {
			case req.Root:
				root, err := did.FromString(req.Data)
				if err != nil {
					return nil, err
				}
				request.Root = append(request.Root, root)

			case req.Require:
				var token ucan.Token
				if err := json.Unmarshal([]byte(req.Data), &token); err != nil {
					return nil, err
				}
				request.Require.Tokens = append(request.Require.Tokens, &token)

			case req.Provide:
				var token ucan.Token
				if err := json.Unmarshal([]byte(req.Data), &token); err != nil {
					return nil, err
				}
				request.Provide.Tokens = append(request.Provide.Tokens, &token)

			case req.Revoke:
				var token ucan.Token
				if err := json.Unmarshal([]byte(req.Data), &token); err != nil {
					return nil, err
				}

				request.Revoke.Tokens = append(request.Revoke.Tokens, &token)
			}

			return dmsClient.CapAnchor(ctx, *request, opts.MsgOpts...)
		},
		Short: "Add capability anchors",
		Long: `Invokes the /dms/cap/anchor behavior on an actor

This behavior anchors capabilities on the node.

Examples:

  nunet actor cmd --context user /dms/cap/anchor --root did
  nunet actor cmd --context user /dms/cap/anchor --require token
  nunet actor cmd --context user /dms/cap/anchor --provide token
  nunet actor cmd --context user /dms/cap/anchor --revoke token`,
	},
}
