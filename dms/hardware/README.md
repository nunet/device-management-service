# dms

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
3. [Functionality](#functionality)
4. [Data Types](#data-types)
5. [Testing](#testing)
6. [References](#references)


## Specification

### Description

The hardware package is responsible for handling the hardware related functionalities of the DMS. 

### Structure and Organisation

Here is quick overview of the contents of this package:

*[cpu](https://gitlab.com/nunet/device-management-service/-/tree/main/dms/hardware/cpu)*: This package contains the functionality related to the CPU of the device.

*[ram.go](https://gitlab.com/nunet/device-management-service/-/tree/main/dms/hardware/ram)*: This file contains the functionality related to the RAM.

*[disk.go](https://gitlab.com/nunet/device-management-service/-/tree/main/dms/hardware/disk)*: This file contains the functionality related to the Disk.

*[gpu](https://gitlab.com/nunet/device-management-service/-/tree/main/dms/hardware/gpu)*: This package contains the functionality related to the GPU of the device.

### Functionality

##### `GetMachineResources()`

- signature: `GetMachineResources() (types.MachineResources, error)`
- input: None
- output: `types.MachineResources`
- output(error): error

##### `GetCPU()`

- signature: `GetCPU() (types.CPU, error)`
- input: None
- output: `types.CPU`
- output(error): error

##### `GetRAM()`

- signature: `GetRAM() (types.RAM, error)`
- input: None
- output: `types.RAM`
- output(error): error

##### `GetDisk()`

- signature: `GetDisk() (types.Disk, error)`
- input: None
- output: `types.Disk`
- output(error): error

### Data Types

The hardware types can be found in the [types](https://gitlab.com/nunet/device-management-service/-/tree/main/types/hardware.go) package.

### Testing

The tests can be found in the `*_test.go` files in the respective packages.

### References