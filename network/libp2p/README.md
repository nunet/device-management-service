# libp2p

- [Project README](https://gitlab.com/nunet/device-management-service/-/blob/develop/README.md)
- [Release/Build Status](https://gitlab.com/nunet/device-management-service/-/releases)
- [Changelog](https://gitlab.com/nunet/device-management-service/-/blob/develop/CHANGELOG.md)
- [License](https://www.apache.org/licenses/LICENSE-2.0.txt)
- [Contribution Guidelines](https://gitlab.com/nunet/device-management-service/-/blob/develop/CONTRIBUTING.md)
- [Code of Conduct](https://gitlab.com/nunet/device-management-service/-/blob/develop/CODE_OF_CONDUCT.md)
- [Secure Coding Guidelines](https://gitlab.com/nunet/team-processes-and-guidelines/-/blob/main/secure_coding_guidelines/README.md)

## Table of Contents

1. [Description](#1-description)
2. [Structure and Organisation](#2-structure-and-organisation)
3. [Class Diagram](#3-class-diagram)
4. [Functionality](#4-functionality)
5. [Data Types](#5-data-types)
6. [Testing](#6-testing)
7. [Proposed Functionality/Requirements](#7-proposed-functionality--requirements)
8. [References](#8-references)

## Specification

### 1. Description

This package implements `Network` interface defined in root level network dir.

#### `proposed` Requirements

proposed: @sam

##### Peer discovery and handshake

Instead of keeping track of all the peers. Peers should only in touch with peers of their types in terms of network latency, resources, or uptime.

A reason for this is, if some low performing peer is with some high performing peers, and job is distributed among them, it can slow others peers as well overall. 

##### Max number of handshake peers

Different nodes will have different requirements regarding the number of peers that they should remain handshaking with. e.g. a small node on a mobile network will not need to maintain a large list of peers. But, a node acting as network load balancer in a data center might need to maintain a large list of peers.

##### Filter list

We can have filters that ensures that the only peers that are handshaked with are ones that meet certain criteria. The following list is not exhaustive:

1. Network latency. Have a set of fastest peers.
2. Resource. Relates to job types.
3. Uptime. Connect to peers who are online for certain period of time.

##### Network latency

For the network latency part, DMS should also be able to keep latency table between ongoing jobs on different CPs. The network package should be able to report it to the master node (SP). Orchestrator can then make decision on whether to replace workers or not.

**Thoughts**:

* Filter peers at the time of discovery. Based on above parameters.
* SP/orchestrator specifies what pool of CP it is looking for.
* CP connects to same kind of CP.
* Can use gossipsub.

### 2. Structure and Organisation

Here is quick overview of the contents of this pacakge:

* [README](https://gitlab.com/nunet/device-management-service/-/blob/develop/network/libp2p/README.md): Current file which is aimed towards developers who wish to use and modify the package functionality.

* [conn](https://gitlab.com/nunet/device-management-service/-/blob/develop/network/libp2p/conn.go): This file defines the method to ping a peer.

* [dht](https://gitlab.com/nunet/device-management-service/-/blob/develop/network/libp2p/dht.go): This file contains functionalities of a libp2p node. It includes functionalities for setting up libp2p hosts, performing pings between peers, fetching DHT content, checking the status of a peer and validating data against their signatures.

* [discover](https://gitlab.com/nunet/device-management-service/-/blob/develop/network/libp2p/discover.go): This file contains methods for peer discovery in a libp2p node. 

* [filter](https://gitlab.com/nunet/device-management-service/-/blob/develop/network/libp2p/filter.go): This file defines functionalities for peer filtering and connection management in a libp2p node. 

* [init](https://gitlab.com/nunet/device-management-service/-/blob/develop/network/libp2p/init.go): This file defines configurations and initialization logic for a libp2p node.

* [libp2p](https://gitlab.com/nunet/device-management-service/-/blob/develop/network/libp2p/libp2p.go): This file defines stubs for Libp2p peer management functionalities, including configuration, initialization, events, status, stopping, cleanup, ping, and DHT dump.

* [p2p](https://gitlab.com/nunet/device-management-service/-/blob/develop/network/libp2p/p2p.go): This file defines a Libp2p node with core functionalities including discovery, peer management, DHT interaction, and communication channels, with several stub implementations.

### 3. Class Diagram

The class diagram for the libp2p sub-package is shown below.

#### Source file

[libp2p Class Diagram](https://gitlab.com/nunet/device-management-service/-/blob/develop/network/libp2p/specs/class_diagram.puml)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/develop"
!$packageRelativePath = "/network/libp2p"
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath
 
!include $packageUrlGitlab/specs/class_diagram.puml
```

### 4. Functionality

As soon as DMS starts, and if it is onboarded to the network, `libp2p.RunNode` is executed. This gets up entire thing related to libp2p. Let's run down through it to see what it does.

1. RunNode calls `NewHost`. NewHost in itself does a lot of things. Let's dive into the NewHost:

- It creates a [connection manager](https://github.com/libp2p/go-libp2p/blob/v0.33.2/p2p/net/connmgr/connmgr.go#L110). This defines what is the upper and lower limit of peers current peer will connect to.
- It then defines a multiaddr filter which is used to deny discovering on local network. This was added to stop scanning local network in a data center.
- NewHost then sets various options for the and passes it to libp2p.New to create a new host. Options such as NAT traversal is configured at this point.

2. Getting back to other RunNode, it calls `p2p.BootstrapNode(ctx)`. Bootstrapping basically is connecting to initial peers.

3. Then the function continues setting up streams. Streams are bidirectional connection between peers. More on this in next section. Here is an example of setting up a stream handler on host for particular protocol:

```go
host.SetStreamHandler(protocol.ID(DepReqProtocolID), depReqStreamHandler)
```

4. After that, we have `discoverPeers`, 

```go
go p2p.StartDiscovery(ctx, utils.GetChannelName())
```

5. After that, we have DHT update and get functions to store information about peer in peerstore.


#### Streams

Communication between libp2p peers, or more generally DMS happens using libp2p streams. A DMS can have one or many stream with one or more peer. We currently we have adopted following streams for our usecases.

1. Ping

We can count this as internal to libp2p and is used for operational purposes. Unlike ICMP pings, libp2p [pings works on streams](https://github.com/libp2p/specs/blob/master/ping/ping.md), and is closed after the ping.

2. Chat

A utility functionality to enable chat between peers.

3. VPN

Most recent addition to DMS, where we send IP packets through libp2p stream.

4. File Transfer

File transfer is generally used to carry files from one DMS to another. Most notably used to carry checkpoint files from a job from CP to SP.

5. Deployment Request (DepReq)

Used for deployment of a job and for getting their progress.

##### Current DepReq Stream Handler

Each stream need to have a handler attached to it. Let's get to know more about **deployment request** stream handler. Deployment request handler handles incoming deployment request from the service provider side. Similarly, some function has to listen for update from the service provider side as well. More on that in the next in a minute.

Following is a sequence of event happening on compute provider side:

1. Checks if `InboundDepReqStream` variable is set. And if it is, reply to service provider: `DepReq open stream length exceeded`. Currently we have only 1 job allowed per dep req stream.
2. If above is not the case, we go and read from the stream. We are expecting a 
3. Now is the point to set the `InboundDepReqStream` to the incoming stream value.
4. In unmarshal the incoming message into `types.DeploymentRequest` struct. If it can't, it informs the other party about the it.
5. Otherwise, if everything is going well till now, we check the `txHash` value from the depReq. And make sure it exist on the blockchain before proceeding. If the txHash is not valid, or it timed out while waiting for validation, we let the other side know.
6. Final thing we do it to put the depReq inside the `DepReqQueue`.

After this step, the command is handed over to `executor` module. Please refer to [Relation between libp2p and docker modules](#relation-between-libp2p-and-docker-modules).

Deployment Request stream handler can further be segmented into different message types:

```
MsgDepReq    = "DepReq"
MsgDepResp   = "DepResp"
MsgJobStatus = "JobStatus"
MsgLogStderr = "LogStderr"
MsgLogStdout = "LogStdout"
```
Above message types are used by various functions inside the stream. Last 4 or above is handled on the SP side. Further by the websocket server which started the deployment request. This does not means CP does not deals with them.

##### Relation between libp2p and docker modules

When DepReq streams receives a deployment request on the stream, it does some json validation, and pushes it to `DepReqQueue`. This extra step instead of directly passing the command to docker package was for decoupling and scalibility.

There is a `messaging.DeploymentWorker()` goroutine which is launched at DMS startup in `dms.Run()`.

This `messaging.DeploymentWorker()` is the crux of the job deployment, as what is done in current proposed version of DMS. Based on executor type (currently firecracker and docker), it was passed to specific functions on different modules.

#### PeerFilter Interface
`PeerFilter` is an interface for filtering peers based on a specified criteria.

```
type PeerFilter interface {
	satisfies(p peer.AddrInfo) bool
}
```


### 5. Data Types

- `types.DeploymentResponse`: DeploymentResponse is initial response from the Compute Provider (CP) to Service Provider (SP). It tells the SP that if deployment was successful or was declined due to operational or validational reasons. Most of the validation is just error check at stream handling or executor level.


- `types.DeploymentUpdate`: DeploymentUpdate update is used to inform SP about the state of the job. Most of the update is handled using libp2p stream on network level and websocket on the user level. There is no REST API defined. This should change in next iteration. See the proposed section for this.

On the service provider side, we have `DeploymentUpdateListener` listening to the stream for any activity from the computer provider for update on the job. 

Based on the message types, it does specific actions, which is more or less sending it to websocket client. These message types are `MsgJobStatus`, `MsgDepResp`, `MsgLogStdout` and `MsgLogStderr`

- `network.libp2p.DHTValidator`: `TBD`

```
type DHTValidator struct {
	PS peerstore.Peerstore
}
```

- `network.libp2p.SelfPeer`: `TBD`

```
type SelfPeer struct {
	ID    string
	Addrs []multiaddr.Multiaddr
}
```

- `network.libp2p.NoAddrIDFilter`: filters out peers with no listening addresses
// and a peer with a specific ID

```
type NoAddrIDFilter struct {
	ID peer.ID
}
```

- `network.libp2p.Libp2p`: contains the configuration for a Libp2p instance

```
type Libp2p struct {
	Host   host.Host
	DHT    *dht.IpfsDHT
	PS     peerstore.Peerstore
	peers  []peer.AddrInfo
	config Libp2pConfig
}

type Libp2pConfig struct {
	PrivateKey     crypto.PrivKey
	ListenAddr     []string
	BootstrapPeers []multiaddr.Multiaddr
	Rendezvous     string
	Server         bool
	Scheduler      *bt.Scheduler
}
```

- `network.libp2p.Advertisement`: `TBD`

type Advertisement struct {
	PeerID    string `json:"peer_id"`
	Timestamp int64  `json:"timestamp,omitempty"`
	Data      []byte `json:"data"`
}

- `network.libp2p.OpenStream`: `TBD`

```
type OpenStream struct {
	ID         int    `json:"id"`
	StreamID   string `json:"stream_id"`
	FromPeer   string `json:"from_peer"`
	TimeOpened string `json:"time_opened"`
}
```

**Note: Data types are expected to change due to DMS refactoring**

### 6. Testing

`TBD`

### 7. Proposed Functionality / Requirements 

#### List of issues

All issues that are related to the implementation of `network` package can be found below. These include any proposals for modifications to the package or new data structures needed to cover the requirements of other packages.

- [network package implementation](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=opened&label_name%5B%5D=collaboration_group_24%3A%3A39&first_page_size=20)

### 8. References



