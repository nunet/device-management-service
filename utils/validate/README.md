# validate

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

### Description `TBD`

Utils specifically used for the validation of different types.

### Structure and Organisation

Here is quick overview of the contents of this directory:

- [README](https://gitlab.com/nunet/device-management-service/-/blob/main/utils/validate/README.md): Current file which is aimed towards developers who wish to use and modify the package functionality.

- [numerics](https://gitlab.com/nunet/device-management-service/-/blob/main/utils/validate/numerics.go): This file contains method for conversion of numerical data to `float64` type.

- [strings](https://gitlab.com/nunet/device-management-service/-/blob/main/utils/validate/strings.go): This file contains method for validation check of data types.

- [specs](https://gitlab.com/nunet/device-management-service/-/blob/main/utils/validate/specs): This folder contains the class diagram of the package.


### Class Diagram

#### Source File

[validate Class Diagram](https://gitlab.com/nunet/device-management-service/-/blob/main/utils/validate/specs/class_diagram.puml)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/main"
!$packageRelativePath = "/utils/validate"
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath
 
!include $packageUrlGitlab/specs/class_diagram.puml
```

### Functionality

This package contains helper methods that perform validation check for different data types. Refer to `strings.go` file for more details.

### Data Types

This package does not define any new data types.

### Testing

Unit tests for the functionality are defined in files with `*_test.go` in their names.

### Proposed Functionality / Requirements

List of issues related to the implementation of the `utils` package can be found below. These include proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [utils Package Issues]() `TBD`

### References
