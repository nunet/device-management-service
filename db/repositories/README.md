# db

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

The db package contains the configuration and functionality of database used by the DMS

### 2. Structure and organisation

Here is quick overview of the contents of this pacakge:

_Files_ 

* [README](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/README.md): Current file which is aimed towards developers who wish to use and modify the database functionality. 

* [generic_repository](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/generic_repository.go): This file defines the interface defining the main methods for db pacakge. It is designed using generic types and can be adapted to specific data type as needed.

* [generic_entity_repository](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/generic_entity_repository.go): This file contains the interface for those databases which will hold only a single record.

* [deployment](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/deployment.go): This file specifies a database interface having `models.DeploymentRequestFlat` data type.

* [elk_stats](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/elk_stats.go): This file specifies a database interface having `models.RequestTracker` data type.

* [errors](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/errors.go): This file specifies the different types of errors.

* [firecracker](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/firecracker.go): This file specifies a database interface having `models.VirtualMachine` data type.

* [machine](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/machine.go): This file defines database interfaces of various data types. 

* [onboarding](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/onboarding.go): This file specifies a database interface having `models.LogBinAuth` data type.

* [utils](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/utils.go): This file contains some utility functions with respect to database operations.

* [utils_test](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/utils_test.go): This file contains unit tests for functions defined in [utils.go](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/utils.go) file 

_Subpackages_

* [gorm](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/gorm): This folder contains SQlite database implementation using gorm.

