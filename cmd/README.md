
# cmd

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
3. [Class Diagram](#3-class-diagram)
4. [Functionality](#4-functionality)
5. [Data Types](#5-data-types)
6. [Testing](#6-testing)
7. [Proposed Functionality/Requirements](#7-proposed-functionality--requirements)
8. [References](#8-references)

## Specification

### 1. Description

The cmd package contains all functionality of Device Management Service (DMS) available via command line interface (CLI). 

### 2. Structure and organisation

Here is quick overview of the contents of this directory:

* [README](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/README.md): Current file which is aimed towards developers who wish to use and modify the cmd functionality. 

* [amd](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/amd.go): This file contains methods to collect information about AMD GPUs.

* [autocomplete](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/autocomplete.go): This file defines a command that allows users to generate shell autocompletion scripts for the Nunet CLI tool. It supports generating scripts for both Bash and Zsh shells.

* [capacity](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/capacity.go): This file defines the `capacity` command for the nunet CLI tool.

* [chat](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/chat.go): This file contains implementation of `chat` functionality.

* [chat_clear](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/chat_clear.go): This file contains implementation of clear chat functionality.

* [chat_join](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/chat_join.go): This file contains implementation of join chat functionality.

* [chat_list](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/chat_list.go): This file contains implementation of list chat functionality.

* [chat_start](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/chat_start.go): This file contains implementation of start chat functionality.

* [device](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/device.go): This file contains implementation of device related operations.

* [gpu](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/gpu.go): This file defines the `gpu` command.

* [gpu_capacity](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/gpu_capacity.go): This file defines the `gpu capacity` command and its flags.

* [gpu_interface](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/gpu_interface.go): This file defines GPU interface for accessing information about GPUs.

* [gpu_onboard](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/gpu_onboard.go): This file defines the `gpu onboard` command.

* [gpu_status](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/gpu_status.go): This file defines the `gpu status` command.

* [info](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/info.go): This file defines the `info` command which displays the information about onboarded device

* [init](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/init.go): This file initializes services for the nunet CLI tool and defines top-level commands and sub-commands. It also sets flags for some commands.

* [log_darwin](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/log_darwin.go): This file defines `log` command for MacOS.

* [log_linux](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/log_linux.go): This file defines `log` command for Linux.

* [nvidia](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/nvidia.go): This file contains implementation of `GPU` interface for NVIDIA GPUs.

* [offboard](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/offboard.go): This file defines the `offboard` command. 

* [onboard-ml](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/onboard-ml.go): This file defines the `onboard-ml` command.

* [onboard](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/onboard.go): This file defines the `onboard` command.

* [peer](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/peer.go): This file defines the `peer` command.

* [peer_default](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/peer_default.go): This file defines command to set a default peer for job deployment. Note that this is expected to be deprecated.

* [peer_list](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/peer_list.go): This file defines the list sub-command for `peer` command.

* [peer_self](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/peer_self.go): This file defines the self sub-command for `peer` command.

* [resource-config](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/resource-config.go): This file defines the `resource-config` command.

* [root](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/root.go): This file defines the root command `nunet`.

* [run](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/run.go): This file defines the `run` command.

* [shell](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/shell.go): This file defines the `shell` command

* [utils](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/utils.go): This file contains utility functions for the CLI functionality.

* [version](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/version.go): This file defines the `version` command.

* [wallet](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/wallet.go): This file defines the `wallet` command.

* [wallet_new](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/wallet_new.go): This file defines the subcommand `new` for the `wallet` command.

* [backend](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/backend): This directory contains 

All of the files named as `*_test.go` contains the unit tests for the corresponding functionality.

### 3. Class Diagram

The class diagram for the `cmd` package is shown below.

#### Source file

[cmd Class diagram](https://gitlab.com/nunet/device-management-service/-/blob/develop/cmd/specs/class_diagram.puml)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/develop"
!$packageRelativePath = "/cmd"
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath
 
!include $packageUrlGitlab/specs/class_diagram.puml
```

### 4. Functionality

The following sections describe the command line options that can be used with the CLI.

#### capacity

This command displays the capacity of the machine.

Usage:

```
nunet capacity --pretty
```

It has three flags that can be used along with this command

1. `full`: displays the full capacity of the machine.

```
nunet capacity --pretty --full
```

2. `onboarded`: displays the onboarded capacity of the machine.

```
nunet capacity --pretty --onboarded
```

3.  `available`: displays the resources still available for onboarding

```
nunet capacity --pretty --available
```


#### chat

**Note: this is expected to be deprecated after DMS refactoring**

This command allows users to chat with each other. The usage is explained below:

`start`: start a chat with a peer 

```
nunet chat start <node-id>
```

`list`: list open chat requests

```
nunet chat list
```

`clear`: clear open chat requests

```
nunet chat clear
```

`join`: join a chat stream using the request ID

```
nunet chat join <request-id>
```

The request-id mentioned above can be obtained from the `nunet chat list` command stated earlier.


#### device

This command allows users to perform device related operations. The usage is explained below:

`status`: get the current status of the device (online / offline)

```
nunet device status
```

`set`: set the status of the device (online/offline)

```
nunet device set <status>
```


#### gpu

These commands allows users to perform GPU related operations. The usage is explained below:

`gpu capacity`: check the GPU capacity of the machine

``` 
nunet gpu capacity
```

The gpu capacity command has three flags:

1. `cuda-tensor`: check the availability of CUDA and Tensor Cores 

``` 
nunet gpu capacity --cuda-tensor

or 

nunet gpu capacity -c
```

2. `rocm-hip`: check the availability of ROCm and HIP (AMD GPUs)

``` 
nunet gpu capacity --rocm-hip

or 

nunet gpu capacity -r

```

3. `intel-xpu`: check the availability of Intel XPU

``` 
nunet gpu capacity --intel-xpu

or

nunet gpu capacity -i

```


`gpu onboard`: install GPU drivers and Container Runtime

``` 
nunet gpu onboard
```

`gpu status`: check GPU status in real time

``` 
nunet gpu status
```

#### info

This command displays the metadata (`models.onboarding.Metadata`) of the onboarded device. 

``` 
nunet info
```

#### log

This command gathers all the logs into a tarball. The command must be run as root with SUDO access. Note that current MacOS is not supported.

``` 
nunet log
```

#### offboard

This command offboards the device from Nunet. 

``` 
nunet offboard
```

Use of flag `force` will force the offboarding process despite encountering any errors.

``` 
nunet offboard --force 

or 

nunet offboard -f
```

#### onboard-ml

This command is used to setup the environment for Machine Learning with GPU. It checks for WSL (Windows Subsystem for Linux) and detects available GPU vendors (AMD or NVIDIA). Note that it requires at least one type of GPU (AMD or NVIDIA) to be present.

It will create a Docker client and list existing images. Based on the OS (WSL or not) and detected GPU vendors, it pulls the required Docker images from a predefined list for either NVIDIA or AMD. It checks if the image already exists before pulling and provides informative messages during the process.

```
nunet onboard-ml
```

#### onboard

This command is used to onboard a compute provider machine onto Nunet. It expects input parameters specified in `models.onboarding.CapacityForNunet`. The machine metadata (`models.onboarding.Metadata`) is displayed once the onboarding process is completed.

Example usage: 

```
nunet onboard -m <memory in MB> -c <cpu in MHz> -n <channel> -a <address> [-C] [-l]
```

The various flags are explained below:

`-m <memory in MB>`: RAM of the machine provided to Nunet

`-c <cpu in MHz>`: CPU capacity provided to Nunet

`-n <channel>`: channel on which the onboarding it to be done. For example - `nunet-test`

`-a <address>`: this is the wallet address of the user

`[-C]`: optional flag that allows deployment of a Cardano node on the machine. 

`[-l]`: optional flag which sets server mode to be true. This is needed when running the DMS on a local machine to enable advertisement and discovery on a local network address.

`-x <Price>`: price of the machine is NTX/min

`-u`: to state that machine is not available for job deployment

#### peer

This command allows users to perform peer related operations. The usage is explained below:

`list`: display list of peers in the network. Use of `-d` flag will list only DHT peers.

```
nunet peer list 
```

`self`: display the peer info of the machine 

```
nunet peer self
```

`default`: Retrieve or set a peer as default for job deployment. **Note: This is expected to be deprecated**

```
nunet peer default [peerID]
```

Use <peerID> parameter as '0' to remove default deployment request receiver. Using the command without any <peerID> parameter will give the peer currently set as default deployment request receiver.

#### resource-config

This command is used to update the configuration of onboarded device. The machine metadata (`models.onboarding.Metadata`) is displayed upon update.

```
nunet resource-config -m <memory in MB> -c <cpu in MHz> -x <Price in NTX/min>
```

#### run

This command start the Device Management Service (DMS).

```
nunet run
```

#### shell

**Note: This is expected to be deprecated** 

```
nunet shell
```

#### version

This command prints the version of DMS installed.

```
nunet version
```

#### wallet

This command can be used to create a new wallet.

```
nunet wallet new
```

There are two flags

1. `--cardano or -c`: create wallet address on Cardano blockchain. This is the default option if no flag is provided

```
nunet wallet new --cardano
```

2. `--ethereum or -e`: create wallet address on Ethereum blockchain. Note: currently Ethererum blockchain is **not** supported

```
nunet wallet new --ethereum
```

### 5. Data Types

Refer to `api` package for all the data types applicable for the `cmd` package functionality. 

### 6. Testing

#### Unit Tests

All unit tests for various functionalities can be found in files with `_test` in their name.

#### Functional tests

- **TBD with Abhishek**

### 7. Proposed Functionality / Requirements 

#### List of issues

All issues that are related to the design of `cmd` package can be found below. These include any proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [cmd package design]() `TBD`

### 8. References


