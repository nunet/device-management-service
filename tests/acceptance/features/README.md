# Acceptance Test Features Overview

This document outlines the defined features and scenarios used for acceptance testing of the NuNet platform, with a particular focus on the Device Management Service (DMS). It emphasizes user-facing behavior and system-level interactions across a range of operational contexts.

**Notes:**
- No features or scenarios will be included for running the DMS process, enabling the service, or configuring its environment.
- The stakeholder categories used in the matrix are defined [here](https://gitlab.com/nunet/architecture/-/blob/develop/mapping/stakeholder_categories.md).
- The environment definitions can be found [here](https://gitlab.com/nunet/team-processes-and-guidelines/-/tree/main/nunet_test_process_and_environments/README.md).
- To update the test matrix, go to `tests/acceptance/scripts` directory and run `python3 update_test_matrix.py`.
- To view the features and scenarios marked as @wip and to have an estimation of the time to implement them, go to `tests/acceptance/scripts` directory and run `python3 report_wip_features.py`.

---

# Test Matrix

| Feature Name | Description | Stakeholder Category | Environment | Development Status | Test Status | Comments |
|--------------|-------------|----------------------|-------------|--------------------|-------------|----------|
| [Allocation Running on Subnet](#feature-allocation-running-on-subnet) ([.feature](./subnet.feature)) | As a Service Provider I want to launch services on peers And the services can communicate with each other | Service and Resource Providers | Feature or Testnet | Implemented | In progress | Main subnet test implemented; working now on the redeployment case. |
| [Capabilities Management](#feature-capabilities-management) ([.feature](./capabilities_management.feature)) | Grant, revoke, and delegate behavior-level capabilities across users. | Resource Providers and Facilitators | Feature or Testnet | Implemented | Not started |  |
| [Cardano](#feature-cardano) ([.feature](./cardano.feature)) | Launch Cardano nodes. | Tokenomics Enablers | Testnet | Implemented | Not started |  |
| [Contract Management](#feature-contract-management) ([.feature](./contract_management.feature)) | Create and execute a contract. | Service and Resource Providers, Facilitators | Testnet | In progress | Not started |  |
| [Deployment](#feature-deployment) ([.feature](./deployment.feature)) | As a Service Provider I want to deploy my computation on other nodes So that I don't have to use my machine | Service Providers | Feature or Testnet | Implemented | Implemented |  |
| [Deployment Cancellation](#feature-deployment-cancellation) ([.feature](./deployment_cancellation.feature)) | Cancel deployments in various lifecycle stages and error conditions. | Service Providers | Feature or Testnet | Implemented | Not started |  |
| [Deployment Constraints](#feature-deployment-constraints) ([.feature](./deployment_constraints.feature)) | Consider placement constraints such as location, edge proximity, or hardware specs to do a deployment. | Service and Resource Providers | Testnet - Requires infrastructure specific to each scenario. | Implemented | Not started |  |
| [Deployment Geolocated](#feature-deployment-geolocated) ([.feature](./deployment_geolocated.feature)) | As a Service Provider I want deployments with location constraints to run only on eligible nodes So that deployments follow geographic policies | Services Providers | Testnet | Implemented | Not started |  |
| [Deployment Operations](#feature-deployment-operations) ([.feature](./deployment_operations.feature)) | View, manage, and interact with active or failed deployments. | Service Providers | Feature or Testnet | Implemented | Not started |  |
| [Deployment Update](#feature-deployment-update) ([.feature](./deployment_update.feature)) | As a Service Provider I want to dynamically adjust the nodes and allocations of an ensemble So that I can scale my deployment and control its execution with flexibility | Service Providers | Feature or Testnet | In progress | Not started |  |
| [Deployment in an IP Network](#feature-deployment-in-an-ip-network) ([.feature](./deployment_in_an_ip_network.feature)) | As a Service Provider I want to deploy services in two nodes or more And I want the allocations running in each node to be able to communicate with each other | Service and Resource Providers | Feature or Testnet | Implemented | Not started |  |
| [Executor Types](#feature-executor-types) ([.feature](./executor_types.feature)) | Support multiple container/runtime types such as Docker and Firecracker. |  | Feature or Testnet | Docker Implemented | Not started |  |
| [Failure Recovery](#feature-failure-recovery) ([.feature](./failure_recovery.feature)) | Test different strategies and levels of failure recovery for nodes, allocations, ensembles, deployments etc. | Service Providers | Feature or Testnet | Part of it in open MRs | Not started | The full failure recovery is not implemented yet but there are some open MRs with parts of it: [893](https://gitlab.com/nunet/device-management-service/-/merge_requests/893), [874](https://gitlab.com/nunet/device-management-service/-/merge_requests/874), [863](https://gitlab.com/nunet/device-management-service/-/merge_requests/863). |
| [NuNet Documentation AI Agent](#feature-nunet-documentation-ai-agent) ([.feature](./nunet_documentation_ai_agent.feature)) | Usage of AI agent in public and private modes. | End Users | Testnet - Requires infrastructure as DDNS, Proxy, CA, GlusterFS. | In progress | Not started |  |
| [Observability system integration and behavior](#feature-observability-system-integration-and-behavior) ([.feature](./observability.feature)) | The system must provide structured logging, event emission, tracing, and dynamic configuration capabilities through the observability package. | Service and Resource Providers, Facilitators | Testnet | Implemented | Not started |  |
| [Organization Management](#feature-organization-management) ([.feature](./organization_management.feature)) | Join, create, and manage private and public organizations. | Facilitators | Feature | Implemented | Not started |  |
| [Posemesh](#feature-posemesh) ([.feature](./posemesh.feature)) | Launch Posemesh nodes. | Service Providers | Testnet - Requires infrastructure as DDNS, Proxy, CA. | In progress | Not started |  |
| [Resource Management](#feature-resource-management) ([.feature](./resource_management.feature)) | As a Computer Provider I want to manage the computational resources of my machine (onboard/offboard) And ensure deployments running on my machine stay within declared limits And ensure resources are correctly freed once the deployment finishes | Resource Providers | Feature or Testnet | Implemented | Not started |  |
| [Service Deployment](#feature-service-deployment) ([.feature](./service_deployment_modified.feature)) | As a Service Provider I want to launch services on peers So that I can send requests to them and enable inter-service communication | Service Providers | Feature or Testnet | Implemented | In progress | Main scenario implemented in subnet feature file. |
| [Stateful Deployment](#feature-stateful-deployment) ([.feature](./stateful_deployment.feature)) | Handle deployments with data persistence across reboots or environments. | Service Providers | Feature or Testnet - Requires GlusterFS infrastructure. | Implemented | Not started |  |
| [Supervision](#feature-supervision) ([.feature](./supervision.feature)) | Monitor and react to deployment or system failures automatically. | Service and Resource Providers | Feature or Testnet | Not implemented | Not started |  |
| [System and Peer Status](#feature-system-and-peer-status) ([.feature](./system_and_peer_status.feature)) | View DMS status, peer information, and peer connectivity. | Resource Providers | Feature or TestNet | Implemented | Not started |  |
| [Task Deployment](#feature-task-deployment) ([.feature](./task_deployment.feature)) | As a Service Provider I want to deploy tasks across one or multiple peers So that I don't have to use my machine | Service Providers | Feature or Testnet | Implemented | In progress | Main scenario implemented in deployment feature file. |

---

## Feature Coverage

### Feature: Allocation Running on Subnet

*As a Service Provider I want to launch services on peers And the services can communicate with each other*

**Scenarios:**
- 
- Allocations communicating on the same subnet
- Allocations communicating on the same subnet after restart

---

### Feature: Capabilities Management

*Grant, revoke, and delegate behavior-level capabilities across users.*

**Scenarios:**
- Allow authorized users to invoke behaviors
- Anchor a capability with require/provide/root conditions
- Delegate a capability to another user
- Grant a capability to a user
- Prevent unauthorized users from invoking behaviors
- Retrieve a list of available capabilities
- Revoke a previously granted capability

---

### Feature: Cardano

*Launch Cardano nodes.*

**Scenarios:**
- Launch a deployment with a Block Producer Node
- Launch a deployment with a Relay Node

---

### Feature: Contract Management

*Create and execute a contract.*

**Scenarios:**
- Create a contract with detailed resource requirements, payment terms, participant information, and custom terms
- Digitally sign the contract by all relevant parties
- Execute a deployment to analyze the contract's state transitions through to termination
- Recover contract data in the event of a system failure or restart

---

### Feature: Deployment

*As a Service Provider I want to deploy my computation on other nodes So that I don't have to use my machine*

**Scenarios:**
- 
- Remove node in running ensemble
- Retrieve output from execution

---

### Feature: Deployment Cancellation

*Cancel deployments in various lifecycle stages and error conditions.*

**Scenarios:**
- Attempt to cancel a deployment on an unavailable peer
- Attempt to cancel a deployment that has already completed
- Cancel an ongoing deployment

---

### Feature: Deployment Constraints

*Consider placement constraints such as location, edge proximity, or hardware specs to do a deployment.*

**Scenarios:**
- Launch a deployment considering edge constraints
- Launch a deployment with minimum hardware specifications

---

### Feature: Deployment Geolocated

*As a Service Provider I want deployments with location constraints to run only on eligible nodes So that deployments follow geographic policies*

**Scenarios:**
- 
- Deployment fails when one or more nodes have no matching running nodes for geographic constraints
- Deployment is distributed across two eligible nodes based on location constraints
- Deployment is executed only on a node satisfying location constraints

---

### Feature: Deployment Operations

*View, manage, and interact with active or failed deployments.*

**Scenarios:**
- List all deployments
- Restart a failed deployment
- Shutdown/stop a running deployment
- View a deployment’s current status
- View deployment logs
- View deployment manifest

---

### Feature: Deployment Update

*As a Service Provider I want to dynamically adjust the nodes and allocations of an ensemble So that I can scale my deployment and control its execution with flexibility*

**Scenarios:**
- 
- Add a node to the ensemble
- Add an allocation to a node
- Remove a node from the ensemble
- Remove an allocation from a node

---

### Feature: Deployment in an IP Network

*As a Service Provider I want to deploy services in two nodes or more And I want the allocations running in each node to be able to communicate with each other*

**Scenarios:**
- Check communication between allocations
- Check communication between allocations after redeployment
- Check communication over DNS between allocations
- Check communication over DNS between allocations after redeployment
- Connect to a running deployment via SSH

---

### Feature: Executor Types

*Support multiple container/runtime types such as Docker and Firecracker.*

**Scenarios:**
- Deploy using a Docker executor
- Deploy using a Firecracker executor

---

### Feature: Failure Recovery

*Test different strategies and levels of failure recovery for nodes, allocations, ensembles, deployments etc.*

_(Scenarios to be defined)_

---

### Feature: NuNet Documentation AI Agent

*Usage of AI agent in public and private modes.*

**Scenarios:**
- Launch a NuNet Documentation AI agent for public usage
- Launch a NuNet Documentation AI private agent
- Launch a WebUI or API interface for enabling end-user interaction with the model

---

### Feature: Observability system integration and behavior

*The system must provide structured logging, event emission, tracing, and dynamic configuration capabilities through the observability package.*

**Scenarios:**
- 
- Emit a custom event to the internal event bus
- Enable no-op mode to disable all observability
- Log message with labels for routing
- Produce a structured console log
- Set a new Elasticsearch endpoint at runtime
- Shutdown observability gracefully
- Start a distributed trace with a transaction name
- Update the log level dynamically

---

### Feature: Organization Management

*Join, create, and manage private and public organizations.*

**Scenarios:**
- Attempt to join an unauthorized private organization/network
- Create a private organization/network
- Join NuNet’s public network
- Join a private organization/network
- View organizations the peer is part of

---

### Feature: Posemesh

*Launch Posemesh nodes.*

**Scenarios:**
- Launch a deployment with a Dynamic DNS Node
- Launch a deployment with a Relay Node

---

### Feature: Resource Management

*As a Computer Provider I want to manage the computational resources of my machine (onboard/offboard) And ensure deployments running on my machine stay within declared limits And ensure resources are correctly freed once the deployment finishes*

**Scenarios:**
- 
- Deployment allocates the specified resources
- Deployment does not use more CPU than specified in the ensemble
- Deployment does not use more RAM than specified in the ensemble
- Deployment does not use more disk than specified in the ensemble
- Deployment is reject due to insufficient resources
- Deployment releases the allocated resources
- Offboard resources from the network
- Onboard resources to the network
- Retrieve the current hardware usage
- Retrieve the hardware specifications
- Targeting deployment is reject due to insufficient resources

---

### Feature: Service Deployment

*As a Service Provider I want to launch services on peers So that I can send requests to them and enable inter-service communication*

**Scenarios:**
- 
- Deploy a service and send requests from other nodes
- Deploy a service and send requests using DNS addresses
- Redeploy a service and send requests from other nodes
- Redeploy a service and send requests using DNS addresses

---

### Feature: Stateful Deployment

*Handle deployments with data persistence across reboots or environments.*

**Scenarios:**
- Attempt to restore data from an invalid or missing storage path
- Create an external volume (glusterfs)
- Delete an external volume (glusterfs)
- Launch a new deployment restoring from a local volume
- Launch a new deployment restoring from an external volume (glusterfs)
- Persist deployment data to a local volume (disk)
- Persist deployment data to an external volume (glusterfs)

---

### Feature: Supervision

*Monitor and react to deployment or system failures automatically.*

_(Scenarios to be defined)_

---

### Feature: System and Peer Status

*View DMS status, peer information, and peer connectivity.*

**Scenarios:**
- Broadcast a hello message to a topic
- Connect to a peer
- Ensure connected peers list reflects only active connections
- Ping a connected peer
- View DHT connected peers
- View DMS status (version, onboarding status, DMS DID)
- View connected peers
- View self peer information

---

### Feature: Task Deployment

*As a Service Provider I want to deploy tasks across one or multiple peers So that I don't have to use my machine*

**Scenarios:**
- 
- Launch a deployment on a target peer
- Launch a deployment on any available peer in the network
- Launch multiple deployments on different peers
- Launch multiple deployments on the same peer

---

## Next Steps

- Define scenarios that uses Failure Recovery and Supervision.
- Identify other scenarios for DMS acceptance tests and NuNet platform.
- Define features and scenarios for other projects like running external nodes, appliance, LLM use cases. 
- Implement the scenarios prioritizing the most frequently used functionalities as well as newly implemented features and use cases.

---