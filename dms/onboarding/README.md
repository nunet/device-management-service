# onboarding

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

This file explains the onboarding functionality of Device Management Service (DMS). This functionality is catered towards compute providers who wish provide their hardware resources to Nunet for running computational tasks as well as developers who are contributing to platform development.

### 2. Structure and organisation

Here is quick overview of the contents of this directory:

* [README](README.md): Current file which is aimed towards developers who wish to modify the onboarding functionality and build on top of it. 

* [handler](handler.go): This is main file where the code for onboarding functionality exists.

* [addresses](addresses.go): This file houses functions to generate Ethereum and Cardano wallet addresses along with its private key. 

* [addresses_test](addresses_test.go): This file houses functions to test the address generation functions defined in [addresses](addresses.go).

* [available_resources](available_resources.go): This file houses functions to get the total capacity of the machine being onboarded. 

* [init](init.go): This files initializes the loggers associated with onboarding package.

### 3. Functionality


#### Onboard Compute Provider

- signature: `Onboard(ctx context.Context, capacity models.CapacityForNunet) (*models.Metadata, error)`

- input #1: `Context object`

- input #2: `models.CapacityForNunet`

- output: `models.Metadata`

- output (error): Error message

`Onboard` function executes the onboarding process for a compute provider. 


#### Get Metadata 

- signature: `GetMetadata(ctx context.Context, capacity models.CapacityForNunet) (*models.Metadata, error)`

- input #1: `Context object`

- input #2: `models.CapacityForNunet`

- output: `models.Metadata`

- output (error): Error message

`GetMetadata` function retrieves and returns the machine metadata stored by the DMS.
 
 #### CreatePaymentAddress

- signature: `CreatePaymentAddress(wallet string) (*models.BlockchainAddressPrivKey, error)`

- input: `Blockchain name`

- output: `models.BlockchainAddressPrivKey`

- output (error): Error message

`CreatePaymentAddress` function creates a wallet for the user on the specified blockchain. 

#### Onboarding status

- signature: `Status() (*models.OnboardingStatus, error)`

- input: None

- output: `models.OnboardingStatus`

- output (error): Error message

`Status` function returns the onboarding status of the machine along with some metadata. 


### Change Resource Configuration

- signature: `ResourceConfig(ctx context.Context, capacity models.CapacityForNunet) (*models.Metadata, error)`

- input #1: `Context object`

- input #2: `models.CapacityForNunet`

- output: `models.Metadata`

- output (error): Error message

`ResourceConfig` changes the configuration of the resources onboarded to Nunet. 


### Offboard

- signature: `Offboard(ctx context.Context, force bool) error`

- input #1: `Context object`

- input #2: `force parameter`

- output: None

- output (error): Error message

`Offboard` removes the resources onboarded to Nunet. If the `force` parameter is `True`, then offboarding process will continue even in the presence of errors. 


### 4. Data Types

- `models.BlockchainAddressPrivKey`: This contains public key, private key and mnenmoic associated with it. This is generated when user opts to create a payment address / wallet using the api functionality.

- `models.CapacityForNunet`: This is the input provided by the compute provider user to start the onboarding process.

- `models.Metadata`: This contains information about the machine stored by the DMS. It is generated as a result of the onboarding process.

- `models.Provisioned`: This has total capacity of the machine onboarded by the user.

- `models.OnboardingStatus`: This is returned while retrieving the onboarding status.

- `models.AvailableResources`: This has the available capacity that has been onboarded to Nunet.

### 5. Testing

`TBD`

### 6. Proposed Functionality / Requirements 

#### List of issues

All issues that are related to the implementation of `dms` package can be found below. These include any proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [dms package implementation](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=opened&label_name%5B%5D=collaboration_group_24%3A%3A33&first_page_size=20)


### 7. References

The DMS is being refactored and augmented with several new functionalities. The proposed class diagram can be found here:
- [Class Diagram - Source](https://gitlab.com/nunet/device-management-service/-/blob/develop/specs/classDiagrams/dms-global.mermaid)
- [Class Diagram - Rendered](https://gitlab.com/nunet/device-management-service/-/blob/develop/specs/classDiagrams/dms-global.svg)
