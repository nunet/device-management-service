# gorm

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

This sub package contains Gorm implementation of the database interfaces.

### Structure and Organisation

Here is quick overview of the contents of this pacakge:

* [README](https://gitlab.com/nunet/device-management-service/-/blob/main/db/repositories/gorm/README.md): Current file which is aimed towards developers who wish to use and modify the database functionality. 

* [generic_repository](https://gitlab.com/nunet/device-management-service/-/blob/main/db/repositories/gorm/generic_repository.go): This file implements the methods of `GenericRepository` interface.

* [generic_entity_repository](https://gitlab.com/nunet/device-management-service/-/blob/main/db/repositories/gorm/generic_entity_repository.go): This file implements the methods of `GenericEntityRepository` interface.

* [deployment](https://gitlab.com/nunet/device-management-service/-/blob/main/db/repositories/gorm/deployment.go): This file contains implementation of `DeploymentRequestFlat` interface. 

* [elk_stats](https://gitlab.com/nunet/device-management-service/-/blob/main/db/repositories/gorm/elk_stats.go): This file contains implementation of `RequestTracker` interface.

* [firecracker](https://gitlab.com/nunet/device-management-service/-/blob/main/db/repositories/gorm/firecracker.go): This file contains implementation of `VirtualMachine` interface.

* [machine](https://gitlab.com/nunet/device-management-service/-/blob/main/db/repositories/gorm/machine.go): This file contains implementation of interfaces defined in [machine.go](https://gitlab.com/nunet/device-management-service/-/blob/main/db/repositories/machine.go).  

* [utils](https://gitlab.com/nunet/device-management-service/-/blob/main/db/repositories/gorm/utils.go): This file contains utility functions with respect to Gorm implementation.

All files with `*_test.go` naming convention contain unit tests with respect to the specific implementation.

### Class Diagram

The class diagram for the `gorm` package is shown below.

#### Source file

[gorm Class diagram](https://gitlab.com/nunet/device-management-service/-/blob/main/db/repositories/gorm/specs/class_diagram.puml)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/main"
!$packageRelativePath = "/db/repositories/gorm"
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath
 
!include $packageUrlGitlab/specs/class_diagram.puml
```

### Functionality

#### GenericRepository

##### NewGenericRepository

* signature: `NewGenericRepository[T repositories.ModelType](db *gorm.DB) -> repositories.GenericRepository[T]` <br/>

* input: Gorm Database object <br/>

* output: Repository of type `db.gorm.GenericRepositoryGORM` <br/>

`NewGenericRepository` function creates a new instance of `GenericRepositoryGORM` struct. It initializes and returns a repository with the provided GORM database. 

##### Interface Methods

See `db` package [readme](https://gitlab.com/nunet/device-management-service/-/tree/main/db/repositories?ref_type=heads#genericrepository-interface) for methods of `GenericRepository` interface.

#### GenericEntityRepository

##### NewGenericEntityRepository

* signature: `NewGenericEntityRepository[T repositories.ModelType](db *gorm.DB) -> repositories.GenericEntityRepository[T]` 

* input #1: Gorm Database object <br/>

* output: Repository of type `db.gorm.GenericEntityRepositoryGORM` <br/>

`NewGenericEntityRepository` creates a new instance of `GenericEntityRepositoryGORM` struct. It initializes and returns a repository with the provided GORM database.

##### Interface Methods

See `db` package [readme](https://gitlab.com/nunet/device-management-service/-/tree/main/db/repositories?ref_type=heads#genericentityrepository-interface) for methods of `GenericEntityRepository` interface.


### Data Types

- `db.gorm.GenericRepositoryGORM`: This is a generic repository implementation using GORM as an ORM.

```
type GenericRepositoryGORM[T repositories.ModelType] struct {
	db *gorm.DB
}
```

- `db.gorm.GenericEntityRepositoryGORM`: This is a generic single entity repository implementation using GORM as an ORM

```
type GenericEntityRepositoryGORM[T repositories.ModelType] struct {
	db *gorm.DB // db is the GORM database instance.
}
```

For other data types refer to `db` package readme. 

### Testing

Refer to `*_test.go` files for unit tests of different functionalities.

### Proposed Functionality / Requirements 

#### List of issues

All issues that are related to the implementation of `db` package can be found below. These include any proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [db package implementation](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=opened&label_name%5B%5D=collaboration_group_24%3A%3A36&first_page_size=20)

### References