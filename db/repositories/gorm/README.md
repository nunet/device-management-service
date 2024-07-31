# gorm

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

This sub package contains Gorm implementation of the database interfaces.

### 2. Structure and organisation

Here is quick overview of the contents of this pacakge:

* [README](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/gorm/README.md): Current file which is aimed towards developers who wish to use and modify the database functionality. 

* [generic_repository](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/gorm/generic_repository.go): This file implements the methods of `GenericRepository` interface.

* [generic_entity_repository](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/gorm/generic_entity_repository.go): This file implements the methods of `GenericEntityRepository` interface.

* [deployment](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/gorm/deployment.go): This file contains implementation of `DeploymentRequestFlatRepository` interface. 

* [elk_stats](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/gorm/elk_stats.go): This file contains implementation of `RequestTrackerRepository` interface.

* [firecracker](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/gorm/firecracker.go): This file contains implementation of `VirtualMachineRepository` interface.

* [machine](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/gorm/machine.go): This file contains implementation of interfaces defined in [machine.go](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/machine.go).  

* [utils](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/gorm/utils.go): This file contains utility functions with respect to Gorm implementation.

All files with `*_test.go` naming convention contain unit tests with respect to the specific implementation.

### 3. Functionality

#### GenericRepository

##### NewGenericRepository

* signature: `NewGenericRepository[T repositories.ModelType](db *gorm.DB) -> repositories.GenericRepository[T]` <br/>

* input: Gorm Database object <br/>

* output: Repository of type `dms.database.gorm.GenericRepositoryGORM` <br/>

`NewGenericRepository` function creates a new instance of `GenericRepositoryGORM` struct. It initializes and returns a repository with the provided GORM database. 

##### Interface Methods

See `db` package [readme](https://gitlab.com/nunet/device-management-service/-/tree/develop/db/repositories?ref_type=heads#genericrepository-interface) for methods of `GenericRepository` interface.

#### GenericEntityRepository

##### NewGenericEntityRepository

* signature: `NewGenericEntityRepository[T repositories.ModelType](db *gorm.DB) -> repositories.GenericEntityRepository[T]` 

* input #1: Gorm Database object <br/>

* output: Repository of type `dms.database.gorm.GenericEntityRepositoryGORM` <br/>

`NewGenericEntityRepository` creates a new instance of `GenericEntityRepositoryGORM` struct. It initializes and returns a repository with the provided GORM database.

##### Interface Methods

See `db` package [readme](https://gitlab.com/nunet/device-management-service/-/tree/develop/db/repositories?ref_type=heads#genericentityrepository-interface) for methods of `GenericEntityRepository` interface.


### 4. Data Types

- `GenericRepositoryGORM`: This is a generic repository implementation using GORM as an ORM.

```
type GenericRepositoryGORM[T repositories.ModelType] struct {
	db *gorm.DB
}
```

- `GenericEntityRepositoryGORM`: This is a generic single entity repository implementation using GORM as an ORM

```
type GenericEntityRepositoryGORM[T repositories.ModelType] struct {
	db *gorm.DB // db is the GORM database instance.
}
```

For other data types refer to `db` package readme. 

### 5. Testing

Refer to `*_test.go` files for unit tests of different functionalities.

### 6. Proposed Functionality / Requirements 

#### List of issues

All issues that are related to the implementation of `db` package can be found below. These include any proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [db package implementation](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=opened&label_name%5B%5D=collaboration_group_24%3A%3A36&first_page_size=20)

### 7. References

The DMS is being refactored and augmented with several new functionalities. The proposed class diagram can be found here:
- [Class Diagram - Source](https://gitlab.com/nunet/device-management-service/-/blob/develop/specs/classDiagrams/dms-global.mermaid)
- [Class Diagram - Rendered](https://gitlab.com/nunet/device-management-service/-/blob/develop/specs/classDiagrams/dms-global.svg)
