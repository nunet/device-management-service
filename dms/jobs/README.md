# jobs

- [Project README](https://gitlab.com/nunet/device-management-service/-/blob/main/README.md)
- [Release/Build Status](https://gitlab.com/nunet/device-management-service/-/releases)
- [Changelog](https://gitlab.com/nunet/device-management-service/-/blob/main/CHANGELOG.md)
- [License](https://www.apache.org/licenses/LICENSE-2.0.txt)
- [Contribution Guidelines](https://gitlab.com/nunet/device-management-service/-/blob/main/CONTRIBUTING.md)
- [Code of Conduct](https://gitlab.com/nunet/device-management-service/-/blob/main/CODE_OF_CONDUCT.md)
- [Secure Coding Guidelines](https://gitlab.com/nunet/team-processes-and-guidelines/-/blob/main/secure_coding_guidelines/README.md)
- [EnsembleConfig Documentation](https://gitlab.com/nunet/device-management-service/-/blob/main/jobs/ensemble_fields_reference.md)

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

This pacakge manages local jobs and their allocation, including relation to execution environments, etc. It will manage jobs through whatever executor it's running (Vontainer, VM, Direct_exe, Java etc).

### Structure and Organisation

Here is quick overview of the contents of this directory:

`TBD`

### Class Diagram

The class diagram for the `jobs` package is shown below.

#### Source file

[jobs Class Diagram](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/jobs/specs/class_diagram.puml)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/main"
!$packageRelativePath = "/dms/jobs"
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath
 
!include $packageUrlGitlab/specs/class_diagram.puml
```

### Functionality

`TBD`

**Note: the functionality of DMS is being currently developed. See the [proposed](#7-proposed-functionality--requirements) section for the suggested design of interfaces and methods.**

### Data Types

`TBD`

**Note: the functionality of DMS is being currently developed. See the [proposed](#7-proposed-functionality--requirements) section for the suggested data types.**

### Testing

`TBD`

### Proposed Functionality / Requirements 

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

- `proposed` `dms.jobs.Job`: Nunet job which will be sent to the network wrapped as a `BidRequest`. If needed it will have child jobs to be executed. The relation between parent and child job needs to be specified. 

```
type Job struct {
	// JobID is the unique identifier of the job 
	JobId 				types.ID
    
    // all information that is needed to convert a job description into ExecutionRequest
	// in principle, the logic of inputs and outputs can be expressed 
	// within graph structure, when (and if) we develop general interface 
	// of storage.StorageProvider and make it extend Vertex interface
	// until that we need to use special structures outside Graph interface
	inputs 		Slice[types.SpecConfig]
	outputs		Slice[types.SpecConfig]

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

- `proposed` `dms.jobs.JobLink`: specifies the properties that relate parent and child job.

```
type JobLink struct {
	// extends Edge struct
	// commented as it is not shown in class diagram
    // dms.graph.Edge // but types Vertexes on both end of the link to Jobs...

	// RelationProperties captures the relation between the parent and the child job
    RelationProperties types.SpecConfig
	
    // Child job that is linked to the parent job
    RelatedJob dms.jobs.Job
}
```

- `proposed` `dms.jobs.Pod`: collection of jobs that need to be executed on the same machine.

```
type Pod struct {
	// identifer of the Pod
    ID types.ID 
	
    // Combined capability required by the Pod
    podCapability dms.Capability
	
    // Jobs that are part of the Pod
    jobs Slice[dms.jobs.Job]
}
```

- `proposed` `dms.jobs.Allocation`: maps the job to the process on a executor. Each Allocation is an Actor.

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
    Source  types.ID

    // Mailbox of the Allocation
    Mailbox    dms.orchestrator.Mailbox
}

```

- `proposed` `dms.jobs.AllocationID`: identifier for Allocation objects.

```
type AllocationID struct {
	// UUID is the unique identifier of the Allocation
    UUID types.ID.UUID

	// CID is a Content identifier that can be used in DHT or otherwise;
	// CIDs may be especially useful for routing messages directly to allocations via Kademlia DHT
	// it is not clear if we are going to use it now
	CID string 
}

```

### References

**Allocation as an Actor**: As per initial [specification of NuNet ontology / nomenclature](https://nunet.gitlab.io/research/blog/posts/ontology-and-nomenclature/#actors), `Allocation` is considered as an Actor. That makes a running job a first class citizen of NuNet's Actor model, so being able to send and receive messages and maintain state.

