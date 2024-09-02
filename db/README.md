# db

- [Project README](https://gitlab.com/nunet/device-management-service/-/blob/main/README.md)
- [Release/Build Status](https://gitlab.com/nunet/device-management-service/-/releases)
- [Changelog](https://gitlab.com/nunet/device-management-service/-/blob/main/CHANGELOG.md)
- [License](https://www.apache.org/licenses/LICENSE-2.0.txt)
- [Contribution Guidelines](https://gitlab.com/nunet/device-management-service/-/blob/main/CONTRIBUTING.md)
- [Code of Conduct](https://gitlab.com/nunet/device-management-service/-/blob/main/CODE_OF_CONDUCT.md)
- [Secure Coding Guidelines](https://gitlab.com/nunet/team-processes-and-guidelines/-/blob/main/secure_coding_guidelines/README.md)

## Table of Contents

1. [Description](#1-description)
2. [Structure and Organisation](#2-structure-and-organisation)
3. [Class Diagram](#3-class-diagram)
4. [Package Specification](#4-package-specification)

## Specification

### 1. Description
This package defines the local database functionality for the Device Management Service (DMS). Currently two repository structures have been implemented:

- `gorm`: which is a [SQlite](https://www.sqlite.org/) database implementation.

- `clover`: which is a `NoSQL` or document oriented database implementation. 

### 2. Structure and Organisation

Here is quick overview of the contents of this pacakge:

* [README](https://gitlab.com/nunet/device-management-service/-/blob/main/db/README.md): Current file which is aimed towards developers who wish to use and modify the package functionality. 

* [db.go](https://gitlab.com/nunet/device-management-service/-/blob/main/db/db.go): This file defines the method which opens an SQlite database at a path set by the config parameter `work_dir`, applies migration and returns the `db` instance.

* [repositories](https://gitlab.com/nunet/device-management-service/-/blob/main/db/repositories): This folder contains the sub-packages of the `db` package.

* [specs](https://gitlab.com/nunet/device-management-service/-/blob/main/db/specs): This folder contains the class diagram of the package.

### 3. Class Diagram

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

### 4. Package Specification
Refer to the [README](https://gitlab.com/nunet/device-management-service/-/blob/main/db/repositories/README.md) file defined in the repositories folder for specification of the package.