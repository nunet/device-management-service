# config

- [Project README](https://gitlab.com/nunet/device-management-service/-/blob/develop/README.md)
- [Release/Build Status](https://gitlab.com/nunet/device-management-service/-/releases)
- [Changelog](https://gitlab.com/nunet/device-management-service/-/blob/develop/CHANGELOG.md)
- [License](https://www.apache.org/licenses/LICENSE-2.0.txt)
- [Contribution Guidelines](https://gitlab.com/nunet/device-management-service/-/blob/develop/CONTRIBUTING.md)
- [Code of Conduct](https://gitlab.com/nunet/device-management-service/-/blob/develop/CODE_OF_CONDUCT.md)
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

This package contains all configuration related code such as reading config file and functions to configure at runtime.

`proposed` There are two sides to configuration:
1. default configuration, which has to be loaded for the fresh installation of new dms;
2. dynamic configuration, which can be changed by a user that has access to the DMS; this dynamic configuration may need to be persistent (or not).

#### `proposed` Default configuration

Default configuration should be included into DMS distribution as a `config.yaml` in the root directory. The following is loosely based on general practice of passing yaml configuration to Go programs (see e.g. [A clean way to pass configs in a Go application](https://dev.to/ilyakaznacheev/a-clean-way-to-pass-configs-in-a-go-application-1g64)). DMS would parse this file during onboarding and populate the `internal.config.Config` variable that will be imported to other packages and used accordingly. 

## `proposed` Dynamic configuration

Dynamic configuration would use the same `internal.config.Config` variable, but would allow for adding new values or changing configuration by an authorized DMS user -- via DMS CLI or REST API calls. 

The mechanism of dynamic configuration will enable to override or change default values. For enabling this functionality, the `internal.config.Config` variable will have a synchronized copy in the local DMS database, defined with `db` package. 

### Structure and Organisation

Here is quick overview of the contents of this pacakge:

* [README](https://gitlab.com/nunet/device-management-service/-/tree/develop/internal/config/README.md): Current file which is aimed towards developers who wish to use and modify the package functionality.

* [config](https://gitlab.com/nunet/device-management-service/-/tree/develop/internal/config/config.go): This file contains data structures for this package.

* [load](https://gitlab.com/nunet/device-management-service/-/tree/develop/internal/config/load.go): This file establishes a configuration loader using `Viper`, supports loading JSON files from various locations, applies defaults, and exposes functions to manage and obtain the loaded configuration.

### Class Diagram

#### Source

[config class diagram](https://gitlab.com/nunet/device-management-service/-/blob/develop/internal/config/specs/class_diagram.puml)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/main"
!$packageRelativePath = "/internal/config"
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath
 
!include $packageUrlGitlab/specs/class_diagram.puml
```

### Functionality

The methods of this package are explained below:

#### getViper

* signature: `getViper() *viper.Viper` <br/>

* input: None <br/>

* output: A pointer to a `viper.Viper` struct

`getViper` function creates a new `viper.Viper` instance used for configuration management with search paths.

#### setDefaultConfig

* signature: `setDefaultConfig() *viper.Viper` <br/>

* input: None <br/>

* output: A pointer to a `viper.Viper` struct

Sets default values for various configuration options in a `viper.Viper` instance.


#### LoadConfig

* signature: `LoadConfig()` <br/>

* input: None <br/>

* output: None

`LoadConfig` loads the configuration from the search paths.

#### SetConfig

* signature: `SetConfig(key string, value interface{})` <br/>

* input #1 : key for the configuration value <br/>

* input #2 : value corresponding to the key provided <br/>

* output: None

`SetConfig` Sets a specific configuration value using key value pair provided. In case of any error, the deafult configuration values are applied.

#### GetConfig

* signature: `GetConfig() *Config` <br/>

* input: None <br/>

* output: `internal.config.Config`

`GetConfig` returns the loaded configuration data.

#### findConfig

* signature: `findConfig(paths []string, filename string) ([]byte, error)` <br/>

* input # 1: list of search paths <br/>

* input # 1: name of the configuration file  <br/>

* output: contents of the configuration file 

* output(error): error message

`findConfig` Searches for the configuration file in specified paths and returns content or error.

#### removeComments

* signature: `removeComments(configBytes []byte) []byte` <br/>

* input: The byte array containing the configuration file content <br/>

* output: byte array with comments removed

`removeComments` Removes comments from the configuration file content using a regular expression.

### Data Types

- `internal.config.Config`: holds the overall configuration with nested structs for specific sections

```
type Config struct {
	General `mapstructure:"general"`
	Rest    `mapstructure:"rest"`
	P2P     `mapstructure:"p2p"`
	Job     `mapstructure:"job"`
}
```

- `internal.config.General`: Configuration related to general application behavior (data paths, debug mode)

```
type General struct {
	MetadataPath string `mapstructure:"metadata_path"`
	DataDir      string `mapstructure:"data_dir"`
	Debug        bool   `mapstructure:"debug"`
}
```

- `internal.config.Rest`: Configuration for the REST API (port number)

```
type Rest struct {
	Port int `mapstructure:"port"`
}
```

- `internal.config.P2P`: Configuration for the P2P network

```
type P2P struct {
	ListenAddress  []string `mapstructure:"listen_address"`
	BootstrapPeers []string `mapstructure:"bootstrap_peers"`
}
```

- `internal.config.Job`: Configuration for background tasks (log update interval, target peer for deployments, container cleanup interval)

```
type Job struct {
	LogUpdateInterval int    `mapstructure:"log_update_interval"` // in minutes
	TargetPeer        string `mapstructure:"target_peer"`         // specific peer to send deployment requests to - XXX probably not a good idea. Remove after testing stage.
	CleanupInterval   int    `mapstructure:"cleanup_interval"` // docker container and images clean up interval in days
}
```

### Testing

`proposed` Unit tests for each functionality are defined in files with `*_test.go` naming convention.

### Proposed Functionality / Requirements 

#### List of issues

All issues that are related to the implementation of `internal` package can be found below. These include any proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [internal package implementation]() `TBD`

#### `proposed` Functionalities

Following Gherkin feature files describe the proposed functionality for `config` package.

1. **Load default DMS configuration**: see [scenario definition](https://gitlab.com/nunet/test-suite/-/blob/proposed/stages/functional_tests/features/device-management-service/internal/config/configurationManagement.feature?ref_type=heads#L3)

2. **Restore default DMS configuration**: see [scenario definition](https://gitlab.com/nunet/test-suite/-/blob/proposed/stages/functional_tests/features/device-management-service/internal/config/configurationManagement.feature?ref_type=heads#L14)

3. **Load existing DMS configuration**: see [scenario definition](https://gitlab.com/nunet/test-suite/-/blob/proposed/stages/functional_tests/features/device-management-service/internal/config/configurationManagement.feature?ref_type=heads#L28)

### References








