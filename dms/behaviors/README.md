# DMS Capabilities and Behaviors

The DMS Behaviors are a set of functionalities associated with a hierarchical namespace that the DMS performs if requested by an actor that has the necessary capabilities. Since capabilities are hierarchical, they can be applied either as exact match to a behavior or be implied by a top level capability. For example, the `/dms/node/peers/ping` behavior can be accessed by an actor with the `/dms/node/peers/ping` capability when specific or the `/dms/node` capability that implies everything below it.

## Node Capabilities
- **DMS Node Namespace**: `/dms/node`
    - Description: Everything related to the management of a DMS node. Any capability that is implied by this namespace will be able to access all the behaviors below it. These should only be allowed to the controller/user of the DMS. Normally, it would be done by anchoring the DID of the controller user to the root anchor of the dms which allows the controller unlimited root access.

For a fine grained control, the following capabilities can be used:

### Peer Capabilities

The following Peer capabilities are directly associated with peer behaviors under the same path. They allow the controller of a DMS to request it to perform these actions. For example, when the PeerSelf behavior is invoked by the controller, it's the peer address of the DMS node that is returned.

- **PeerPingBehavior**: `/dms/node/peers/ping`
    - Description: Ping a peer to check if it is alive.
- **PeersListBehavior**: `/dms/node/peers/list`
    - Description: List peers visible to the node.
- **PeerSelfBehavior**: `/dms/node/peers/self`
    - Description: Get the peer id and listening address of the node.
- **PeerDHTBehavior**: `/dms/node/peers/dht`
    - Description: Get the peers in DHT of the node along with their DHT parameters.
- **PeerConnectBehavior**: `/dms/node/peers/connect`
    - Description: Connect to a peer.
- **PeerScoreBehavior**: `/dms/node/peers/score`
    - Description: Get the libp2p pubsub peer score of peers

### Onboarding Capabilities

The following capabilities deal with onboarding the DMS node as compute provider on the network. Onboarding a node involves setting a specific amount of compute resources for the node to allocate for incoming jobs.
- **OnboardBehavior**: `/dms/node/onboarding/onboard`
    - Description: Onboard the node as a compute provider.
- **OffboardBehavior**: `/dms/node/onboarding/offboard`
    - Description: Offboard the node as a compute provider.
- **OnboardStatusBehavior**: `/dms/node/onboarding/status`
    - Description: Get the onboarding status. Whether the node is onboarded or not and errors if any.

### Node Deployment Capabilities
- **NewDeploymentBehavior**: `/dms/node/deployment/new`
    - Description: This node behavior is invoked by the controller to start a new deployment on the node. It takes an ensemble config as input and returns the deployment id.
- **DeploymentListBehavior**: `/dms/node/deployment/list`
    - Description: List all the deployments orchestrated by the node.
- **DeploymentLogsBehavior**: `/dms/node/deployment/logs`
    - Description: Get the logs of a particular deployment.
- **DeploymentStatusBehavior**: `/dms/node/deployment/status`
    - Description: Get the status of a deployment.
- **DeploymentManifestBehavior**: `/dms/node/deployment/manifest`
    - Description: Get the manifest of a deployment.
- **DeploymentShutdownBehavior**: `/dms/node/deployment/shutdown`
    - Description: Shutdown a deployment.

### Resource Capabilities
- **ResourcesAllocatedBehavior**: `/dms/node/resources/allocated`
    - Description: The behavior returns the amount of resources allocated to Allocations running on the node. Allocated resources should always be less than or equal to the onboarded resources
- **ResourcesFreeBehavior**: `/dms/node/resources/free`
    - Description: The behavior returns the amount of resources that are free to be allocated on the node. Free resources should always be less than or equal to the onboarded resources
- **ResourcesOnboardedBehavior**: `/dms/node/resources/onboarded`
    - Description: The behavior returns the amount of resources the node is onboarded with.
- **HardwareSpecBehavior**: `/dms/node/hardware/spec`
    - Description: The behavior returns the hardware resource specification of the machine.
- **HardwareUsageBehavior**: `/dms/node/hardware/usage`
    - Description: The behavior returns the full resource usage on the machine including usage by other processes.

### Logger Capabilities
- **LoggerConfigBehavior**: `/dms/node/logger/config`
    - Description: Configure the logger/observability config of the node.

### Deployment Capabilities

The following capabilities are associated with the deployment of jobs on a DMS node. These capabilities and behaviors allow the controller to deploy services on nodes, list the deployed ensembles, get the logs from deployed allocations, get the status of the deployment, get the manifest of the deployment, and shutting down the deployment.

During regular use, it's recommended that compute providers delegate the `/dms/deployment` capability to orchestrators.

- **BidRequestBehavior**: `/dms/deployment/request`
    - Description: The behavior and capability that will need to be invoked by an orchestrator and delegated from a compute provider to the orchestrator. It allows the orchestrator to request a bid from the compute provider for a specific ensemble.
- **BidReplyBehavior**: `/dms/deployment/bid`
    - Description: The behavior and capability that will need to be invoked by a compute provider and delegated from an orchestrator to the compute provider. It allows the compute provider to reply to a bid request from an orchestrator.
- **CommitDeploymentBehavior**: `/dms/deployment/commit`
    - Description: The associated behavior with this capability allows an orchestrator to temporarily commit the resources the provider bid on until full allocation.
