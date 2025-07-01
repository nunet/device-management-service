# Acceptance Test Features Overview

This document outlines the defined features and scenarios used for acceptance testing of the NuNet platform, with a particular focus on the Device Management Service (DMS). It emphasizes user-facing behavior and system-level interactions across a range of operational contexts.

**Notes:**
- No features or scenarios will be included for running the DMS process, enabling the service, or configuring its environment.
- The stakeholder categories used in the matrix are defined [here](https://gitlab.com/nunet/architecture/-/blob/develop/mapping/stakeholder_categories.md).
- The environment definitions can be found [here](https://gitlab.com/nunet/team-processes-and-guidelines/-/tree/main/nunet_test_process_and_environments/README.md).
- To update the test matrix, go to `tests/acceptance/scripts` directory and run `python3 update_test_matrix.py`.
- To view the features and scenarios marked as @wip, go to `tests/acceptance/scripts` directory and run `python3 report_wip_features.py`.

---

# Test Matrix

| Feature Name | Description | Stakeholder Category | Environment | Development Status | Test Status | Comments |
|--------------|-------------|----------------------|-------------|--------------------|-------------|----------|
| [Allocation Types](#feature-allocation-types) ([.feature](./allocation_types.feature)) | Launch deployments with task- or service-based resource allocation. | Service and Resource Providers | Feature or Testnet | Implemented | Not started |  |
| [Capabilities Management](#feature-capabilities-management) ([.feature](./capabilities_management.feature)) | Grant, revoke, and delegate behavior-level capabilities across users. | Resource Providers and Facilitators | Feature or Testnet | Implemented | Not started |  |
| [Contract Management](#feature-contract-management) ([.feature](./contract_management.feature)) | Create and execute a contract. | Service and Resource Providers, Facilitators | Testnet | In progress | Not started |  |
| [Deployment](#feature-deployment) ([.feature](./deployment.feature)) | As a Service Provider I want to deploy my computation on other nodes So that I don't have to use my machine | Service Providers | Feature or Testnet | Implemented | In refactoring | This test is fully implemented but being refactored. |
| [Deployment Cancellation](#feature-deployment-cancellation) ([.feature](./deployment_cancellation.feature)) | Cancel deployments in various lifecycle stages and error conditions. | Service Providers | Feature or Testnet | Implemented | Not started |  |
| [Deployment Constraints](#feature-deployment-constraints) ([.feature](./deployment_constraints.feature)) | Consider placement constraints such as location, edge proximity, or hardware specs to do a deployment. | Service and Resource Providers | Testnet - Requires infrastructure specific to each scenario. | Implemented | Not started |  |
| [Deployment Launch](#feature-deployment-launch) ([.feature](./deployment_launch.feature)) | Enables launching single or multiple deployments across peers. | Service and Resource Providers | Feature or Testnet | Implemented | In progress |  |
| [Deployment Operations](#feature-deployment-operations) ([.feature](./deployment_operations.feature)) | View, manage, and interact with active or failed deployments. | Service Providers | Feature or Testnet | Implemented | Not started |  |
| [Deployment in an IP Network](#feature-deployment-in-an-ip-network) ([.feature](./deployment_in_an_ip_network.feature)) | Support deployments that communicate via IP networking. | Service and Resource Providers | Feature or Testnet | Implemented | Not started |  |
| [Executor Types](#feature-executor-types) ([.feature](./executor_types.feature)) | Support multiple container/runtime types such as Docker and Firecracker. |  | Feature or Testnet | Docker Implemented | Not started |  |
| [Failure Recovery](#feature-failure-recovery) ([.feature](./failure_recovery.feature)) | Test different strategies and levels of failure recovery for nodes, allocations, ensembles, deployments etc. | Service Providers | Feature or Testnet | Part of it in open MRs | Not started | The full failure recovery is not implemented yet but there are some open MRs with parts of it: [893](https://gitlab.com/nunet/device-management-service/-/merge_requests/893), [874](https://gitlab.com/nunet/device-management-service/-/merge_requests/874), [863](https://gitlab.com/nunet/device-management-service/-/merge_requests/863). |
| [NuNet Documentation AI Agent](#feature-nunet-documentation-ai-agent) ([.feature](./nunet_documentation_ai_agent.feature)) | Usage of AI agent in public and private modes. | End Users | Testnet - Requires infrastructure as DDNS, Proxy, CA, GlusterFS. | In progress | Not started |  |
| [Observability](#feature-observability) ([.feature](./observability.feature)) | Provide logs, metrics, and tracing for deployed services and system behavior. | Service and Resource Providers, Facilitators | Testnet - Requires infrastructure for storing and visualizing the generated data. | Implemented | Not started |  |
| [Organization Management](#feature-organization-management) ([.feature](./organization_management.feature)) | Join, create, and manage private and public organizations. | Facilitators | Feature | Implemented | Not started |  |
| [Posemesh](#feature-posemesh) ([.feature](./posemesh.feature)) | Launch Posemesh nodes. | Service Providers | Testnet - Requires infrastructure as DDNS, Proxy, CA. | In progress | Not started |  |
| [Stateful Deployment](#feature-stateful-deployment) ([.feature](./stateful_deployment.feature)) | Handle deployments with data persistence across reboots or environments. | Service Providers | Feature or Testnet - Requires GlusterFS infrastructure. | Implemented | Not started |  |
| [Supervision](#feature-supervision) ([.feature](./supervision.feature)) | Monitor and react to deployment or system failures automatically. | Service and Resource Providers | Feature or Testnet | Not implemented | Not started |  |
| [System and Peer Status](#feature-system-and-peer-status) ([.feature](./system_and_peer_status.feature)) | View DMS status, peer information, and peer connectivity. | Resource Providers | Feature or TestNet | Implemented | Not started |  |

---

## Feature Coverage

### Feature: Allocation Types

*Launch deployments with task- or service-based resource allocation.*

**Scenarios:**
- Launch a deployment with a service-type allocation (long term allocation)
- Launch a deployment with a task-type allocation (temporary allocation)

---

### Feature: Capabilities Management

*Grant, revoke, and delegate behavior-level capabilities across users.*

**Scenarios:**
- Allow authorized users to invoke behaviors
- Anchor a capability with require/provide/root conditions
- Delegate a capability to another user
- Grant a capability to a user
- Prevent unauthorized users from invoking behaviors
- Revoke a previously granted capability

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
- Launch a deployment with geographic (location-based) constraints
- Launch a deployment with minimum hardware specifications

---

### Feature: Deployment Launch

*Enables launching single or multiple deployments across peers.*

**Scenarios:**
- Launch a deployment on a selected peer
- Launch a deployment on any available peer in the network
- Launch multiple deployments communicating on different peers
- Launch multiple deployments communicating on the same peer
- Launch multiple deployments on different peers
- Launch multiple deployments on the same peer

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

### Feature: Deployment in an IP Network

*Support deployments that communicate via IP networking.*

**Scenarios:**
- Connect to a running deployment via SSH
- Launch multiple deployments that communicate via IP network

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

### Feature: Observability

*Provide logs, metrics, and tracing for deployed services and system behavior.*

_(Scenarios to be defined)_

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

### Feature: Stateful Deployment

*Handle deployments with data persistence across reboots or environments.*

**Scenarios:**
- Attempt to restore data from an invalid or missing storage path
- Launch a new deployment restoring from a local volume
- Persist deployment data to a local volume (disk)

---

### Feature: Supervision

*Monitor and react to deployment or system failures automatically.*

_(Scenarios to be defined)_

---

### Feature: System and Peer Status

*View DMS status, peer information, and peer connectivity.*

**Scenarios:**
- Ensure connected peers list reflects only active connections
- Ping a connected peer
- View DMS status (version, onboarding status, DMS DID)
- View connected peers
- View self peer information

---

## Next Steps

- Define scenarios that uses Failure Recovery, Supervision and Observability.
- Identify other scenarios for DMS acceptance tests and NuNet platform.
- Define features and scenarios for other projects like running external nodes, appliance, LLM use cases. 
- Implement the scenarios prioritizing the most frequently used functionalities as well as newly implemented features and use cases.

---
