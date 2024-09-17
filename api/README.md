# api

- [Project README](https://gitlab.com/nunet/device-management-service/-/blob/main/README.md)
- [Release/Build Status](https://gitlab.com/nunet/device-management-service/-/releases)
- [Changelog](https://gitlab.com/nunet/device-management-service/-/blob/main/CHANGELOG.md)
- [License](https://www.apache.org/licenses/LICENSE-2.0.txt)
- [Contribution Guidelines](https://gitlab.com/nunet/device-management-service/-/blob/main/CONTRIBUTING.md)
- [Code of Conduct](https://gitlab.com/nunet/device-management-service/-/blob/main/CODE_OF_CONDUCT.md)
- [Secure Coding Guidelines](https://gitlab.com/nunet/team-processes-and-guidelines/-/blob/main/secure_coding_guidelines/README.md)

## Table of Contents

1. [Description](#description)
2. [Structure and Organisation](#structure-and-organisation)
3. [Class Diagram](#class-diagram)
4. [Functionality](#functionality)
5. [Data Types](#data-types)
6. [Testing](#testing)
7. [Proposed Functionality/Requirements](#proposed-functionality--requirements)
8. [References](#references)  


## Specification

### Description

The api package contains all API functionality of Device Management Service (DMS). DMS exposes various endpoints through which its different functionalities can be accessed.

### Structure and Organisation

Here is quick overview of the contents of this directory:

* [README](https://gitlab.com/nunet/device-management-service/-/blob/main/api/README.md?ref_type=heads): Current file which is aimed towards developers who wish to use and modify the api functionality. 

* [api](https://gitlab.com/nunet/device-management-service/-/blob/main/api/api.go): This file contains router setup using Gin framework. It also applies Cross-Origin Resource Sharing (CORS) middleware and OpenTelemetry middleware for tracing. Further it lists down the endpoint URLs and the associated handler functions.

* [debug](https://gitlab.com/nunet/device-management-service/-/blob/main/api/debug.go): This file contains endpoints which are only available when `DEBUG` mode is enabled.

* [device](https://gitlab.com/nunet/device-management-service/-/blob/main/api/device.go): This file contains endpoints to retrieve and modify the device status.

* [onboarding](https://gitlab.com/nunet/device-management-service/-/blob/main/api/onboarding.go): This file contains endpoints related to the onboarding functionality catered towards compute providers.

* [peers](https://gitlab.com/nunet/device-management-service/-/blob/main/api/peers.go): This file contains various endpoints related to the p2p functionality of DMS. 

* [run](https://gitlab.com/nunet/device-management-service/-/blob/main/api/run.go): This file contains various endpoints related to the deployment and execution of jobs.

* [telemetry](https://gitlab.com/nunet/device-management-service/-/blob/main/api/telemetry.go): This file contains the endpoint to calculate available free resources in a machine.

* [transactions](https://gitlab.com/nunet/device-management-service/-/blob/main/api/transactions.go): This file contains the endpoints related to blockchain transactions.

* [vm](https://gitlab.com/nunet/device-management-service/-/blob/main/api/vm.go): This file contains the endpoints related to starting a [firecracker VM](https://firecracker-microvm.github.io/) with custom or default configuration.

* [docs](https://gitlab.com/nunet/device-management-service/-/blob/main/api/docs): This directory contains the swagger documentation of the API.

* [specs](https://gitlab.com/nunet/device-management-service/-/blob/main/api/specs): This directory contains specifications of the package

All of these files have a counterpart named as `*_test.go` which contains the unit tests for the corresponding endpoints.

### Class Diagram

The class diagram for the `api` package is shown below.

#### Source file

[api Class diagram](https://gitlab.com/nunet/device-management-service/-/blob/main/api/specs/class_diagram.puml)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/main"
!$packageRelativePath = "/api"
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath
 
!include $packageUrlGitlab/specs/class_diagram.puml
```

### Functionality

#### Configuration
The default server address for the REST API is `127.0.0.1` which means that the API can only be accessed from the same machine. Also note that REST API functionality runs on port `9999` by default. 

If needed, the port value can be configured by modifying the `dms_config.json` file. DMS looks for the `dms_config.json` at the time of startup. It searches for the file in the following order:
1. first in the current directory in which DMS is running;
2. then in `~/.nunet`
3. and finally in `/etc/nunet`

The parameters `rest.port` and `rest.addr` define the port and the address. The values specified in the `dms_config.json` file will override the default values specified above.

You can use the following format to construct the URL for accessing API endpoints

```
http://localhost:<port>/api/v1/<endpoint>
```

The following sections describe the different endpoints of the DMS covered in the `api` package. You can also refer to the [Swagger docs](https://nunet.gitlab.io/open-api/device-management-api-spec/main/swagger/#/) for the various endpoints.

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
| **output**:           |`types.BlockchainAddressPrivKey`| 

This endpoint creates a new blockchain payment address for the user. It returns an error message in case of any failure during the operation.

#### Onboard

| Item                  | Value                 |
|--------               |---------              |
| **endpoint**:         | `/onboarding/onboard` |
| **method**:           | `HTTP POST`           |
| **input**:            |  `types.CapacityForNunet` |
| **output**:           | `types.Metadata` |

This endpoint executes the onboarding process for a compute provider device. It expects various details from the user with respect to hardware resources, price etc as specified in `types.CapacityForNunet`. Upon succesful onboarding, it returns the machine metadata recorded by the DMS. 

It returns an error message in case of any failure during the operation.

#### Get Metadata

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/onboarding/metadata` |
| **method**:           | `HTTP GET` |
| **output**:           | `types.Metadata`|

This endpoint fetches the current metadata of the onboarded device. It returns an error message in case of any failure during the operation.

#### Provisioned Capacity

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/onboarding/provisioned` |
| **method**:           | `HTTP GET` |
| **output**:           | `types.Provisioned` |

This endpoint fetches the total capacity of the machine that is onboarded to Nunet.

#### Onboard Status

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/onboarding/status` |
| **method**:           | `HTTP GET` |
| **output**:           | `types.OnboardingStatus` |

This endpoint returns onboarding status of the machine along with some metadata. It returns an error message in case of any failure during the operation.

#### Resource Config

| Item                  | Value             |
|--------               |---------          |
| **endpoint**:         | `/onboarding/resource-config` |
| **method**:           | `HTTP POST` |
| **input**:            | `types.CapacityForNunet` |
| **output**:           |`types.Metadata` |

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

### Data Types

The API functionality of DMS consists of following data types:

- `types.BlockchainAddressPrivKey`: This contains public key, private key and mnenmoic associated with it. This is generated when user opts to create a payment address / wallet using the api functionality.

- `types.CapacityForNunet`: This is the input provided by the compute provider user to start the onboarding process.

- `types.Metadata`: This contains information about the machine stored by the DMS. It is generated as a result of the onboarding process.

- `types.Provisioned`: This has total capacity of the machine onboarded by the user.

- `types.OnboardingStatus`: This is returned while retrieving the onboarding status.

- `Available Resources (TBD)`: This will have the available capacity of the machine that can be considered for running a job.

- `DHT content (TBD)`: This the data structure of content stored in the DHT. 

**Note: More data types are expected to be added as per DMS refactoring**

### Testing

#### Unit Tests

All unit tests for various functionalities can be found in files with `_test` in their name.

#### Functional tests

Reference is made to the [test-suite](https://gitlab.com/nunet/test-suite/-/tree/main/stages/functional_tests/features/device-management-service) repository for functional tests for DMS API functionality.

### Proposed Functionality / Requirements 

#### List of issues

All issues that are related to the design of API package can be found below. These include any proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [API package design](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=opened&label_name%5B%5D=collaboration_group_24%3A%3A12&first_page_size=20)

### References