- **AllocationDeploymentBehavior**: `/dms/deployment/allocate`
    - Description: The associated behavior with this capability allows an orchestrator to allocate the resources the provider bid on after having committed it temporarily.
- **RevertDeploymentBehavior**: `/dms/deployment/revert`
    - Description: The associated behavior with this capability allows an orchestrator to revert any commit or allocation done during a deployment.

### Capability Capabilities

Capability behaviors allow remote nodes to configure capability tokens on the node. The receiver node needs to have delegated the `/dms/cap` capability to the invoking node.

- **CapListBehavior**: `/dms/cap/list`
    - Description: The behavior and associated capability allow getting a list of all the capabilities another node had. The capability should be delegated to the node that needs to get the list of capabilities.
- **CapAnchorBehavior**: `/dms/cap/anchor`
    - Description: Allows anchoring capability tokens on another node.

### Public Capabilities

The following capabilities are associated with public behaviors that can be invoked by any actor on the network. These capabilities are normally granted to all actors on the network that are KYC'd by NuNet with the `/public` capability. However, some nodes may choose to restrict these capabilities to specific actors and may not reply to invocations.

- **PublicHelloBehavior**: `/public/hello`
    - Description: A public hello behavior where any actor can invoke it on a specific node/actor and get a hello message back if public capability has been granted.
- **PublicStatusBehavior**: `/public/status`
    - Description: Invoking this behavior on a node will cause it to reply with its total resource amount it has on the machine along with an error message if any.
- **BroadcastHelloBehavior**: `/broadcast/hello`
    - Description: A public hello broadcast in which any actor/node that receives it will reply with a hello message along with its DID.

### Allocation Capabilities

Allocation capabilities are normally granted to orchestrators once a deployment starts running to allow the orcestrator to manage the allocations it deployed. These capabilities are normally granted temporarily since the allocations themselves are ephemeral and live only for the duration of the deployment.

- **AllocationStartBehavior**: `/dms/allocation/start`
    - Description: Start an allocation after a deployment.
- **AllocationRestartBehavior**: `/dms/allocation/restart`
    - Description: Restart an allocation after a deployment has been started.
- **RegisterHealthcheckBehavior**: `/dms/actor/healthcheck/register`
    - Description: Register a new healthcheck mechanism for an allocation.

### Subnet Capabilities

These too are associate with allocations and are granted to orchestrators once a deployment starts running. These capabilities allow the orchestrator to manage the subnet of the allocations it deployed in order to allow allocations to communicate with an ip layer on top of the p2p network.

- **SubnetAddPeerBehavior**: `/dms/allocation/subnet/add-peer`
    - Description: Add a peer to a subnet.
- **SubnetRemovePeerBehavior**: `/dms/allocation/subnet/remove-peer`
    - Description: Remove a peer from a subnet.
- **SubnetAcceptPeerBehavior**: `/dms/allocation/subnet/accept-peer`
    - Description: Accept a peer in a subnet.
- **SubnetMapPortBehavior**: `/dms/allocation/subnet/map-port`
    - Description: Map a port in a subnet. The mapping will be between the subnet ip and the port on the executor.
- **SubnetUnmapPortBehavior**: `/dms/allocation/subnet/unmap-port`
    - Description: Unmap a port in a subnet.
- **SubnetDNSAddRecordsBehavior**: `/dms/allocation/subnet/dns/add-records`
    - Description: Add DNS records to a subnet. Normally these records identify the allocations within the subnet. Each Allocation can have a dns_name parameter that can be used to identify the allocation but if not provided, the allocation name will be used instead. DNS names have a .internal suffix but can be used without them since the resolver within the executor will add it automatically if it supports it.
- **SubnetDNSRemoveRecordsBehavior**: `/dms/allocation/subnet/dns/remove-records`
    - Description: Remove DNS records from a subnet.

### Ensemble Capabilities

Allocation Ensemble Capabilities are dynamic type namespaces that are created when an ensemble is deployed on a node. These capabilities are granted to orchestrators once a deployment starts running to allow the orchestrator to manage the allocations it deployed. These capabilities are normally granted temporarily and live only as long as the ensemble.

- **EnsembleNamespace**: `/dms/ensemble/%s`
    - Description: A dynamic namespace that allows the controller to interact with ensembles on the node. The `%s` will be replaced by the ensemble id once the deployment is running.
- **AllocationLogsBehavior**: `/dms/ensemble/%s/allocation/logs`
    - Description: Get the logs of an allocation in an ensemble.
- **AllocationShutdownBehavior**: `/dms/ensemble/%s/allocation/shutdown`
    - Description: Shutdown an allocation in an ensemble.
- **SubnetCreateBehavior**:
    - DynamicTemplate: `/dms/ensemble/%s/node/subnet/create`
    - Static: `/dms/node/subnet/create`
    - Description: Create a new subnet for an ensemble. This request is supposed to be received by the node of the compute provider and created for the allocations it creates for the ensemble.
- **SubnetDestroyBehavior**:
    - DynamicTemplate: `/dms/ensemble/%s/node/subnet/destroy`
    - Static: `/dms/node/subnet/destroy`
    - Description: Destroy a subnet for an ensemble. This request is supposed to be received by the node of the compute provider.
