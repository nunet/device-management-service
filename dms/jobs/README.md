# orchestrator

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

This pacakge manages local jobs and their allocation, including relation to execution environments, etc. It will manage jobs through whatever executor it's running (Vontainer, VM, Direct_exe, Java etc).

### 2. Structure and organisation

Here is quick overview of the contents of this directory:

`TBD`

### 3. Functionality

`TBD`

**Note: the functionality of DMS is being currently developed. See the [proposed](#6-proposed-functionality--requirements) section for the suggested design of interfaces and methods.**

### 4. Data Types

`TBD`

**Note: the functionality of DMS is being currently developed. See the [proposed](#6-proposed-functionality--requirements) section for the suggested data types.**

### 5. Testing

`TBD`

### 6. Proposed Functionality / Requirements 

#### List of issues

All issues that are related to the implementation of `dms` package can be found below. These include any proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [dms package implementation](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=opened&label_name%5B%5D=collaboration_group_24%3A%3A33&first_page_size=20)


#### Interfaces & Methods

##### `proposed` Job interface

```
type Job_interface interface {
	// extends graph.Vertex interface in order to be able to build 
	// job structure as a special instantiation of a graph
	// commented it since it not shown in the class diagram
    //graph.Vertex

	// additional methods
	getPods() []jobs.Pod
}
```

`getPods`: will fetch list of pods currently running on the machine

##### `proposed` Pod interface

```
type Pod_interface interface {
	combineCapabilities() dms.Capability
}
```

`combineCapabilities`: will combine the capability requirements of different jobs to calculate total capabality needed for a Pod.

##### `proposed` JobLink interface

```
type JobLink_interface interface {
	// extends graph.Edge interface
	// commented as it is not shown in class diagram
    // dms.graph.Edge

    validateRelations()
}
```
`validateRelations`: It validates the JobLink properties provided. 

##### `proposed` Allocation interface

```
type Allocation_interface interface {
	// start the allocation execution via the executor package
    start()                
    
    // send message to another actor
    sendMessage(telemetry.Message)
    
    // register the allocation with the node
    register()
}
```

`start`: starts the allocation execution

`sendMessage`: sends a message to another actor (Node/Allocation)

`register`: registers the allocation with the node that it is running on

#### Data types

- `proposed` `Job`: Nunet job which will be sent to the network wrapped as a `BidRequest`. If needed it will have child jobs to be executed. The relation between parent and child job needs to be specified. 

```
type Job struct {
	// JobID is the unique identifier of the job 
	JobId 				models.ID
    
    // all information that is needed to convert a job description into ExecutionRequest
	// in principle, the logic of inputs and outputs can be expressed 
	// within graph structure, when (and if) we develop general interface 
	// of storage.StorageProvider and make it extend Vertex interface
	// until that we need to use special structures outside Graph interface
	inputs 		Slice[models.SpecConfig]
	outputs		Slice[models.SpecConfig]

    RequiredCapability dms.Capability

    // child jobs that have to be executed as part of this job
    // specified using JobLink structure
    children Slice[dms.jobs.JobLink] // in graph.Vertex type this is outEdge

    // extend graph.Vertex type 
    // given that, all children jobs will be expressed via the this interface
	// just expanding on that here
	// clearly saying that a job can have only one parent but many children
    // commented it as we have not included this in the class diagram
    // dms.graph.Vertex
}

```

- `proposed` `JobLink`: specifies the properties that relate parent and child job.

```
type JobLink struct {
	// extends Edge struct
	// commented as it is not shown in class diagram
    // dms.graph.Edge // but types Vertexes on both end of the link to Jobs...

	// RelationProperties captures the relation between the parent and the child job
    RelationProperties models.SpecConfig
	
    // Child job that is linked to the parent job
    RelatedJob dms.jobs.Job
}
```

- `proposed` `Pod`: collection of jobs that need to be executed on the same machine.

```
type Pod struct {
	// identifer of the Pod
    ID models.ID 
	
    // Combined capability required by the Pod
    podCapability dms.Capability
	
    // Jobs that are part of the Pod
    jobs Slice[dms.jobs.Job]
}
```

- `proposed` `Allocation`: maps the job to the process on a executor. Each Allocation is an Actor.

```
type Allocation struct {
    // allocation is an actor (so extends Actor struct)
    // commented since it is not shown in class diagram
    // dms.orchestrator.Actor 
	
    // identifier for the allocation
    ID      dms.jobs.AllocationID
	
    // identifier of the job that had led to this allocation
    jobID   dms.jobs.Job.JobID

    // identifier of the node that is executing the allocation
	NodeID  dms.node.NodeID
	
    // identifier of the node that created the job
    Source  models.ID

    // Mailbox of the Allocation
    Mailbox    dms.orchestrator.Mailbox
}

```

- `proposed` `AllocationID`: identifier for Allocation objects.

```
type AllocationID struct {
	// UUID is the unique identifier of the Allocation
    UUID models.ID.UUID

	// CID is a Content identifier that can be used in DHT or otherwise;
	// CIDs may be especially useful for routing messages directly to allocations via Kademlia DHT
	// it is not clear if we are going to use it now
	CID string 
}

```

### 7. References

The DMS is being refactored and augmented with several new functionalities. The proposed class diagram can be found here:
- [Class Diagram - Source](https://gitlab.com/nunet/device-management-service/-/blob/develop/specs/classDiagrams/dms-global.mermaid)
- [Class Diagram - Rendered](https://gitlab.com/nunet/device-management-service/-/blob/develop/specs/classDiagrams/dms-global.svg)

**Allocation as an Actor**: As per initial [specification of NuNet ontology / nomenclature](https://nunet.gitlab.io/research/blog/posts/ontology-and-nomenclature/#actors), `Allocation` is considered as an Actor. That makes a running job a first class citizen of NuNet's Actor model, so being able to send and receive messages and maintain state.

