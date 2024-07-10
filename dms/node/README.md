# node

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

### 1. `proposed` Description

This package is responsible for creation of a `Node` object which is the main actor residing on the machine as long as DMS is running. The `Node` gets created when the DMS is onboarded.

The `Node` is responsible for:

- Communicating with other actors (nodes and allocations) via messages. This will include sending bid requests, bids, invocations, job status etc

- Checking used and free resource before creating allocations

- Continuous monitoring of the machine


### 2. Structure and organisation

Here is quick overview of the contents of this pacakge:

* [README](https://gitlab.com/nunet/device-management-service/-/blob/develop/dms/node/README.md): Current file which is aimed towards developers who wish to use and modify the DMS functionality. 

### 3. Functionality

`TBD`

**Note: the functionality of DMS is being currently developed. See the [proposed](#6-proposed-functionality--requirements) section for the suggested design of interfaces and methods.**


### 4. Data Types

`TBD`

**Note: the functionality of DMS is being currently developed. See the [proposed](#6-proposed-functionality--requirements) section for the suggested data types.**


### 5. Testing

`proposed` Refer to `*_test.go` files for unit tests of different functionalities.

### 6. Proposed Functionality / Requirements 

#### List of issues

All issues that are related to the implementation of `dms` package can be found below. These include any proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [dms package implementation](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=opened&label_name%5B%5D=collaboration_group_24%3A%3A33&first_page_size=20)

#### Interfaces & Methods

##### `proposed` Node_interface

```
type Node_interface interface {
	// Extends Actor interface from models package;
	// which implements message passing logic

	// Somewhat similar to the libp2p.Host interface in the current implementation;
	// (however, network.libp2p package is too low level for this interface in the 
	// new architecture)
	
	models.Actor

    // extends the Orchestrator interface from orchestrator sub-package
	dms.orchestrator.Orchestrator

	// extends Vertex interface, so connecting Actor model with Graph computing model...
	dms.graph.Vertex_interface

	// each node will hold a structure with allocations that will be running on it
	// allocation type is specified in jobs package
	getAllocation(allocationID jobs.Allocation.allocationID) jobs.Allocation

	// allocations will be short-lived objects (depending on the requirements of a job pertaining to that allocation)
	// but (ideally) all allocations that have ever been initiated on a node
	// need to be logged into the local database
	checkAllocationStatus()
	
	// routes message to the allocation of the job that is running on the machine
	routeToAllocation()

	// below methods related to COMPUTE PROVIDER functionality (mostly)
	// All setters and getters are not included into global dms class diagram to make
	// it more compact

	// benchmark Capability of the machine on which the Node is running
	benchmarkCapability()

	// save benchmarked Capability into persistent data store for further retrieval
	setRegisteredCapability()

	// get all registered capability of the node
	getRegisteredCapability()

	// set available Capability, given the registered Capability and the new allocation
	setAvailableCapability()

	// return currently available capability of the node
	getAvailableCapability()

    // reserve resources for a job
	lockCapability(jobs.Pod, dms.Capability)

	// get all locked Capabilities
	getLockedCapabilities()

	// set preferences of a node in the form of Capability Comparator
	setPreferences()

	getPreferences() dms.orchestrator.CapabilityComparator

	// below methods are related to SERVICE PROVIDER functionality (mostly)
	getRegisteredBids(bidRequestID models.ID) []orchestrator.Bid

	// start a new allocation
	startAllocation(orchestrator.Invocation) 

}
```

`getAllocation` method retrieves an `Allocation` on the machine based on the provided `AllocationID`.

`checkAllocationStatus` method will retrieve status of an `Allocation`.

`routeToAllocation` method will route a message to the `Allocation` of the job that is running on the machine.

`benchmarkCapability` method will perform machine benchmarking

`setRegisteredCapability` method will record the benchmarked Capability of the machine into a persistent data store for retrieval and usage (mostly in job orchestration functionality)

`getRegisteredCapability` method will retrieve the benchmarked Capability of the machine from the persistent data store.

`setAvailableCapability` method changes the available capability of the machine when resources are locked 

`getAvailableCapability` method will return currently available capability of the node
	
`lockCapability` method will lock certain amount of resources for a job. This can happen during bid submission. But it must happen once job is accepted and before invocation.

`getLockedCapabilities` method retrieves the locked capabilities of the machine.

`setPreferences` method sets the preferences of a node as `dms.orchestrator.CapabilityComparator`

`getPreferences` method retrieves the node preferences as `dms.orchestrator.CapabilityComparator`

`getRegisteredBids` method retrieves list of bids receieved for a job.

`startAllocation` method will create an allocation based on the invocation received.


#### Data types

##### `proposed` Node

An initial data model for `Node` is defined below.

```
type Node struct {
	// unique identifier
	id dms.node.nodeID

	// unique mailbox
	mailbox models.Mailbox
	
	// access to the local events database on the node
	db db.LocalDatabaseCollector // a type allowing to issue queries to our local database

	// registered capability will be a slow changing value
	// most probably it will not change for the whole lifecycle of a DMS
	// therefore it will be written into the persistent database
	// however it is good to have it in the cache, since it may be used often
	registeredCapability dms.Capability

	// available capability can be always calculated / recalculated
	// by: (1) taking the value of registered Capability (from persistent database)
	// (2) adding all capabilities of current Allocations of the node and locked capabilities
	// available capability will be (1) - (2)..
	// it can also extracted via database queries
	// lets consider that the current available capability will always be cached as a variable too
	// readily available for the Node interface
	availableCapability dms.Capability

	// part of the machine that is reserved for future jobs
	// it is the difference between registered capability and available capability
	// can be calculated readily, but it is good to have it cached too.
	// for orchestration purposes
	lockedCapabilities map[dms.Job]dms.Capability

	// each node has its preferences encoded into CapabilityComparator type
	// which will be used directly in the functions which compare job requirements with availableCapabilities
	// note that Capabilities include price
	// these types and variables are introduced in order to create ability for each node
	// to autonomously 'choose' best options as per individually defined preferences
	// the initial implementation based on these types could be pretty trivial
	// but we want this to be extendable to unlimited complexity of behaviors
	preferences dms.orchestrator.CapabilityComparator

	allocations slice[jobs.AllocationID] // list of all allocations running on the node

	// -- FOLLOWING FIELDS NOT INCLUDED INTO dms global class diagram --

	// indices of activity
	// we may need to have more indexes
	// note, that all information in these indexes in principle
	// should be re-constructable from the local database 
	jobIndex dms.jobs.JobIndex // index of all accepted job posts

 	// index of all sent/received bid requests
	// (note that a Node can both receive and send bids depending on the mode)
	// (if to implement this in one or more data structure -- shall be decided by developers)
	bidRequestsSentIndex dms.orchestrator.BidRequestsSentIndex
	
    bidRequestsReceivedIndex dms.orchestrator.BidRequestsReceivedIndex

	// index of all received/sent bids
	// (note, that a dms can both receive and send bids, depending on which role
	// it takes in the orchestration -- compute provider or service provider;
    // both, however are available via the same dms package
	// and may even be combined -- possibly in the future)
	bidsSentIndex dms.orchestrator.BidsSentIndex
	
    bidsReceivedIndex dms.orchestrator.BidsReceivedIndex

}

```


##### `proposed` NodeID


```
type NodeID struct {
	// ID is unique identifier created by DMS 
    ID models.ID.UUID
   
    // CID is a Content identifier that can be used in DHT or otherwise
    CID models.ID.CID

    // libp2p peerID
	PeerID string

	// DID is the decentralized identifier that can be used for authentication and authorization
	DID string 
}
```


### 7. References

The DMS is being refactored and augmented with several new functionalities. The proposed class diagram can be found here:
- [Class Diagram - Source](https://gitlab.com/nunet/device-management-service/-/blob/develop/specs/classDiagrams/dms-global.mermaid)
- [Class Diagram - Rendered](https://gitlab.com/nunet/device-management-service/-/blob/develop/specs/classDiagrams/dms-global.svg)






