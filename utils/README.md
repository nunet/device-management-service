# utils

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
This package contains utility tools and functionalities used by other packages

### Structure and Organisation

Here is quick overview of the contents of this directory:

- [README](https://gitlab.com/nunet/device-management-service/-/blob/main/utils/README.md): Current file which is aimed towards developers who wish to use and modify the package functionality.

- [blockchain](https://gitlab.com/nunet/device-management-service/-/blob/main/utils/blockchain.go): This file contains methods and data types related to interaction with blockchain.

- [file_system](https://gitlab.com/nunet/device-management-service/-/blob/main/utils/file_system.go): This file contains a method to retrieve the size of the volume.

- [init](https://gitlab.com/nunet/device-management-service/-/blob/main/utils/init.go): This file initializes an Open Telemetry logger for this package. It also defines constants to reflect the status of transaction. 

- [network](https://gitlab.com/nunet/device-management-service/-/blob/main/utils/network.go): This file contains helper methods for DMS API calls and responses.

- [progress_io](https://gitlab.com/nunet/device-management-service/-/blob/main/utils/progress_io.go): This file defines wrapper functions for readers and writers with progress tracking capabilities.

- [syncmap](https://gitlab.com/nunet/device-management-service/-/blob/main/utils/syncmap.go): This file defines a `SyncMap` type which is a thread-safe version of the standard Go `map` with strongly-typed methods and functions for managing key-value pairs concurrently.

- [utils](https://gitlab.com/nunet/device-management-service/-/blob/main/utils/utils.go): This file contains various utility functions for the DMS functionality.

- [validate](https://gitlab.com/nunet/device-management-service/-/blob/main/utils/validate): This contains helper functions that perform different kinds of validation checks and numeric conversions.

- [specs](https://gitlab.com/nunet/device-management-service/-/blob/main/utils/specs): This folder contains the class diagram for the package. 

Files with `*_test.go` naming contains unit tests of the specified functionality.

### Class Diagram

#### Source File

[utils Class Diagram](https://gitlab.com/nunet/device-management-service/-/blob/main/utils/specs/class_diagram.puml)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/main"
!$packageRelativePath = "/utils"
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath
 
!include $packageUrlGitlab/specs/class_diagram.puml
```

### Functionality

`utils` package defines various helper methods for functionality defined in the different packages of DMS. Refer to [utils.go](https://gitlab.com/nunet/device-management-service/-/blob/main/utils/utils.go) for details.

### Data Types

**Blockchain data models**

- `utils.UTXOs`: `TBD`

```
type UTXOs struct {
	TxHash  string `json:"tx_hash"`
	IsSpent bool   `json:"is_spent"`
}
```

- `utils.TxHashResp`: `TBD`

```
type TxHashResp struct {
	TxHash          string `json:"tx_hash"`
	TransactionType string `json:"transaction_type"`
	DateTime        string `json:"date_time"`
}
```

- `utils.ClaimCardanoTokenBody`: `TBD`

```
type ClaimCardanoTokenBody struct {
	ComputeProviderAddress string `json:"compute_provider_address"`
	TxHash                 string `json:"tx_hash"`
}
```

- `utils.rewardRespToCPD`: `TBD`

```
type rewardRespToCPD struct {
	ServiceProviderAddr string `json:"service_provider_addr"`
	ComputeProviderAddr string `json:"compute_provider_addr"`
	RewardType          string `json:"reward_type,omitempty"`
	SignatureDatum      string `json:"signature_datum,omitempty"`
	MessageHashDatum    string `json:"message_hash_datum,omitempty"`
	Datum               string `json:"datum,omitempty"`
	SignatureAction     string `json:"signature_action,omitempty"`
	MessageHashAction   string `json:"message_hash_action,omitempty"`
	Action              string `json:"action,omitempty"`
}
```

- `utils.UpdateTxStatusBody`: `TBD`

```
type UpdateTxStatusBody struct {
	Address string `json:"address,omitempty"`
}
```

**progress_io data models**

- `utils.IOProgress`: `TBD`
```
type IOProgress struct {
	n         float64
	size      float64
	started   time.Time
	estimated time.Time
	err       error
}
```

- `utils.Reader`: `TBD`
```
type Reader struct {
	reader   io.Reader
	lock     sync.RWMutex
	Progress IOProgress
}
```

- `utils.Writer`: `TBD`
```
type Writer struct {
	writer   io.Writer
	lock     sync.RWMutex
	Progress IOProgress
}
```

**syncmap data models**

- `utils.SyncMap`: a concurrency-safe sync.Map that uses strongly-typed method signatures to ensure the types of its stored data are known.

```
type SyncMap[K comparable, V any] struct {
	sync.Map
}
```

### Testing

The unit tests for the functionality are defined in `network_test.go` and `utils_test.go` files.

### Proposed Functionality / Requirements

List of issues related to the implementation of the `utils` package can be found below. These include proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- utils Package Issues - `TBD`

### References
