# api

- [Project README](https://gitlab.com/nunet/device-management-service/-/blob/develop/README.md)
- [Release/Build Status](https://gitlab.com/nunet/device-management-service/-/releases)
- [Changelog](https://gitlab.com/nunet/device-management-service/-/blob/develop/CHANGELOG.md)
- [License](https://www.apache.org/licenses/LICENSE-2.0.txt)
- [Contribution guidelines](https://gitlab.com/nunet/device-management-service/-/blob/develop/CONTRIBUTING.md)
- [Code of conduct](https://gitlab.com/nunet/device-management-service/-/blob/develop/CODE_OF_CONDUCT.md)
- [Secure coding guidelines](https://gitlab.com/nunet/documentation/-/wikis/secure-coding-guidelines)

## Table of Contents

1. [Description](#1-description)
2. [Structure and organisation](#2-structure-and-organisation)
3. [Functionality](#3-functionality)
4. [Data Types](#4-data-types)
5. [Testing](#5-testing)
6. [Proposed Functionality/Requirements](#6-proposed-functionality--requirements)
7. [References](#7-references)


## Specification

### 1. Description

The api package contains all API functionality of Device Management Service (DMS). DMS exposes various endpoints through which its different functionalities can be accessed.

### 2. Structure and organisation

Here is quick overview of the contents of this directory:

* [README](README.md): Current file which is aimed towards developers who wish to use and modify the api functionality. 

* [api](api.go): This file contains router setup using Gin framework. It also applies Cross-Origin Resource Sharing (CORS) middleware and OpenTelemetry middleware for tracing. Further it lists down the endpoint URLs and the associated handler functions.

* [debug](debug.go): This file contains endpoints which are only available when `DEBUG` mode is enabled.

* [device](device.go): This file contains endpoints to retrieve and modify the device status.

* [onboarding](onboarding.go): This file contains endpoints related to the onboarding functionality catered towards compute providers.

* [peers](peers.go): This file contains various endpoints related to the p2p functionality of DMS. 

* [run](run.go): This file contains various endpoints related to the deployment and execution of jobs.

* [telemetry](telemetry.go): This file contains the endpoint to calculate available free resources in a machine.

* [transactions](transactions.go): This file contains the endpoints related to blockchain transactions.

* [vm](vm.go): This file contains the endpoints related to starting a [firecracker VM](https://firecracker-microvm.github.io/) with custom or default configuration.

* [docs](docs): This directory contains the swagger documentation of the API.

All of these files have a counterpart named as `*_test.go` which contains the unit tests for the corresponding endpoints.

### 3. Functionality

The following sections describe the different functionality of the DMS covered in the `api` package.

#### Device Status

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         |`/device/status`   |
| **method**:           | `HTTP GET`        |
| **output**:           | `Device Status`   |

This endpoint retrieves the current status of the machine (online / offline). It returns an error message in case of any failure during the operation.

#### Change Device Status

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         |  `/device/status` |
| **method**:           | `HTTP POST`       |
| **output**:           | `Success Message` |


This endpoint changes the current status of the machine (online / offline). It returns a success message to indicate the operation was successful. 

It returns an error message in case of any failure during the operation.


#### Create Payment Address

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/onboarding/address/new` |
| **method**:           | `HTTP GET` |
| **output**:           |`models.onboarding.BlockchainAddressPrivKey`| 

This endpoint creates a new blockchain payment address for the user. It returns an error message in case of any failure during the operation.

#### Onboard

| Item                  | Value                 |
|--------               |---------              |
| **endpoint**:         | `/onboarding/onboard` |
| **method**:           | `HTTP POST`           |
| **input**:            |  `models.onboarding.CapacityForNunet` |
| **output**:           | `models.onboarding.Metadata` |

This endpoint executes the onboarding process for a compute provider device. It expects various details from the user with respect to hardware resources, price etc as specified in `models.onboarding.CapacityForNunet`. Upon succesful onboarding, it returns the machine metadata recorded by the DMS. 

It returns an error message in case of any failure during the operation.

#### Get Metadata

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/onboarding/metadata` |
| **method**:           | `HTTP GET` |
| **output**:           | `models.onboarding.Metadata`|

This endpoint fetches the current metadata of the onboarded device. It returns an error message in case of any failure during the operation.

#### Provisioned Capacity

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/onboarding/provisioned` |
| **method**:           | `HTTP GET` |
| **output**:           | `models.onboarding.Provisioned` |

This endpoint fetches the total capacity of the machine that is onboarded to Nunet.

#### Onboard Status

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/onboarding/status` |
| **method**:           | `HTTP GET` |
| **output**:           | `models.onboarding.OnboardingStatus` |

This endpoint returns onboarding status of the machine along with some metadata. It returns an error message in case of any failure during the operation.

#### Resource Config

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/onboarding/resource-config` |
| **method**:           | `HTTP POST` |
| **input**:            | `models.onboarding.CapacityForNunet` |
| **output**:           |`models.onboarding.Metadata` |

This endpoint allows the user to change the configuration of the resources onboarded to Nunet. It returns the updated metadata of the machine.

It returns an error message in case of any failure during the operation.

#### Offboard

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/onboarding/offboard` |
| **method**:           | `HTTP DELETE` |
| **output**:           | `Success Message` |

This endpoint allows the user to remove the resources onboarded to Nunet. It returns a message indicating whether the operation was successful or not.

#### List Peers

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/peers` |
| **method**:           | `HTTP GET` |
| **output**:           | `Peer List` |

This endpoint gets a list of `peerID`s that the node can see within the network. It returns an error message in case of any failure during the operation.

#### List DHT Peers

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/peers/dht` |
| **method**:           | `HTTP GET` |
| **output**:           | `DHT Peer List` |

This endpoint gets a list of `peerID`s that the node has received a dht update from. It returns an error message in case of any failure during the operation.

#### List Kad DHT Peers

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/peers/kad-dht` |
| **method**:           | `HTTP GET` |
| **output**:           | `DHT Peer List` |

This endpoint gets a list of `peerID`s that the node has received a dht update from. It returns an error message in case of any failure during the operation.

#### Self Peer Info (`TBD`)

**Note: libp2p.SelfPeerInfo() method is deprecated. This will be replaced by a new method. The output data type and name of this endpoint will need to be confirmed as per the new implementation** 

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/peers/self`   |
| **method**:           | `HTTP GET` |
| **output**:           | `Peer Info` |

This endpoint gets the peer info of the libp2p node. It returns an error message in case of any failure during the operation.

**Note: Chat functionality is expected to be deprecated post refactoring of DMS.** 
---

#### List Chat (`TBD`)

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/peers/chat` |
| **method**:           | `HTTP GET` |
| **output**:           | `List of chats` |

This endpoint gets the list of chat requests from peers. It returns an error message in case of any failure during the operation.

#### Clear Chat (`TBD`)

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/peers/chat/clear` |
| **method**:           | `HTTP GET`|
| **output**:           | `Message` |

This endpoint clears the chat request streams from peers.

#### Start Chat (`TBD`)

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/peers/chat/start` |
| **method**:           |`HTTP GET` |
| **output**:           | `None` |

This endpoint starts a chat session with a peer.

#### Join Chat (`TBD`)

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/peers/chat/join` |
| **method**:           | `HTTP GET` |
| **output**:           | `None` |

---

#### Dump DHT (`TBD`)

**Note: libp2p.DumpDHT() method is deprecated. This will be replaced by a new method. The output data type and name of this endpoint will need to be confirmed as per the new implementation** 

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/dht` |
| **method**:           | `HTTP GET` |
| **output**:           | `DHT Content` |

This endpoint returns the entire DHT content. It returns an error message in case of any failure during the operation.

#### Default DepReq Peer (`TBD`)

**Note: This endpoint is expected to be deprecated post refactoring of DMS.**

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/peers/depreq`  |
| **method**:           | `HTTP GET` |
| **output**:           | `Message including peerID` |

This endpoint is used to set peer as the default receipient of deployment requests by setting the peerID parameter on GET request. 

Note:
* By sending a GET request without any parameters we get the peer currently set as default deployment request receiver. 

* Sending a GET request with `peerID` parameter set to '0' will remove default deployment request receiver.

**Note: Implementation of below endpoints is expected to change during refactoring of DMS. The below content will need to updated accordingly**
---

#### Clear File Transfer Requests (`TBD`)

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/peers/file/clear` |
| **method**:           | `HTTP GET` |
| **output**:           | `Message` |

This endpoint is used to clear file transfer request streams from peers.

#### List File Transfer Requests (`TBD`)

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/peers/file` |
| **method**:           | `HTTP GET` |
| **output**:           | `Message` |

This endpoint is used to get a list of file transfer requests from peers.

#### Send File Transfer (`TBD`)

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/peers/file/send` |
| **method**:           | `HTTP GET` |
| **output**:           | `NIL` |

This endpoint is used to initiate file transfer to a peer. Note that `filePath` and `peerID` are required arguments.

#### Accept File Transfer (`TBD`)

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/peers/file/accept` |
| **method**:           | `HTTP GET` |
| **output**:           | `NIL` |

This endpoint is used to initiate file transfer to a peer. Note that `filePath` and `peerID` are required arguments.

#### Request Service

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/run/request-service` |
| **method**:           | `HTTP POST` |
| **output**:           | `TBD` |

`TBD`

#### Deployment Request

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/run/deploy`   |
| **method**:           | `HTTP GET` |
| **output**:           | `TBD` |

`TBD`

#### List Checkpoint

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/run/checkpoints` |
| **method**:           | `HTTP GET` |
| **output**:           | `Checkpoints` |

`TBD`

#### Get Free Resources

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/telemetry/free` |
| **method**:           | `HTTP GET` |
| **output**:           | `Free resources` (`TBD`) |

This endpoint checks and returns the amount of free resources available in a machine.

#### Get Job Transaction Hashes

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/transactions` |
| **method**:           | `HTTP GET` |
| **output**:           | `Transaction Hashes` |

This endpoint gets the list of transaction hashes along with the date and time of jobs done.

#### Request Reward

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/transactions/request-reward` |
| **method**:           | `HTTP POST` |
| **output**:           | `Reward Response` |

This endpoint takes request from the compute provider, talks with Oracle and releases tokens if conditions are met.

#### Send Transaction Status

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/transactions/send-status` |
| **method**:           | `HTTP POST` |
| **output**:           | `Transaction Status` |

This endpoint returns the status of a blockchain transaction such as token withrawal.

#### Update Transaction Status

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/transactions/update-status` |
| **method**:           | `HTTP POST` |
| **output**:           | `Message` |

This endpoint updates the status of saved transactions by fetching info from blockchain using koios REST API.

#### Start Custom

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/vm/start-custom` |
| **method**:           | `HTTP POST` |
| **output**:           | `Message` |

This endpoint start a firecracker VM with custom configuration. It returns a message indicating success of failure of the operation.

#### Start Default

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/vm/start-default` |
**method**:             | `HTTP POST` |
**output**:             | `Message` |

This endpoint start a firecracker VM with default configuration. It returns a message indicating success or failure of the operation.

_Debug Endpoints - These endpoints are only available when `DEBUG` mode is enabled._
---


#### Manual DHT Update

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/dht/update` |
| **method**:           | `HTTP GET` |
| **output**:           | `Message` |

This endpoint initiates a manual update of the DHT. It returns a message upon successful operation.

#### Cleanup Peer

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/cleanup` |
| **method**:           | `HTTP GET`  |
| **output**:           | `Message` |

This endpoint removes a peer from the local database.  It returns a message indicating success or failure of the operation.

#### Ping Peer

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/ping` |
| **method**:           | `HTTP GET` |
| **output**:           | `Ping Peer Response` |

This endpoint pings a peer and checks the peer's presence in the DHT. It also returns the round trip time (RTT) for the ping.

It returns an error message in case of any failure during the operation.

#### Old Ping Peer

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/oldping` |
| **method**:           | `HTTP GET` |
| **output**:           | `Ping Peer Response` |

This endpoint pings a peer and checks the peer's presence in the DHT. It also returns the round trip time (RTT) for the ping. It returns an error message in case of any failure during the operation.

#### Dump Kademlia DHT

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/kad-dht` |
| **method**:           | `HTTP GET` |
| **output**:           | `DHT Content` (`TBD`) |

This endpoint returns the DHT contents. 

### 4. Data Types

The API functionality of DMS consists of following data types:

- `models.onboarding.BlockchainAddressPrivKey`: This contains public key, private key and mnenmoic associated with it. This is generated when user opts to create a payment address / wallet using the api functionality.

- `models.onboarding.CapacityForNunet`: This is the input provided by the compute provider user to start the onboarding process.

- `models.onboarding.Metadata`: This contains information about the machine stored by the DMS. It is generated as a result of the onboarding process.

- `models.onboarding.Provisioned`: This has total capacity of the machine onboarded by the user.

- `models.onboarding.OnboardingStatus`: This is returned while retrieving the onboarding status.

- `Available Resources (TBD)`: This will have the available capacity of the machine that can be considered for running a job.

- `DHT content (TBD)`: This the data structure of content stored in the DHT. 

**Note: More data types are expected to be added as per DMS refactoring**

### 5. Testing

#### Unit Tests

All unit tests for various functionalities can be found in files with `_test` in their name.

#### Functional tests

- **TBD with Abhishek**

### 6. Proposed Functionality / Requirements 

#### List of issues

All issues that are related to the design of API package can be found below. These include any proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [API package design](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=opened&label_name%5B%5D=collaboration_group_24%3A%3A12&first_page_size=20)

### 7. References

The DMS is being refactored and augmented with several new functionalities. The proposed class diagram can be found here:
- [Class Diagram - Source](https://gitlab.com/nunet/device-management-service/-/blob/develop/specs/classDiagrams/dms-global.mermaid)
- [Class Diagram - Rendered](https://gitlab.com/nunet/device-management-service/-/blob/develop/specs/classDiagrams/dms-global.svg)