* [clover](https://gitlab.com/nunet/device-management-service/-/blob/develop/db/repositories/clover): This folder contains CloverDB database implementation.


### 3. Functionality

There are two types of interfaces defined to cover database operations:

1. `GenericRepository` 

2. `GenericEntityRepository`

These interfaces are described below.

#### GenericRepository Interface

`GenericRepository` interface defines basic CRUD operations and standard querying methods. It is defined with generic data types. This allows it to be used for any data type. 

- interface definition: `type GenericRepository[T ModelType] interface`

The methods of `GenericRepository` are as follows:

##### Create

* signature: `Create(ctx context.Context, data T) -> (T, error)` <br/>

* input #1: Go context <br/>

* input #2: Data to be added to the database. It should be of type used to initialize the repository <br/>

* output (success): Data type used to initialize the repository <br/>

* output (error): error message

`Create` function adds a new record to the database. 

##### Get

* signature: `Get(ctx context.Context, id interface{}) -> (T, error)` <br/>

* input #1: Go context <br/>

* input #2: Identifier of the record. Can be any data type <br/>

* output (success): Data with the identifier provided. It is of type used to initialize the repository <br/>

* output (error): error message

`Get` function retrieves a record from the database by its identifier. 

##### Update

* signature: `Update(ctx context.Context, id interface{}, data T) -> (T, error)` <br/>

* input #1: Go context <br/>

* input #2: Identifier of the record. Can be any data type <br/>

* input #3: New data of type used to initialize the repository  <br/>

* output (success): Updated record of type used to initialize the repository <br/>

* output (error): error message

`Update` function modifies an existing record in the database using its identifier.

##### Delete

* signature: `Delete(ctx context.Context, id interface{}) -> error` <br/>

* input #1: Go context <br/>

* input #2: Identifier of the record. Can be any data type <br/>

* output (success): None <br/>

* output (error): error message

`Delete` function deletes an existing record in the database using its identifier.

##### Find

* signature: `Find(ctx context.Context, query Query[T]) -> (T, error)` <br/>

* input #1: Go context <br/>

* input #2: Query of type `dms.database.query` <br/>

* output (success): Result of query having the data type used to initialize the repository <br/>

* output (error): error message

`Find` function retrieves a single record from the database based on a query.

##### FindAll

* signature: `FindAll(ctx context.Context, query Query[T]) -> ([]T, error)` <br/>

* input #1: Go context <br/>

* input #2: Query of type `dms.database.query` <br/>

* output (success): Lists of records based on query result. The data type of each record will be what was used to initialize the repository <br/>

* output (error): error message

`FindAll` function retrieves multiple records from the database based on a query.

##### GetQuery
* signature: `GetQuery() -> Query[T]` <br/>

* input: None <br/>

* output: Query of type `dms.database.query`<br/>

`GetQuery` function returns an empty query instance for the repository's type.

#### GenericEntityRepository Interface

`GenericEntityRepository` defines basic CRUD operations for repositories handling a single record. It is defined with generic data types. This allows it to be used for any data type. 

- interface definition: `type GenericEntityRepository[T ModelType] interface`

The methods of `GenericEntityRepository` are as follows:

##### Save

* signature: `Save(ctx context.Context, data T) -> (T, error)` <br/>

* input #1: Go context <br/>

* input #2: Data to be saved of type used to initialize the database <br/>

* output (success): Updated record of type used to initialize the repository <br/>

* output (error): error message

`Save` function adds or updates a single record in the repository

##### Get

* signature: `Get(ctx context.Context) -> (T, error)` <br/>

* input: Go context <br/>

* output (success): Record of type used to initialize the repository <br/>

* output (error): error message

`Get` function retrieves the single record from the database.

##### Clear

* signature: `Clear(ctx context.Context) -> error` <br/>

* input: Go context <br/>

* output (success): None <br/>

* output (error): error 

`Clear` function removes the record and its history from the repository.

##### History

* signature: `History(ctx context.Context, qiery Query[T]) -> ([]T, error)` <br/>

* input #1: Go context <br/>

* input #2: query of type `dms.database.query` <br/>

* output (success):List of records of repository's type <br/>

* output (error): error 

`History` function retrieves previous records from the repository which satisfy the query conditions.

##### GetQuery

* signature: `GetQuery() -> Query[T]` <br/>

* input: None <br/>

* output: New query of type `dms.database.query` <br/>

`GetQuery` function returns an empty query instance for the repository's type.

### 4. Data Types

- `db.Query`: This contains parameters related to a query that is passed to the database.

```
type Query[T any] struct {
	Instance   T                // Instance is an optional object of type T used to build conditions from its fields.
	Conditions []QueryCondition // Conditions represent the conditions applied to the query.
	SortBy     string           // SortBy specifies the field by which the query results should be sorted.
	Limit      int              // Limit specifies the maximum number of results to return.
	Offset     int              // Offset specifies the number of results to skip before starting to return data.
}
```

- `db.QueryCondition`: This contains parameters defining a query condition.

```
type QueryCondition struct {
	Field    string      // Field specifies the database or struct field to which the condition applies.
	Operator string      // Operator defines the comparison operator (e.g., "=", ">", "<").
	Value    interface{} // Value is the expected value for the given field.
}
```

`GenericRepository` has been initialised for the following data types:

- `models.DeploymentRequestFlat`
- `models.VirtualMachine`
- `models.PeerInfo`
- `models.Machine`
- `models.Services`
- `models.ServiceResourceRequirements`
- `models.Connection`
- `models.ElasticToken`

`GenericEntityRepository` has been initialised for the following data types:
- `models.FreeResources`
- `models.AvailableResources`
- `models.Libp2pInfo`
- `models.MachineUUID`
- `models.LogBinAuth`


### 5. Testing

The unit tests for utility functions are defined in `utils_test.go`. Refer to `*_test.go` files for unit tests of various implementations covered in subpackages.

### 6. Proposed Functionality / Requirements 

#### List of issues

All issues that are related to the implementation of `db` package can be found below. These include any proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [db package implementation](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=opened&label_name%5B%5D=collaboration_group_24%3A%3A36&first_page_size=20)

### 7. References

The DMS is being refactored and augmented with several new functionalities. The proposed class diagram can be found here:
- [Class Diagram - Source](https://gitlab.com/nunet/device-management-service/-/blob/develop/specs/classDiagrams/dms-global.mermaid)
- [Class Diagram - Rendered](https://gitlab.com/nunet/device-management-service/-/blob/develop/specs/classDiagrams/dms-global.svg)