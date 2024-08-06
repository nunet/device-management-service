# matching

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
3. [Class Diagram](#3-class-diagram)
4. [Functionality](#4-functionality)
5. [Data Types](#5-data-types)
6. [Testing](#6-testing)
7. [Proposed Functionality/Requirements](#7-proposed-functionality--requirements)
8. [References](#8-references)

## Specification

### 1. Description

The `matching` package (subpackage of `dms/orchestrator`) is responsible for the logic of matching requirements of compute workflows deployed and orchestrated on the platform and machine capabilities of compute providers.

The main goal of the package is to define and express the `Comparator` logic for comparing two variables of custom type, which can be defined in different packages within `device-management-service` repository. `Comparator` logic is needed for the following high level functionalities:

1. Comparing computational resource requirements of posted jobs and computational capabilities of machines in the network. This functionality is tightly coupled with the `Capability` model (collection of types and their relations / nesting), which defines the way we describe and declare computational resources. `Comparator` logic to a large extent informs and interacts with `Capability` model, because the latter cannot be conceived without the former. `Comparator` logic is a part of `Capability` model, which central functionality of the platform as it implements decentralized search and matching concept. This aspect is the main current focus of the implemented functionality.

2. According to the current orchestration logic, a `dms` acting as 'service provider', sends to the network a `bidRequest` for each requested job. If a `dms` acting as 'compute provider' decides (using functionality of this package) that it is willing to execute a job, it sends a `bid`. Both `bidRequest` and `bid` types contain not only declarations of compute requirements and demands, but also pricing, timing and other preferences that need to be considered by the `Comparator` logic. This aspect is the secondary focus of the implemented functionality.

3. `Comparator` logic will also be used in sorting `bid`s or `bidRequest`s in order to choose a preferred one (e.g. if and when a `dms` acting as 'service provider' will receive multiple `bid`s for the same `bidRequest`, it will have to choose the best in the list). This aspect is not yet implemented but proposed. 

The `matching` sub-package should be considered in the context of `orchestrator` functionality and general 'proposed' Job Orchestration logic (see [`orchestrator` package README.md](https://gitlab.com/nunet/device-management-service/-/blob/develop/dms/orchestrator/README.md)).

Provided implementation of `Capability` and `Comparison` model is designed in a way that should allow developers to update and upgrade comparison semantics of each complex type  (e.g. `models.GPU`s comparison may involve pulling information about GPU benchmarking from external sources and adjusting comparison operators to use it or involve new hardware (e.g. TPUs)). Type comparators that can be updated separately and plugged at runtime into generic comparator operator were conceived for this purpose.

### 2. Structure and organisation

Here is quick overview of the contents of this directory:

* [README](https://gitlab.com/nunet/device-management-service/-/blob/develop/dms/orchestrator/matching/README.md): Current file which is aimed towards developers who wish to use and modify the `orchestrator` functionality. 

* [Comparator.go](https://gitlab.com/nunet/device-management-service/-/blob/develop/dms/orchestrator/matching/Comparator.go): implements generic comparison function for comparing two variables of a custom type; the function detects the type of parameters and uses an appropriate type comparator for comparing them.

* *type comparators*: separate '.go' file for each type-specific `Comparator` that can be picked by the generic comparison function defined in `Comparator.go`:

    * `CapabilityComparator.go`: Comparator for variables of `models.Capability` type; Most of the other listed comparators are fields of `models.Capability`;
	* `ExecutionResourcesComparator`: Comparator for variables of `models.ExecutionResources` type;
	* `ExecutorComparator`: Comparator for variables of `models.Executor` type;
	* `ExecutorsComparator`: Comparator for variables of `models.Executors` type (which is just a Slice of `models.Executor`);
	* `GpuComparator`: Comparator for variables of `models.GPU` type;
	* `GPUsComparator`: Comparator for slices of `model.GPU` typed variables, which is a field in `models.Capability`;
	* `GPUVendorComparator`: Comparator for variables of `models.GPUVendor` type;
	* `JobTypeComparator`: Comparator for variables of `models.JobType` type;
	* `JobTypesComparator`: Comparator for variables of `models.JobTypes` type (which is just a Slice of `models.JobType`);
	* `LibrariesComparator`: Comparator for slices of `model.Library` typed variables, which is a field in `models.Capability`;
	* `LibraryComparator`: Comparator for variables of `models.Library` type;
	* `LocalitiesComparator`: Comparator for slices of `model.Locality` typed variables, which is a field in `models.Capability`;
	* `LocalityComparator`: Comparator for variables of `models.Locality` type;
	* `NumericComparator`: Comparator for all go numeric variables (int, float, etc);

* [./specs/](https://gitlab.com/nunet/device-management-service/-/tree/develop/orchestrator/matching/specs): Directory containing package specifications, including package class diagram.

* [utils.go](https://gitlab.com/nunet/device-management-service/-/blob/develop/dms/orchestrator/matching/utils.go): Utility methods specific to this package only.

### 3. Class Diagram

#### Source

[matching class diagram](https://gitlab.com/nunet/device-management-service/-/blob/develop/dms/orchestrator/matching/specs/class_diagram.puml)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/develop"
!$packageRelativePath = "/dms/orchestrator/matching"
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath
 
!include $packageUrlGitlab/specs/class_diagram.puml
```

### 4. Functionality

#### Generic type comparison

Generic type comparison is implemented by the function `Compare(l, r interface{}, preference ...Preference) models.Comparison`, which takes two variables of the same type, comparison Preferences (_note: note yet implemented_) and outputs a variable of `models.Comparison` type. The function:

* detects the type of parameters supplied to it;
* searches for the appropriate comparator;
* applies the comparator, which, if needed, calls the same function recursively;

So the `Compose` function implements the comparison operator on custom types with a kind of generic tree recursion algorithm. Since each type comparator is plugged in at runtime, the actual logic of each type comparison has to be defined and implemented in type comparators (see below). For this to work, when a new type is introduced and included into `models.Capability` structure (or any other type that is _comparable_ in this sense), its type comparator has to be explicitly defined and registered in `matching.ComparatorMap`.

The comparison operator is generally not symmetric, therefore the semantics of assigning variables to `l` and `r` parameters is important. Current implementation is based on the following semantics:

* *left represent machine capabilities // right represent required capabilities*. 

#### Type Comparators for `models.Capability`

As provided in [Description](#1-description), the main purpose of this package is to compare `models.Capability` typed variables, which is a complex custom type, nesting other complex custom types. Each custom type has its own comparison logic, defined in respective comparator. This section is organized by fields of `models.Capability` type. Comparison of each field may involve one or more `Comparator` function (see `methods and interfaces`).

Currently `models.Capability` is defined as follows:

```

type Capability struct {
	Executors    Executors         `json:"executor" description:"Executor type required for the job (docker, vm, wasm, or others)"`
	JobTypes     JobTypes          `json:"type" description:"Details about type of the job (One time, batch, recurring, long running). Refer to dms.jobs package for jobType data model"`
	Resources    ExecutionResources `json:"resources" description:"Resources required for the job"`
	Libraries    []Library          `json:"libraries" description:"Libraries required for the job"`
	Localities   []Locality         `json:"locality" description:"Preferred localities of the machine for execution"`
	Storage      []Storage          `json:"storage" description:"Preferred storage options that the machine should have"`
	Connectivity Connectivity       `json:"connectivity" description:"Network configuration required"`
	Price        []PriceInformation `json:"price" description:"Pricing information"`
	Time         TimeInformation    `json:"time" description:"Time constraints"`
	KYCs         []KYC
}
```

We may need to update the structure during development or after that with additional fields of information that are needed in order to reason about the jobs requirements and machine capabilities. Such changes should necessarily be accompanied by respective semantics of the comparison, achieved by:

* writing a respective type comparator for each new introduced type;
* registering new type comparator on `matching.ComparatorMap`;
* making sure that the generic comparison operator implemented in `matching.Comparator.Compare()` handles the new type as well as all nested types (if needed correctly);
* finally, update comparison semantics of `matching.CapabilityComparator` as needed in order to take into account new types.

Each comparator is unit-tested via implementation in `Comparator_test.go`. Besides implementing the tests, unit tests also explain the logic and semantics of each type comparator. See [Testing](#5-testing).

##### Executors

`Executors` field determines the available executors on dms (in case `models.Capability` represents available resources of a compute provider) or required executor needed to execute a job (in case `models.Capability` represent requested resource) holds variable of `models.Executors` type which is just a wrapper around `[]model.Executor`. Full comparison requires two functions / comparators:

* `ExecutorsComparator(lraw, rraw interface{}, preference ...Preference) models.Comparison` is the comparator for Executors types. It returns `Equal` if both left and right parameters hold deeply equal variables, `Worse` if left parameter holds contains more executors than the right parameter and `Better` if `Executor`s in available capabilities contain the required executor.

* `ExecutorComparator(lraw, rraw interface{}, preference ...Preference) models.Comparison` is the comparator for `models.Executor` type. It is needed because executor type is defined as enum of `ExecutorType`'s in `models/execution.go`. It is not so complex as the type has only one field therefore this method just passes through the result of wrapped `ExecutorType`.

* `ExecutorTypeComparator(l, r interface{}, preference ...Preference) models.Comparison` is the comparator for `models.ExecutorType` variables. It returns `Equal` if the `l` and `r` variables are equivalent and otherwise an `Error`. For this comparator `Worse` and `Better` returns are undefined. 

##### JobTypes

* `JobTypeComparator(l, r interface{}, preference ...Preference) models.Comparison` returns `Equal` if two Job types are equivalent and `Error` otherwise. Semantics of other comparison values (`Worse` and `Better`) are undefined for this type;

* `JobTypesComparator(lraw, rraw interface{}, preference ...Preference) models.Comparison` is a comparator for `models.JobTypes` variables, which are just wrappers of `[]models.JobType` and captures the need to capabilities that may involve more than one possible `JobType`. If machine capabilities contain oll the required capabilities, then we are good to go. If available capabilities are equal to required capabilities, then the result of comparison is `Equal`. If available JobTypes in capabilities contain required JobTypes, the comparator returns `Better`. If required JobTypes contain available JobTypes, then the comparator returns `Worse`. 


##### Resources

'ExecutionResources' is one of the main fields of the `models.Capability` type as it contains hardware requirements definition.

* `ExecutionResourcesComparator(l, r interface{}, preference ...Preference) models.Comparison` is a comparator for `models.ExecutionResources` type which recursively compares all fields (currently CPU, Memory, Disk, GPUs) and then compares then to each other.

* This type comparator deals with a complex custom type constructed from other complex custom types and therefore uses the functionality provided by `models.ComplexComparison` type.

* Currently we consider that all fields of have to be 'Better' or 'Equal' for the comparison to be 'Better' or 'Equal' else we return 'Worse'.


##### Libraries

'Libraries' field holds available libraries installed on a machine and required libraries by a job. Reasoning about this field involves two data structures and therefore two type comparators.

* `func LibraryComparator(lraw, rraw interface{}, preference ...Preference) models.Comparison` compares two variables of `models.Library` type, which itself has three fields `Name`, `Version` and `Constraint`. The `Constraint` field is conceived for enabling the same type to express both available and required libraries (which often involve constraints like 'more or less' or 'strictly more', etc.). The comparator returns 'Error' if libraries do not match and 'Worse' or 'Better' depending on the provided constraints and library versions.

* `func LibrariesComparator(lraw, rraw interface{}, preference ...Preference) models.Comparison` compares two slices of `models.Library` types, so `[]models.Library` which also can be of different length (i.e. in cases when a job needs two specific libraries, and a machine has 10 libraries installed). This is done internally in the comparator by constructing matrix of pair-wise comparisons and then reasoning on top of them.  

Note, that, contrary to 'Executors' field, 'Libraries' field comparison does not involve a separate type `models.Libraries`, but reasons directly on `[]model.Libraries` structure. So overall we showcase two different ways to implement the comparison logic. The difference between them is that they need different treatment when adjusting `matching.Comparator.Compare()` function: explicit types are resolved automagically with type reflection, given they are correctly registered in `matching.ComparisonMap`, while `[]any` style structures have to be resolved manually (i.e. hardcoded). 

##### Localities

It is not yet clear how are we will be defining localities in our model therefore the comparator of this field was constructed in a similar way to 'Libraries' and involves two data structures and type comparators:

* `func LocalityComparator(lraw interface{}, rraw interface{}, preference ...Preference) models.Comparison` compares two variables of `models.Locality` type, which has two fields `Kind` and `Name`. It may make sense to define the type better (e.g. introduce enums, etc). The current logic of the type is best understood by looking at unit test for `LocalityComparator`.

* `LocalitiesComparator(lraw interface{}, rraw interface{}, preference ...Preference) models.Comparison` compares two slices `[]models.Library` of (potentially) different lengths. 

##### Storage

`TBD`


##### Connectivity

`TBD`

##### Price

`TBD`

##### Time

`TBD`

##### KYCs

`TBD`

**Note: the functionality of the package is currently being developed; please see description for details and developed / proposed aspects.**

### 5. Data Types

Most of the data types used by this package are defined in other packages. Here is the list of important types used by this package, indicating whether or not they are defined here or elsewhere. Note that the `matching` package potentially will deal with comparison of most of the complex types defined within the whole `dms`, therefore it does not make sense to list them all here, but they are mentioned / explained as needed in functionality description of each type comparator (see above).

* `models.Comparison` is an `enum` that defines the result comparison (possible values: `better`, `worse`, `equal` or `error`); the goal of having separate data type for that is to be able to make decisions based on the comparison.

* `models.ComplexComparison` is a helper type for holding `map[string]models.Comparison` which is needed when a comparison of complex custom type depends on the comparisons of its fields and separate logic has to be applied to them.

* `matching.Comparator` defined the type of the function that each comparator has to implement in order to be used by `Compare()`: `type Comparator func(l, r interface{}, preference ...Preference) models.Comparison`.

* `matching.Preference` proposed type for handling functionality of injecting complex type comparison semantics at run time.

* `matching.ComparatorMap` is used for registering (and then retrieving by type name) each type comparator. `Compare()` operator of `matching.Comparator` is currently constructed in a way that it detects input type and then pulls the required type comparator from the comparatorMap by name.

### 6. Testing

All unit tests for the package are implemented in `comparator_test.go`. This file includes at least one test for each type comparator. Currently unit tests are implemented using assertions on manually constructed types. This approach allows to check the logic of comparison, but also limits the input domain of tests. In the future these manually constructed tests maybe augmented by a more elaborate complex type mocking code.

### 7. Proposed Functionality / Requirements 

#### List of issues

All issues that are filed in GitLab related to the implementation of `dms/orchestrator/matching` package can be found below. These include any proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [Capability related issues in 'orchestrator' package](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=opened&search=Capability&or%5Blabel_name%5D%5B%5D=collaboration_group_24%3A%3A16&or%5Blabel_name%5D%5B%5D=collaboration_group_24%3A%3A15&first_page_size=20)
- [All issues mentioning Capability model](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=opened&search=Capability&first_page_size=20)

#### Proposed functionalities

* _Sorting_: based on the `Comparator` model and comparisons between types, we will need to implement sorting of Capabilities, and them Bids or BidRequests. For that, we may need to consider the implementation of `Preference` type (see below) for expressing preferences of sorting, which may be use-case specific. For example, some jobs may need to be processed quickly but can remunerate better while others can be volunteer based or prefer price-efficiency instead of time-efficiency. _Sorting_ logic and `Preference` operators should be constructed in general enough way to handle open-ended list of possible parameters (ideally or in the long term) or clear and efficient process for updating it (in current implementation and in the short term).

* _Upgrading capabilities on demand_: We want to consider a situation when a machine does not have requested capabilities by a job but **can** upgrade them to match the requirements. For now, this functionality may involve Capability fields like 'Libraries'. For this we want to include upgrade method into `matching.Comparator` interface (or somehow into `models.Capability`), so that `matching.CapabilityComparator` could call it if needed. It is important to build this functionality in at interface level + some trivial implementation at least. In the future, it may evolve into quite advanced functionality of the platform, e.g. issuing hardware configuration commands (if installed on configurable hardware clusters) or installing plugins on demand (i.e. 'Executors').

* _Generic Add / Subtract functions_: We need to have a way to add and subtract computing resources (for calculating the Capability needed for several combined jobs and for updating available Capability of a machine after deploying jobs).It my be beneficial to use expand this Capability / Comparator model with generic Add and Subract functionality that would take care of that in a flexible manner. This was originally proposed in [initial job orchestration proposal](https://gitlab.com/nunet/open-api/platform-data-model/-/merge_requests/35#note_1915687497).

#### Data types

- `proposed` `matching.Preference` is a type that would hold preferences of comparison and sorting (e.g. if the preference is price, speed or any other criteria);

- `proposed` `LocalNetworkConfiguration` is a type that should hold detected local network  configuration that is relevant to consider and reason about when comparing capabilities. Currently this could include things like public IPs, whether a machine is behind NAT, type of NATs, possibly constraints imposed by ISPs, etc.

- `proposed` `LocalNetworkTopology` more complex deployments may need a data structure, which considers local network topology of a node / dms -- i.e. for reasoning about speed of connection (as well as capabilities) between neighbors.

### 8. References


#### Related research blogs 

`TBD`

