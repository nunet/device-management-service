# Computing in the NuNet Network

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

**Table of Contents**

- [Computing in the NuNet Network](#computing-in-the-nunet-network)
  - [Ensemble Orchestration](#ensemble-orchestration)
    - [Compute Ensembles](#compute-ensembles)
    - [Ensemble Specification](#ensemble-specification)
    - [Ensemble Constraints](#ensemble-constraints)
    - [Ensemble Deployment](#ensemble-deployment)
    - [Ensemble Supervision](#ensemble-supervision)
  - [Deploying in the NuNet Network](#deploying-in-the-nunet-network)
    - [Behaviors and Capabilities](#behaviors-and-capabilities)
    - [Deploying in a Private Network](#deploying-in-a-private-network)
    - [Authorizing a Third Party to Vet Users](#authorizing-a-third-party-to-vet-users)
    - [Distributing and Revoking Capability Tokens](#distributing-and-revoking-capability-tokens)
    - [Public Deployment](#public-deployment)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Ensemble Orchestration

In NuNet, compute workloads are structured as compute _ensembles_.
Here, we discuss how an ensemble can be created, deployed, and
supervised in the NuNet network.

### Compute Ensembles

An ensemble is a collection of logical _nodes_ and _allocations_. Nodes
represent the hardware where the compute workloads run. Allocations
are the individual compute jobs that comprise the workload. Each
allocation is assigned to a node, and a node can have multiple
allocations assigned to it.

All allocations in the ensemble are assigned a private IP address in
the 10/8 range and are connected with a virtual private network,
implemented using IP over libp2p. All allocations can reach each other
through the VPN. Allocation IP addresses can be discovered internally
in the ensemble using DNS: each allocation has a name and a DNS name,
which by default is just the allocation name in the `.internal`
domain.

Allocation and Node names within an ensemble must be unique. The
ensemble as a whole has a globally unique ID (a randomn UUID).

### Ensemble Specification

In order to deploy an ensemble, the user must specify its structure
and constraints; this is done with a YAML file encoding the [ensemble configuration data structure](types/ensemble.go); the fields of the
configuration structure are described in detail in this [reference](ensemble_fields_reference.md).

Fundamentally the ensemble configuration has the following structure:

- A map of allocations, mapping allocation names to configuration for individual allocations.
- A map of nodes, mapping node names to configuration for individual nodes.
- A list of edges between nodes, encoding specific logical edge constraints.
- There are additional fields in the data structure which allows us to include ssh keys and scripts in the configuration, as well as supervision strategies policies.

An allocation's configuration has the following structure:

- The type of allocation:
  - Service: A long-running process that should be continuously available and automatically restarted on failure
  - Task: A one-off job that runs to completion and exits
- The executor type (the runtime environment for the allocation):
  - Currently supports: docker, firecracker VMs
  - Future support planned for: WASM and other sandboxed environments
- The resources required to run the allocation, such as memory, cpu cores, gpus, and so on.
- The execution details, which encodes the executor specific configuration of the allocation.
- The DNS name for internal name resolution of the allocation. This can be omitted, in which case the allocation's name becomes the DNS name.
- The list of ssh keys ton drop in the allocation, so that administrators can ssh into the allocation.
- The list of scripts to execute during provisioning, in execution order.
- Finally, the user can also specify the application specific health check to be performede by the supervisor, so that the health of the application can be ascertained and failures detected.

A node's configuration has the following structure:

- The list of allocations that are assigned to the node
- The configuration of mapping public ports to ports in allocations
- The Location constraints for the node
- An optional field for explicitly specifying the peer on which the node should be assigned, allowing users and organizations to bring their own nodes into the mix, for instance for hosting sensitive data.

In the near future, we also plan to support directly parsing
kubernetes job description files. We also plan to provide a
declarative format for specifying large ensembles so that it is
possible to succinctly describe a 10k GPU ensemble for training an LLM
and so on.

### Ensemble Constraints

It is worth reiterating that ensembles carry with the constraints, as
specified by the user. This allows the user to have finegrained
control of their ensemble deployment and ensure that certain
requirements are met.

In DMS v0.5 we support the following constraints:

- Resources for an allocation, such as memory, core count, gpu details, and so on.
- Location for nodes; the user can specify the region, city, etc all the way to choosing a particular ISP. Location constraints can also be negative, so that a node will not be deployed in certain locations e.g. because of regulatory considerations such as GPDR.
- Edge Constraints, which specify the relationship between nodes in the allocation in terms of available bandwidth and round trip time.

In subsequent releases we plan to add additional constraints
(e.g. existence of a contract, price range, explicit datacenter
placement, energy sources and so on) and generalize the constraint
expression language as graphs.

### Ensemble Deployment

Given an ensemble specification, the core functionality of the NuNet
network is to find and assign peers to nodes that satisfies the
constraints of the ensemble. The system treats the deployment as a
constraint satisfaction problem over permutations of available peers
(compute nodes) on which the user is authorized to deploy. The process
of deploying an ensemble is called _orchestration_. In the following
we summarize how deployment orchestration is performed.

<p align="center">
  <img src="https://gitlab.com/nunet/device-management-service/-/raw/main/dms/jobs/specs/diagrams/ensemble_deployment.png?ref_type=heads&inline=true" width="100%" alt="Ensemble Deployment Sequence Diagram">
</p>

Ensemble deployment is initiated with a user invoking the
`/dms/node/deployment/new` behavior on the node which is willing to
run an orchestrator for them; this can be just the user's private DMS
running on his laptop. The node accepting the invocation creates the
orchestrator actor inside its process space, initiates the deployment
orchestration, and return to the user the ensemble identifier. The
user can use this identifier to poll the status of the deployment and
control of the ensemble through the orchestrator actor. The user also
specifies a timeout on how long the deployment process should take
before declaring failure. This is simply the expiration on the message
that invokes `/dms/node/deployment/new`.

The orchestrator then proceeds to request _bids_ for each node in the
ensemble. This is accomplished by broadcasting a message to the
`/dms/deployment/request` behavior in the `/nunet/deployment`
broadcast topic. The deployment request contains a mapping of node
names in the ensemble, together with their aggregate (for all
allocations to be assigned in the node) resource constraints, together
with location and other constraints that can restrict the search
space.

In order for this to proceed, the orchestrator must have the
appropriate capabilities; only provider nodes that accept the user's
capabilities will respond to the broadcast message. The response to
the bid request is a bid for a node in the ensemble, by sending a
message to the `/dms/deployment/bid` behavior in the
orchestrator. This also implies that the nodes that submit such bids
must have appropriate capabilities accepted by the orchestrator.

Given the appropriate capabilities, the orchestrator collects bids
until it has a sufficient number of bids or a timeout that ensures
prompt progress in the deployment. If the orchestrator doesn't have
bids for all nodes, then it rebroadcasts its bid request, excluding
peers that have already submitted a bid. This continues until there
are bids for all nodes or the deployment times out, at which point a
deployment failure is declared.

Note that in the case of node pinning, where a specific peer is
assigned to an ensemble node in advance (ie when a user brings their
own nodes into the ensemble), bid requests are not broadcast but
rather directly invoked on the peer.

Next, the orchestrator generates permutations of assignments of peers
to nodes and evaluates the constraints. Some constraints can be
directly rejected without measurement, for instance round trip latency
constraints can be rejected by using speed of light calculations that
provide a lower bound on physically realizable latency. We plan to do
the same with bandwidth constraints, given the node measured link
capacity and the throughput bound equation that governs TCP's
behavior given bottleneck bandwidth and RTT.

Once a candidate assignment is deemed viable, the orchestrator
proceeds to measure specific constraints for satisfiability. This
involves measuring round trip time and bandwidth between node pairs,
and is accomplished by invoking the `/dms/deployment/constraint/edge`
behavior.

If a candidate assignment satisfies the constraints, the orchestrator
proceeds with committing and provisioning the deployment. This is done
with a two phase commit process: first the orchestrator sends a commit
message to all peers to ensure that the resources are still available
(nodes don't lock resources when submitting a bid), by invoking the
`/dms/deployment/commit` behavior. If any node fails to commit, the
candidate deployment is reverted and the orchestrator starts anew;
revert happens with the `/dms/deployment/revert` behavior.

If all nodes successfully commit, the orchestrator proceeds to
_provision_ the deployment by sending allocation details to the
relevant nodes and creating the VPN. This is initiated by invoking the
`/dms/deployment/allocate` behavior on the provider nodes, which
creates a new allocation actor. Subsequently, the orchestrator assigns
IP addresses to allocations and creates the VPN (what we call the
subnet) by invoking the appropriate behaviors on the allocation
actors, and then starts the allocations. Once all nodes provision,
the deployment is now considered running and enters supervision.

The deployment will keep running until the user shuts it down, as long
as the user's agreement with the provider is active; in the near
future we will also support explicitly specifying durations for
running ensembles, and the ability to modify running ensembles in order
to support mechanisms like auto scaling.

<p align="center">
  <img src="https://gitlab.com/nunet/device-management-service/-/raw/main/dms/jobs/specs/diagrams/ensemble_components.png?ref_type=heads&inline=true" width="50%" alt="Ensemble deployment component diagram: shows ensemble configuration on actual network after deployment is successfully finished.">
</p>

### Ensemble Supervision

TODO

## Deploying in the NuNet Network

In order to discuss authorization flow for deployment in the NuNet
network, we need to distinguish certain actors in the system in the
course of an ensembles lifetime.

Specifically, we introduce the following notation:

- Let's call `U`, the user as an actor.
- Let's call `O` the orchestrator, which is an actor living inside a DMS instance (node) for which the user is authorized to initiate a deployment. We call the node where the orchestrator runs `N_o`. Note that the DID of the orchestrator actor will be the same as the DID of the node on which it runs, but it will have an ephemeral actor ID.
- Let's call `P_i` the set of compute providers that are willing to accept deployment requests from `U`.
- Let's call `N_{P_i,j}` the DMS nodes controlled by the providers that are willing to accept deployments from users.
- And finally let's call `A_i` the allocation actor for each running allocation. The DID of each allocation actor will be the same as the DID of the node on which the allocation is running, but it will have an ephemeral actor ID.

Also note that we have certain identifiers pertaining to these actors; let's define the following notation:

- `DID(x)` is the DID of actor `x`; in general this is the DID that identifies the node on which the actor is running.
- `ID(x)` is the ID of actor `x`; this is generally ephemeral, except for node root actors which have persistent identities matching their DID.
- `Peer(x)` is the peer ID of a node/actor `x`.
- `Root(x)` is the DID of the root anchor of trust for the node/actor `x`.

### Behaviors and Capabilities

Using the notation above we can enumerate the behavior namespaces and requisite
capabilities for deployment of an ensemble:

- Invocations from `U` to `N_o` are in the `/dms/node/deployment` namespace
- Invocations from `O` to `N_{P_i,j}` for deployment bids:
  - broadcast `/dms/deployment/request` via the `/nunet/deployment` topic
  - unicast `/dms/deployment/request` for pinned ensemble nodes
- Messages from `N_{P_i,j}` to `O`:
  - `/dms/deployment/bid` as the reply to a bid request
- Invocations from `O` to `N_{P_i,j}` for deployment control are in the `/dms/deployment` namespace.
- Invocations from `O` to `A_i` are in the `/dms/allocation` namespace and are dynamically granted programmatically.
- Invocations from `O` to `N_{P_i,j}` for allocation control are in the dynamic `/dms/ensemble/<ensemble-id>` namespace and are dynamically granted programatically.

This creates the following structure:

- `U` must be authorized with `/dms/node/deployment` capability in `N_o`
- `N_o` must be authorized with `/dms/deployment` capability in `N_{P_i,j}` so that the orchestrator can make the appropriate invocations.
- `N_{P_i,j}` must be authorized with `/dms/deployment/bid` capability on `N_o` so that it can submit bids to the orchestrator.

Note that the decentralized structure and fine grained capability model of the
NuActor system allows for very tight access control. This ensures
that:

- Orchestrators can only run on DMS instances where the user is authorized to initiate deployment.
- Bid requests will only be accepted by provider DMS instances where the user is authorized to deploy.
- Bids will only be accepted by provider DMS instances whom the user has authorized.

In the following we examine common functional scenarios on how to set
up the system so that deployments are properly authorized.

### Deploying in a Private Network

TODO

### Authorizing a Third Party to Vet Users

TODO

### Distributing and Revoking Capability Tokens

TODO

### Public Deployment

TODO
