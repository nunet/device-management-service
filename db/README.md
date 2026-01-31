# db

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
4. [Package Specification](#package-specification)

## Specification

### Description
This package defines the local database functionality for the Device Management Service (DMS). It provides a generic repository pattern for database operations with a `clover` implementation, which is a `NoSQL` or document oriented database implementation.

The package uses generic interfaces to support different data types and provides query condition functions for building database queries.

### Structure and Organisation

Here is quick overview of the contents of this pacakge:

* [README](https://gitlab.com/nunet/device-management-service/-/blob/main/db/README.md): Current file which is aimed towards developers who wish to use and modify the package functionality.

* [repositories](https://gitlab.com/nunet/device-management-service/-/blob/main/db/repositories): This folder contains the core interfaces and implementations for database operations.
  * [generic_repository.go](https://gitlab.com/nunet/device-management-service/-/blob/main/db/repositories/generic_repository.go): Defines the `GenericRepository` interface with basic CRUD operations and standard querying methods using generic types.
  * [generic_entity_repository.go](https://gitlab.com/nunet/device-management-service/-/blob/main/db/repositories/generic_entity_repository.go): Defines the `GenericEntityRepository` interface for repositories handling a single record.
  * [conditions.go](https://gitlab.com/nunet/device-management-service/-/blob/main/db/repositories/conditions.go): Contains query condition functions (EQ, GT, GTE, LT, LTE, IN, LIKE) for building database queries.
  * [conditions_test.go](https://gitlab.com/nunet/device-management-service/-/blob/main/db/repositories/conditions_test.go): Contains unit tests for the query condition functions.
  * [utils.go](https://gitlab.com/nunet/device-management-service/-/blob/main/db/repositories/utils.go): Contains utility functions for database operations.
  * [utils_test.go](https://gitlab.com/nunet/device-management-service/-/blob/main/db/repositories/utils_test.go): Contains unit tests for the utility functions.
  * [errors.go](https://gitlab.com/nunet/device-management-service/-/blob/main/db/repositories/errors.go): Defines error types for database operations.

* [clover](https://gitlab.com/nunet/device-management-service/-/blob/main/db/clover): Contains the CloverDB (NoSQL) implementation of the repository interfaces.
  * [generic_repository.go](https://gitlab.com/nunet/device-management-service/-/blob/main/db/clover/generic_repository.go): Implements the `GenericRepository` interface for CloverDB.
  * [generic_entity_repository.go](https://gitlab.com/nunet/device-management-service/-/blob/main/db/clover/generic_entity_repository.go): Implements the `GenericEntityRepository` interface for CloverDB.
  * [clover.go](https://gitlab.com/nunet/device-management-service/-/blob/main/db/clover/clover.go): Contains the CloverDB connection and initialization logic.
  * Various test files for ensuring functionality works as expected.

* [specs](https://gitlab.com/nunet/device-management-service/-/blob/main/db/specs): This folder contains the class diagram of the package.

### Class Diagram

The class diagram for the `db` package is shown below.

#### Source file

[db Class diagram](https://gitlab.com/nunet/device-management-service/-/blob/main/db/specs/class_diagram.puml)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/main"
!$packageRelativePath = "/db"
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath
 
!include $packageUrlGitlab/specs/class_diagram.puml
```

### Package Specification
Refer to the [README](https://gitlab.com/nunet/device-management-service/-/blob/main/db/repositories/README.md) file defined in the repositories folder for detailed specification of the package.

#### Key Interfaces

1. **GenericRepository**: Defines basic CRUD operations (Create, Read, Update, Delete) and standard querying methods using generic types, allowing it to be used with any data type.

2. **GenericEntityRepository**: Defines operations for repositories handling a single record, with methods for saving, retrieving, clearing, and tracking history.

#### Query Conditions

The package provides functions to create query conditions for database operations:

- `EQ`: Creates equality comparison (`field = value`)
- `GT`: Creates greater-than comparison (`field > value`)
- `GTE`: Creates greater-than-or-equal comparison (`field >= value`)
- `LT`: Creates less-than comparison (`field < value`)
- `LTE`: Creates less-than-or-equal comparison (`field <= value`)
- `IN`: Creates IN comparison (`field IN values`)
- `LIKE`: Creates LIKE comparison for pattern matching (`field LIKE pattern`)
