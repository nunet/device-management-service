# dms

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

This package is responsible for starting the whole application. It also contains various core functionality of DMS:
- Onboarding compute provider devices
- Job orchestration and management
- Resource management
- Actor implementation for each node 

### Structure and Organisation

Here is quick overview of the contents of this pacakge:

* [README](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/README.md): Current file which is aimed towards developers who wish to use and modify the dms functionality. 

* [dms](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/dms.go): This file contains code to initialize the DMS by loading configuration, starting REST API server etc

* [init](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/init.go): This file creates a new logger instance.

* [sanity_check](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/sanity_check.go): This file defines a method for performing consistency check before starting the DMS. `proposed` _Note that the functionality of this method needs to be developed as per refactored DMS design._

_Subpackages_

* [jobs](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/jobs): Deals with the management of local jobs on the machine.

* [node](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/node): Contains implementation of `Node` as an actor.

* [onboarding](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/onboarding): Code related to onboarding of compute provider machines to the network.

* [orchestrator](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/orchestrator): Contains job orchestration logic.

* [resources](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/resources): Deals with the management of resources on the machine.

`proposed`: All files with `*_test.go` naming convention contain unit tests with respect to the specific implementation.

### Class Diagram

The class diagram for the `dms` package is shown below.

#### Source file

[dms Class diagram](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/specs/class_diagram.puml)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/main"
!$packageRelativePath = "/dms"
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

`proposed` Refer to `*_test.go` files for unit tests of different functionalities.

### Proposed Functionality / Requirements 

#### List of issues

All issues that are related to the implementation of `dms` package can be found below. These include any proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [dms package implementation](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=opened&label_name%5B%5D=collaboration_group_24%3A%3A33&first_page_size=20)


#### Interfaces & Methods

##### `proposed` Capability_interface

```
type Capability_interface interface {
	add()
	subtract()
}
```

`add` method will combine capabilities of two nodes. Example usage - When two jobs have to be run on a single machine, the capability requirements of each will need to be combined.

`subtract` method will subtract two capabilities. Example usage - When resources are locked for a job, the available capability of a machine will need to be reduced.


#### Data types

##### `proposed` dms.Capability

The `Capability` struct will capture all the relevant data that defines the capability of a node to perform the job. At the same time this will be used to define capability requirements that a job requires from a node.

An initial data model for `Capability` is defined below.

```
type Capability struct {
	// Executor is the type of executor available on the machine or the executor 
    // required for the job (example - Docker, VM, WASM etc)
    Executor    string           
	
    // Type specifies the details of type of job (One time, batch, recurring,
    // long running)
    Type        dms.jobs.JobType        
	
    // Resources specifies the description of the resources required
    Resources   dms.resources.Resource

    // Libraries specifies the libraries needed for the job        
	Libraries   []string         
	
    // Locality contains preferred localities of the machine for execution
    Locality    []string         
	
    // Storage specifies the preferred storage options that the machine should have
    Storage         []string         
	
    // Connectivity specifies the network configuration required
    Connectivity    dms.Connectivity          
	
    // Price specifies the price information of the job / machine
    Price           dms.PriceInformation 
	
    // Time specifies the time information of the job / machine
	Time            dms.TimeInformation 
	
    // KYC specifies the KYC requirements or KYC status of the machine 
    KYC  []string        
}
```

##### `proposed` dms.Connectivity

type Connectivity struct {

    // Ports contains the ports that need to be open for the job to run
	Ports []int
	
    // VPN specifies whether VPN is required
    VPN       bool  
}

##### `proposed` dms.PriceInformation

```
type PriceInformation struct {
	// Currency holds which currency is used for pricing ex - NTX
	Currency   string 
	
    // CurrencyPerHour is the price of the machine per hour
    CurrencyPerHour int   
	
    // TotalPerJob is the maximum total price or budget of the job
	TotalPerJob   int 
	
    // Preference is Pricing preference as compared to time
    Preference int 
}
```

##### `proposed` dms.TimeInformation

type TimeInformation struct {
	// Units holds the units of time ex - hours, days, weeks
    Units      string 
	
    // MaxTime holds the maximum time that the job should run
    MaxTime    int

    // Preference holds the time preference as compared to price
	Preference int    
}


### References






