// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cap

import (
	"fmt"

	"github.com/spf13/cobra"
)

var behaviorsAndCapsHelp = `
The following are the implemented behaviors and their associated capabilities:

/dms/node/peers/ping
- PeerPingBehavior: Ping a peer to check if it is alive.
/dms/node/peers/list
- PeersListBehavior: List peers visible to the node.
/dms/node/peers/self
- PeerSelfBehavior: Get the peer id and listening address of the node.
/dms/node/peers/dht
- PeerDHTBehavior: Get the peers in DHT of the node along with their DHT parameters.
/dms/node/peers/connect
- PeerConnectBehavior: Connect to a peer.
/dms/node/peers/score
- PeerScoreBehavior: Get the libp2p pubsub peer score of peers.

/dms/node/onboarding/onboard
- OnboardBehavior: Onboard the node as a compute provider.
/dms/node/onboarding/offboard
- OffboardBehavior: Offboard the node as a compute provider.
/dms/node/onboarding/status
- OnboardStatusBehavior: Get the onboarding status.

/dms/node/deployment/new
- NewDeploymentBehavior: Start a new deployment on the node.
/dms/node/deployment/list
- DeploymentListBehavior: List all the deployments orchestrated by the node.
/dms/node/deployment/logs
- DeploymentLogsBehavior: Get the logs of a particular deployment.
/dms/node/deployment/status
- DeploymentStatusBehavior: Get the status of a deployment.
/dms/node/deployment/manifest
- DeploymentManifestBehavior: Get the manifest of a deployment.
/dms/node/deployment/info
- DeploymentInfoBehavior: Get comprehensive information about a deployment in a single call.
/dms/node/deployment/shutdown
- DeploymentShutdownBehavior: Shutdown a deployment.

/dms/node/resources/allocated
- ResourcesAllocatedBehavior: Get the amount of resources allocated to Allocations running on the node.
/dms/node/resources/free
- ResourcesFreeBehavior: Get the amount of resources that are free to be allocated on the node.
/dms/node/resources/onboarded
- ResourcesOnboardedBehavior: Get the amount of resources the node is onboarded with.
/dms/node/hardware/spec
- HardwareSpecBehavior: Get the hardware resource specification of the machine.
/dms/node/hardware/usage
- HardwareUsageBehavior: Get the full resource usage on the machine.

/dms/node/logger/config
- LoggerConfigBehavior: Configure the logger/observability config of the node.

/dms/deployment/request
- BidRequestBehavior: Request a bid from the compute provider for a specific ensemble.
/dms/deployment/bid
- BidReplyBehavior: Reply to a bid request from an orchestrator.
/dms/deployment/commit
- CommitDeploymentBehavior: Temporarily commit the resources the provider bid on.
/dms/deployment/allocate
- AllocationDeploymentBehavior: Allocate the resources the provider bid on.
/dms/deployment/revert
- RevertDeploymentBehavior: Revert any commit or allocation done during a deployment.

/dms/cap/list
- CapListBehavior: Get a list of all the capabilities another node had.
/dms/cap/anchor
- CapAnchorBehavior: Anchor capability tokens on another node.

/public/hello
- PublicHelloBehavior: Get a hello message from a node.
/public/status
- PublicStatusBehavior: Get the total resource amount on the machine.
/broadcast/hello
- BroadcastHelloBehavior: Broadcast a hello message and get replies from nodes.

/dms/allocation/start
- AllocationStartBehavior: Start an allocation after a deployment.
/dms/allocation/restart
- AllocationRestartBehavior: Restart an allocation after a deployment has been started.
/dms/actor/healthcheck/register
- RegisterHealthcheckBehavior: Register a new healthcheck mechanism for an allocation.

/dms/allocation/subnet/add-peer
- SubnetAddPeerBehavior: Add a peer to a subnet.
/dms/allocation/subnet/remove-peer
- SubnetRemovePeerBehavior: Remove a peer from a subnet.
/dms/allocation/subnet/accept-peer
- SubnetAcceptPeerBehavior: Accept a peer in a subnet.
/dms/allocation/subnet/map-port
- SubnetMapPortBehavior: Map a port in a subnet.
/dms/allocation/subnet/unmap-port
- SubnetUnmapPortBehavior: Unmap a port in a subnet.
/dms/allocation/subnet/dns/add-records
- SubnetDNSAddRecordsBehavior: Add DNS records to a subnet.
/dms/allocation/subnet/dns/remove-records
- SubnetDNSRemoveRecordsBehavior: Remove a DNS record from a subnet.

/dms/ensemble/<ENSEMBLE_ID>
- EnsembleNamespace: Interact with ensembles on the node.
/dms/ensemble/<ENSEMBLE_ID>/allocation/logs
- AllocationLogsBehavior: Get the logs of an allocation in an ensemble.
/dms/ensemble/<ENSEMBLE_ID>/allocation/shutdown
- AllocationShutdownBehavior: Shutdown an allocation in an ensemble.
/dms/ensemble/<ENSEMBLE_ID>/node/subnet/create
- SubnetCreateBehavior: Create a new subnet for an ensemble.
/dms/ensemble/<ENSEMBLE_ID>/node/subnet/destroy
- SubnetDestroyBehavior: Destroy a subnet for an ensemble.
`

func newHelpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "help",
		Short: "Description of Behaviors and Associated Capabilities",
		Long: "List of all implemented behaviors and their associated capabilities\n" +
			"with a short description of each behavior and intended use.\n",
		Args: cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.OutOrStdout(), behaviorsAndCapsHelp)
		},
	}
	return cmd
}
