# clover

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
4. [Implementation Details](#implementation-details)
5. [Testing](#testing)

## Specification

### Description
The `clover` package provides a NoSQL document-oriented database implementation of the repository interfaces defined in the parent `db` package. It uses CloverDB as the underlying storage engine and implements both the `GenericRepository` and `GenericEntityRepository` interfaces.

### Structure and Organisation

Here is a quick overview of the contents of this package:

* [README.md](https://gitlab.com/nunet/device-management-service/-/blob/main/db/clover/README.md): Current file which is aimed towards developers who wish to use and modify the CloverDB implementation.

* [clover.go](https://gitlab.com/nunet/device-management-service/-/blob/main/db/clover/clover.go): Contains the CloverDB connection and initialization logic.

* [clover_test.go](https://gitlab.com/nunet/device-management-service/-/blob/main/db/clover/clover_test.go): Contains tests for the CloverDB connection and initialization.

* [generic_repository.go](https://gitlab.com/nunet/device-management-service/-/blob/main/db/clover/generic_repository.go): Implements the `GenericRepository` interface for CloverDB, providing CRUD operations and query functionality.

* [generic_repo_test.go](https://gitlab.com/nunet/device-management-service/-/blob/main/db/clover/generic_repo_test.go): Contains tests for the `GenericRepository` implementation.

* [generic_entity_repository.go](https://gitlab.com/nunet/device-management-service/-/blob/main/db/clover/generic_entity_repository.go): Implements the `GenericEntityRepository` interface for CloverDB, providing operations for repositories handling a single record.

* [entity_repo_test.go](https://gitlab.com/nunet/device-management-service/-/blob/main/db/clover/entity_repo_test.go): Contains tests for the `GenericEntityRepository` implementation.

* [utils.go](https://gitlab.com/nunet/device-management-service/-/blob/main/db/clover/utils.go): Contains utility functions specific to the CloverDB implementation.

* [utils_test.go](https://gitlab.com/nunet/device-management-service/-/blob/main/db/clover/utils_test.go): Contains tests for the utility functions.

* [log.go](https://gitlab.com/nunet/device-management-service/-/blob/main/db/clover/log.go): Contains logging functionality for the CloverDB implementation.

* [specs](https://gitlab.com/nunet/device-management-service/-/blob/main/db/clover/specs): This folder contains the class diagram of the package.

### Class Diagram

The class diagram for the `clover` package is shown below.

#### Source file

[clover Class diagram](https://gitlab.com/nunet/device-management-service/-/blob/main/db/clover/specs/class_diagram.puml)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/main"
!$packageRelativePath = "/db/clover"
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath
 
!include $packageUrlGitlab/specs/class_diagram.puml
```

### Implementation Details

#### CloverDB Repository

The CloverDB implementation provides a document-oriented storage solution with the following key features:

1. **Document Storage**: Data is stored as JSON documents in collections.

2. **Type Safety**: Uses Go generics to ensure type safety when working with different data types.

3. **Query Capabilities**: Supports complex queries using the query condition functions (EQ, GT, GTE, LT, LTE, IN, LIKE).

4. **Transactions**: Provides transaction support for atomic operations.

5. **History Tracking**: For `GenericEntityRepository`, maintains a history of changes to records.

#### Key Components

1. **CloverRepository**: The main implementation of `GenericRepository` that handles CRUD operations and queries.

2. **CloverEntityRepository**: The implementation of `GenericEntityRepository` for single-record repositories.

3. **CloverDB Connection**: Manages the connection to the underlying CloverDB database.

### Testing

The package includes comprehensive tests for all components:

* **Unit Tests**: Tests for individual functions and methods.

* **Integration Tests**: Tests that verify the interaction between different components.

* **Repository Tests**: Tests that verify the repository implementations meet the interface requirements.

To run the tests for this package, use the following command:

```bash
go test -v ./db/clover/...
```