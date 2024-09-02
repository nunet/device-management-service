# onboarding

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
4. [Functionality](#4-functionality)
5. [Data Types](#5-data-types)
6. [Testing](#6-testing)
7. [Proposed Functionality/Requirements](#7-proposed-functionality--requirements)
8. [References](#8-references)


## Specification

### 1. Description

This file explains the onboarding functionality of Device Management Service (DMS). This functionality is catered towards compute providers who wish provide their hardware resources to Nunet for running computational tasks as well as developers who are contributing to platform development.

### 2. Structure and Organisation

Here is quick overview of the contents of this directory:

* [README](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/onboarding/README.md): Current file which is aimed towards developers who wish to modify the onboarding functionality and build on top of it. 

* [handler](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/onboarding/handler.go): This is main file where the code for onboarding functionality exists.

* [addresses](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/onboarding/addresses.go): This file houses functions to generate Ethereum and Cardano wallet addresses along with its private key. 

* [addresses_test](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/onboarding/addresses_test.go): This file houses functions to test the address generation functions defined in [addresses](addresses.go).

* [available_resources](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/onboarding/available_resources.go): This file houses functions to get the total capacity of the machine being onboarded. 

* [init](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/onboarding/init.go): This files initializes the loggers associated with onboarding package.

### 3. Class Diagram

The class diagram for the `onboarding` package is shown below.

#### Source file

[onboarding Class Diagram](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/onboarding/specs/class_diagram.puml)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/main"
!$packageRelativePath = "/dms/onboarding"
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath
 
!include $packageUrlGitlab/specs/class_diagram.puml
```

### 4. Functionality

#### Onboard Compute Provider

- signature: `Onboard(ctx context.Context, capacity types.CapacityForNunet) (*types.Metadata, error)`

- input #1: `Context object`

- input #2: `types.CapacityForNunet`

- output: `types.Metadata`

- output (error): Error message

`Onboard` function executes the onboarding process for a compute provider. 


#### Get Metadata 

- signature: `GetMetadata(ctx context.Context, capacity types.CapacityForNunet) (*types.Metadata, error)`

- input #1: `Context object`

- input #2: `types.CapacityForNunet`

- output: `types.Metadata`

- output (error): Error message

`GetMetadata` function retrieves and returns the machine metadata stored by the DMS.
 
 #### CreatePaymentAddress

- signature: `CreatePaymentAddress(wallet string) (*types.BlockchainAddressPrivKey, error)`

- input: `Blockchain name`

- output: `types.BlockchainAddressPrivKey`

- output (error): Error message

`CreatePaymentAddress` function creates a wallet for the user on the specified blockchain. 

#### Onboarding status

- signature: `Status() (*types.OnboardingStatus, error)`

- input: None

- output: `types.OnboardingStatus`

- output (error): Error message

`Status` function returns the onboarding status of the machine along with some metadata. 


### Change Resource Configuration

- signature: `ResourceConfig(ctx context.Context, capacity types.CapacityForNunet) (*types.Metadata, error)`

- input #1: `Context object`

- input #2: `types.CapacityForNunet`

- output: `types.Metadata`

- output (error): Error message

`ResourceConfig` changes the configuration of the resources onboarded to Nunet. 


### Offboard

- signature: `Offboard(ctx context.Context, force bool) error`

- input #1: `Context object`

- input #2: `force parameter`

- output: None

- output (error): Error message

`Offboard` removes the resources onboarded to Nunet. If the `force` parameter is `True`, then offboarding process will continue even in the presence of errors. 


### 5. Data Types

- `types.BlockchainAddressPrivKey`: This contains public key, private key and mnenmoic associated with it. This is generated when user opts to create a payment address / wallet using the api functionality.

- `types.CapacityForNunet`: This is the input provided by the compute provider user to start the onboarding process.

- `types.Metadata`: This contains information about the machine stored by the DMS. It is generated as a result of the onboarding process.

- `types.Provisioned`: This has total capacity of the machine onboarded by the user.

- `types.OnboardingStatus`: This is returned while retrieving the onboarding status.

- `types.AvailableResources`: This has the available capacity that has been onboarded to Nunet.

### 6. Testing

`TBD`

### 7. Proposed Functionality / Requirements 

#### List of issues

All issues that are related to the implementation of `dms` package can be found below. These include any proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [dms package implementation](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=opened&label_name%5B%5D=collaboration_group_24%3A%3A33&first_page_size=20)


### 8. References

