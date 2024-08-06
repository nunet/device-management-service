# clover

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

This sub package contains CloverDB implementation of the database interfaces.

### 2. Structure and organisation

Here is quick overview of the contents of this pacakge:

* [README](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/clover/README.md): Current file which is aimed towards developers who wish to use and modify the database functionality. 

* [generic_repository](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/clover/generic_repository.go): This file implements the methods of `GenericRepository` interface.

* [generic_entity_repository](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/clover/generic_entity_repository.go): This file implements the methods of `GenericEntityRepository` interface.

* [deployment](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/clover/deployment.go): This file contains implementation of `DeploymentRequestFlatRepository` interface. 

* [elk_stats](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/clover/elk_stats.go): This file contains implementation of `RequestTrackerRepository` interface.

* [firecracker](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/clover/firecracker.go): This file contains implementation of `VirtualMachineRepository` interface.

* [machine](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/clover/machine.go): This file contains implementation of interfaces defined in [machine.go](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/machine.go).  

* [utils](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/clover/utils.go): This file contains utility functions with respect to clover implementation.

All files with `*_test.go` naming convention contain unit tests with respect to the specific implementation.

### 3. Class Diagram

The class diagram for the `clover` package is shown below.

#### Source file

[clover Class diagram](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/clover/specs/class_diagram.puml)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/develop"
!$packageRelativePath = "/db/repositories/clover"
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath
 
!include $packageUrlGitlab/specs/class_diagram.puml
```

### 4. Functionality

#### GenericRepository

##### NewGenericRepository

* signature: `NewGenericRepository[T repositories.ModelType](db *clover.DB) -> repositories.GenericRepository[T]` <br/>

* input: clover Database object <br/>

* output: Repository of type `dms.database.clover.GenericRepositoryclover` <br/>

`NewGenericRepository` function creates a new instance of `GenericRepositoryclover` struct. It initializes and returns a repository with the provided clover database. 

##### Interface Methods

See `db` package [readme](https://gitlab.com/nunet/device-management-service/-/tree/develop/db/repositories?ref_type=heads#genericrepository-interface) for methods of `GenericRepository` interface

##### query

* signature: `query(includeDeleted bool) -> *clover_q.Query` <br/>

* input: boolean value to choose whether to include deleted records <br/>

* output: [CloverDB query object](https://pkg.go.dev/github.com/ostafen/clover/v2/query#Query) <br/>

`query` function creates and returns a new CloverDB [Query](https://pkg.go.dev/github.com/ostafen/clover/v2/query#Query) object. Input value of `False` will add a condition to exclude the deleted records.

##### queryWithID

* signature: `queryWithID(id interface{}, includeDeleted bool) -> *clover_q.Query` <br/>

* input #1: identifier <br/>

* input #2: boolean value to choose whether to include deleted records <br/>

* output: [CloverDB query object](https://pkg.go.dev/github.com/ostafen/clover/v2/query#Query) <br/>

`queryWithID` function creates and returns a new CloverDB [Query](https://pkg.go.dev/github.com/ostafen/clover/v2/query#Query) object. The provided inputs are added to query conditions. The identifier will be compared to primary key field value of the repository. 

Providing `includeDeleted` as `False` will add a condition to exclude the deleted records.

#### GenericEntityRepository

##### NewGenericEntityRepository

* signature: `NewGenericEntityRepository[T repositories.ModelType](db *clover.DB) repositories.GenericEntityRepository[T]` 

* input: clover Database object <br/>

* output: Repository of type `dms.database.clover.GenericEntityRepositoryclover` <br/>

`NewGenericEntityRepository` creates a new instance of `GenericEntityRepositoryclover` struct. It initializes and returns a repository with the provided clover database instance and name of the collection in the database.

##### Interface Methods

See `db` package [readme](https://gitlab.com/nunet/device-management-service/-/tree/develop/db/repositories?ref_type=heads#genericentityrepository-interface) for methods of `GenericEntityRepository` interface.

##### query

* signature: `query() -> *clover_q.Query` <br/>

* input: None <br/>

* output: [CloverDB query object](https://pkg.go.dev/github.com/ostafen/clover/v2/query#Query) <br/>

`query` function creates and returns a new CloverDB [Query](https://pkg.go.dev/github.com/ostafen/clover/v2/query#Query) object. 

### 5. Data Types

- `db.clover.GenericRepositoryClover`: This is a generic repository implementation using clover as an ORM

```
type GenericRepositoryClover[T repositories.ModelType] struct {
	db         *clover.DB // db is the Clover database instance.
	collection string     // collection is the name of the collection in the database.
}
```

- `db.clover.GenericEntityRepositoryClover`: This is a generic single entity repository implementation using clover as an ORM

```
type GenericEntityRepositoryClover[T repositories.ModelType] struct {
	db         *clover.DB // db is the Clover database instance.
	collection string     // collection is the name of the collection in the database.
}
```

For other data types refer to `db` package readme. 

### 6. Testing

Refer to `*_test.go` files for unit tests of different functionalities.

### 7. Proposed Functionality / Requirements 

#### List of issues

All issues that are related to the implementation of `db` package can be found below. These include any proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [db package implementation](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=opened&label_name%5B%5D=collaboration_group_24%3A%3A36&first_page_size=20)

### 8. References
