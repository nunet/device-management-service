# internal

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

This package contains all code that is very specific to the whole of the dms, which will not be imported by any other packages and used only on the running instance of dms (like config and background task).

### 2. Structure and organisation

Here is quick overview of the contents of this pacakge:

* [README](README.md): Current file which is aimed towards developers who wish to use and modify the package functionality. 

* [init](init.go): This file handles controlled shutdown and initializes OpenTelemetry-based Zap logger.

* [websocket](websocket.go): This file contains communication protocols for a websocket server including message handling and command execution.

_subpackages_
* [config](config): This sub-package contains the configuration related data for the whole dms.

* [background_tasks](background_tasks): This sub-package contains functionality that runs in the background.

### 3. Functionality

`TBD`

### 4. Data Types

- `WebSocketConnection`

```
// WebSocketConnection is pointer to gorilla/websocket.Conn
type WebSocketConnection struct {
	*websocket.Conn
}
```

- `Command`

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

### 5. Testing

`TBD`

### 6. Proposed Functionality / Requirements 

#### List of issues

All issues that are related to the implementation of `internal` package can be found below. These include any proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [internal package implementation]() `TBD`


### 7. References

The DMS is being refactored and augmented with several new functionalities. The proposed class diagram can be found here:
- [Class Diagram - Source](https://gitlab.com/nunet/device-management-service/-/blob/develop/specs/classDiagrams/dms-global.mermaid)
- [Class Diagram - Rendered](https://gitlab.com/nunet/device-management-service/-/blob/develop/specs/classDiagrams/dms-global.svg)