# internal

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

This package contains all code that is very specific to the whole of the dms, which will not be imported by any other packages and used only on the running instance of dms (like config and background task).

### Structure and Organisation

Here is quick overview of the contents of this pacakge:

* [README](https://gitlab.com/nunet/device-management-service/-/tree/main/internal/README.md): Current file which is aimed towards developers who wish to use and modify the package functionality. 

* [init](https://gitlab.com/nunet/device-management-service/-/tree/main/internal/init.go): This file handles controlled shutdown and initializes OpenTelemetry-based Zap logger.

* [websocket](https://gitlab.com/nunet/device-management-service/-/tree/main/internal/websocket.go): This file contains communication protocols for a websocket server including message handling and command execution.

_subpackages_
* [config](https://gitlab.com/nunet/device-management-service/-/tree/main/internal/config): This sub-package contains the configuration related data for the whole dms.

* [background_tasks](https://gitlab.com/nunet/device-management-service/-/tree/main/internal/background_tasks): This sub-package contains functionality that runs in the background.

### Class Diagram

#### Source

[internal class diagram](https://gitlab.com/nunet/device-management-service/-/blob/main/internal/specs/class_diagram.puml)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/main"
!$packageRelativePath = "/internal"
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath
 
!include $packageUrlGitlab/specs/class_diagram.puml
```

### Functionality

`TBD`

### Data Types

- `internal.WebSocketConnection`

```
// WebSocketConnection is pointer to gorilla/websocket.Conn
type WebSocketConnection struct {
	*websocket.Conn
}
```

- `internal.Command`

```
// Command represents a command to be executed
type Command struct {
	Command string
	NodeID  string // ID of the node where command will be executed
	Result  string
	Conn    *WebSocketConnection
}
```

**Note: The data types are expected to change during refactoring of DMS**

### Testing

`TBD`

### Proposed Functionality / Requirements 

#### List of issues

All issues that are related to the implementation of `internal` package can be found below. These include any proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [internal package implementation]() `TBD`


### References

