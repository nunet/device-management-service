# onboarding

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

This file explains the onboarding functionality of Device Management Service (DMS). This functionality is catered towards compute providers who wish provide their hardware resources to Nunet for running computational tasks as well as developers who are contributing to platform development.

### Structure and Organisation

Here is quick overview of the contents of this directory:

* [README](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/onboarding/README.md): Current file which is aimed towards developers who wish to modify the onboarding functionality and build on top of it. 

* [handler](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/onboarding/handler.go): This is main file where the code for onboarding functionality exists.

* [addresses](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/onboarding/addresses.go): This file houses functions to generate Cardano wallet addresses along with its private key. 

* [addresses_test](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/onboarding/addresses_test.go): This file houses functions to test the address generation functions defined in [addresses](addresses.go).

* [available_resources](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/onboarding/available_resources.go): This file houses functions to get the total capacity of the machine being onboarded. 

* [init](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/onboarding/init.go): This files initializes the loggers associated with onboarding package.

### Class Diagram

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

### Functionality

#### Onboard

- signature: `Onboard(ctx context.Context, config types.OnboardingConfig) error`

- input #1: `Context object`

- input #2: `types.OnboardingConfig`

- output (error): Error message

`Onboard` function executes the onboarding process for a compute provider based on the configuration provided.

### Offboard

- signature: `Offboard(ctx context.Context) error`

- input #1: `Context object`

- output: None

- output (error): Error message

`Offboard` removes the resources onboarded to Nunet.

### IsOnboarded

- signature: `IsOnboarded(ctx context.Context) (bool, error)`
- input #1: `Context object`
- output #1: `bool`
- output #2: `error`

`IsOnboarded` checks if the compute provider is onboarded.

### Info

- signature: `Info(ctx context.Context) (types.OnboardingConfig, error)`
- input #1: `Context object`
- output #1: `types.OnboardingConfig`
- output #2: `error`

`Info` returns the configuration of the onboarding process.

### Data Types

- `types.OnboardingConfig`: Holds the configuration for onboarding a compute provider.

### Testing

- All the tests for the onboarding package can be found in the [onboarding_test.go](https://gitlab.com/nunet/device-management-service/-/blob/main/dms/onboarding/onboarding_test.go) file.

### Proposed Functionality / Requirements 

#### List of issues

All issues that are related to the implementation of `dms` package can be found below. These include any proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [dms package implementation](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=opened&label_name%5B%5D=collaboration_group_24%3A%3A33&first_page_size=20)


### References

