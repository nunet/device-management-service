# models

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

`models` package defines and keeps data structures and interfaces that are used across the whole DMS component by different packages. 

### 2. Structure and organisation

Here is quick overview of the contents of this pacakge:

* [README](https://gitlab.com/nunet/device-management-service/-/tree/develop/models/README.md): Current file which is aimed towards developers who wish to use and modify the package functionality.

* [capability](https://gitlab.com/nunet/device-management-service/-/tree/develop/models/capability.go): This file contains data structures to describe machine capability.

* [comparison](https://gitlab.com/nunet/device-management-service/-/tree/develop/models/comparison.go): This file contains constans and types used for capability comparison.

* [constants](https://gitlab.com/nunet/device-management-service/-/tree/develop/models/constants.go): This file contains constants that are used across different packages.

* [deployment](https://gitlab.com/nunet/device-management-service/-/tree/develop/models/deployment.go): This file contains data structure related to job deployment.

* [elk_stats](https://gitlab.com/nunet/device-management-service/-/tree/develop/models/elk_stats.go): This file contains data structure to be sent to elasticsearch collector.

* [encryption](https://gitlab.com/nunet/device-management-service/-/tree/develop/models/encryption.go): This file contains data structure related to encryption in DMS.

* [execution](https://gitlab.com/nunet/device-management-service/-/tree/develop/models/execution.go): This file contains data structure related to executor functionality.

* [firecracker](https://gitlab.com/nunet/device-management-service/-/tree/develop/models/firecracker.go): This file contains data structure related to firecracker.

* [machine](https://gitlab.com/nunet/device-management-service/-/tree/develop/models/machine.go): This file contains data structure related to the machine - resources, peer details, services etc.

* [network](https://gitlab.com/nunet/device-management-service/-/tree/develop/models/network.go): This file contains data structure related to networking functionality of DMS

* [network_config](https://gitlab.com/nunet/device-management-service/-/tree/develop/models/network_config.go): This file defines message types, network types (libp2p, NATS) with configurations, and libp2p specific configurations (DHT, keys, peers, scheduling etc).

* [onboarding](https://gitlab.com/nunet/device-management-service/-/tree/develop/models/onboarding.go): This file contains data structure related to compute provider onboarding.

* [resource](https://gitlab.com/nunet/device-management-service/-/tree/develop/models/resource.go): This file contains data structures of GPU and execution resources.

* [spec_config](https://gitlab.com/nunet/device-management-service/-/tree/develop/models/spec_config.go): This file defines a `SpecConfig` struct for configuration data with type, parameters, normalization, validation, and type checking functionalities.

* [storage](https://gitlab.com/nunet/device-management-service/-/tree/develop/models/storage.go): This file contains data structures related to storage.

* [telemetry](https://gitlab.com/nunet/device-management-service/-/tree/develop/models/telemetry_config.go): This file defines structs related to telemetry configuration and methods to load configuration from environment variables.

* [types](https://gitlab.com/nunet/device-management-service/-/tree/develop/models/types.go):  This file defines a base model for entities in the application with auto-generated UUIDs, timestamps, and soft delete functionality using GORM hooks.

### 3. Class Diagram

#### Source

[models class diagram](https://gitlab.com/nunet/device-management-service/-/blob/develop/models/specs/class_diagram.puml)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/develop"
!$packageRelativePath = "/models"
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath
 
!include $packageUrlGitlab/specs/class_diagram.puml
```

### 4. Functionality

`models` package holds interfaces and methods that are used by multiple packages. The functionality of these interfaces/methods are typically implemented in other packages.

Here are some methods defined in `models` package:

#### NewExecutionResult

* signature: `NewExecutionResult(code int) *ExecutionResult` <br/>

* input: `exit code` <br/>

* output: `models.ExecutionResult`

`NewExecutionResult` creates a new `ExecutionResult` object.

#### NewFailedExecutionResult

* signature: `NewFailedExecutionResult(err error) *ExecutionResult` <br/>

* input: `error` <br/>

* output: `models.ExecutionResult`

`NewFailedExecutionResult` creates a new `ExecutionResult` object for a failed execution. It sets the error message from the provided error and sets the exit code to -1.

#### Config interface `TBD`

```
type Config interface {
	GetNetworkConfig() *SpecConfig
}
```

`GetNetworkConfig` will return the network configuration parameters.

#### NewSpecConfig

* signature: `NewSpecConfig(t string) *SpecConfig` <br/>

* input: `type for the configuration object` <br/>

* output: `models.SpecConfig`

`NewSpecConfig` creates new `SpecConfig` with the given type and an empty params map.

#### WithParam

* signature: `(s *SpecConfig) WithParam(key string, value interface{}) *SpecConfig` <br/>

* input #1 : `key` <br/>

* input #2 : `value associated with the key` <br/>

* output: `models.SpecConfig`

`WithParam` adds a new key-value pair to the spec parameters and returns the updated `SpecConfig` object.

#### Normalize

* signature: `(s *SpecConfig) Normalize()` <br/>

* input: None <br/>

* output: None

`Normalize` ensures that the spec config is in a valid state by trimming whitespace from the Type field and initializing the Params map if empty.

#### Validate

* signature: `(s *SpecConfig) Validate() error` <br/>

* input: None <br/>

* output: `error`

`Validate` checks if the spec config is valid. It returns an error if the `SpecConfig` is nil or if the Type field is missing (blank). Otherwise, it returns no error indicating a valid configuration.

#### IsType

* signature: `(s *SpecConfig) IsType(t string) bool` <br/>

* input: `type` <br/>

* output: `bool`

`IsType` checks if the `SpecConfig` matches the provided type, ignoring case sensitivity. It returns `true` if there's a match, otherwise `false.`

#### IsEmpty

* signature: `(s *SpecConfig) IsEmpty() bool` <br/>

* input: None <br/>

* output: `bool`

`IsEmpty` checks if the `SpecConfig` is empty, meaning it's either nil or has an empty Type and no parameters. It returns true if empty, otherwise false.

#### GetID

* signature: `(m models.BaseDBModel) GetID() string` <br/>

* input: None <br/>

* output: `identifier of the entity`

`GetID` returns the identifier of the entity.

#### BeforeCreate

* signature: `(m *Model) BeforeCreate(tx *gorm.DB) error` <br/>

* input: `gorm database object to be created` <br/>

* output: `bool`

`BeforeCreate` sets the `ID` and `CreatedAt` fields before creating a new entity.

#### BeforeUpdate

* signature: `(m *Model) BeforeUpdate(tx *gorm.DB) error` <br/>

* input: `gorm database object to be updated` <br/>

* output: `bool`

`BeforeUpdate` returns true if the spec config is empty.

#### LoadConfigFromEnv

* signature: `LoadConfigFromEnv() (*TelemetryConfig, error)` <br/>

* input: none <br/>

* output: `models.TelemetryConfig`

* output (error): error message

`LoadConfigFromEnv` loads the telemetry configuration from environment variables. This includes observabilitty level and collector configuration. It returns the final configuration as `models.TelemetryConfig` object.

#### parseObservabilityLevel

* signature: `parseObservabilityLevel(levelStr string) int` <br/>

* input: `observability level` <br/>

* output: `int value corresponding to the input observability level`

`parseObservabilityLevel` returns the integer representation of the provided observability level string. When the input string does not match the defined observability levels, default observability level `INFO` is considered and its integer value is returned.



### 5. Data Types

#### Deployment

- `DeploymentRequest`

```
type DeploymentRequest struct {
	RequesterWalletAddress string    `json:"address_user"` // service provider wallet address
	MaxNtx                 int       `json:"max_ntx"`
	Blockchain             string    `json:"blockchain"`
	TxHash                 string    `json:"tx_hash"`
	ServiceType            string    `json:"service_type"`
	Timestamp              time.Time `json:"timestamp"`
	MetadataHash           string    `json:"metadata_hash"`
	WithdrawHash           string    `json:"withdraw_hash"`
	RefundHash             string    `json:"refund_hash"`
	Distribute_50Hash      string    `json:"distribute_50_hash"`
	Distribute_75Hash      string    `json:"distribute_75_hash"`
	Params                 struct {
		ImageID   string `json:"image_id"`
		ModelURL  string `json:"model_url"`
		ResumeJob struct {
			Resume       bool   `json:"resume"`
			ProgressFile string `json:"progress_file"` // file path
		} `json:"resume_job"`
		Packages        []string `json:"packages"`
		RemoteNodeID    string   `json:"node_id"`          // NodeID of compute provider (machine to deploy the job on)
		RemotePublicKey string   `json:"public_key"`       // Public key of compute provider
		LocalNodeID     string   `json:"local_node_id"`    // NodeID of service provider (machine triggering the job)
		LocalPublicKey  string   `json:"local_public_key"` // Public key of service provider
		MachineType     string   `json:"machine_type"`
	} `json:"params"`
	Constraints struct {
		Complexity string `json:"complexity"`
		CPU        int    `json:"cpu"`
		RAM        int    `json:"ram"`
		Vram       int    `json:"vram"`
		Power      int    `json:"power"`
		Time       int    `json:"time"`
	} `json:"constraints"`
	TraceInfo struct {
		TraceID     string `json:"trace_id"`
		SpanID      string `json:"span_id"`
		TraceFlags  string `json:"trace_flags"`
		TraceStates string `json:"trace_state"`
	} `json:"traceinfo"`
}
```

- `DeploymentResponse`

```
type DeploymentResponse struct {
	Success bool   `json:"success"`
	Content string `json:"content"`
}
```

- `DeploymentUpdate`

```
type DeploymentUpdate struct {
	MsgType string `json:"msg_type"`
	Msg     string `json:"msg"`
}
```

- `DeploymentRequestFlat`

```
type DeploymentRequestFlat struct {
	models.BaseDBModel
	DeploymentRequest string `json:"deployment_request"`
	// represents job status from services table; goal is to keep then in sync (both tables are on different DMSes).
	JobStatus string `json:"job_status"`
}
```

- `BlockchainTxStatus`

```
type BlockchainTxStatus struct {
	TransactionType   string `json:"transaction_type"` // No need of this param maybe be deprecated in future
	TransactionStatus string `json:"transaction_status"`
	TxHash            string `json:"tx_hash"`
}
```

#### ELK

- `NewDeviceOnboarded`

```
// NewDeviceOnboarded defines the schema of the data to be sent to stats db when a new device gets onboarded
type NewDeviceOnboarded struct {
	PeerID        string
	CPU           float32
	RAM           float32
	Network       float32
	DedicatedTime float32
	Timestamp     float32
}
```

- `DeviceStatusChange`
```
// DeviceStatusChange defines the schema of the data to be sent to stats db when a device status gets changed
type DeviceStatusChange struct {
	PeerID    string
	Status    string
	Timestamp float32
}
```

- `DeviceResourceChange`
```
// DeviceResourceChange defines the schema of the data to be sent to stats db when a device resource gets changed
type DeviceResourceChange struct {
	PeerID                   string
	ChangedAttributeAndValue struct {
		CPU           float32
		RAM           float32
		Network       float32
		DedicatedTime float32
	}
	Timestamp float32
}
```

- `DeviceResourceConfig`

```
// DeviceResourceConfig defines the schema of the data to be sent to stats db when a device resource config gets changed
type DeviceResourceConfig struct {
	PeerID                   string
	ChangedAttributeAndValue struct {
		CPU           float32
		RAM           float32
		Network       float32
		DedicatedTime float32
	}
	Timestamp float32
}
```

- `NewService`

// NewService defines the schema of the data to be sent to stats db when a new service gets registered in the platform
type NewService struct {
	ServiceID          string
	ServiceName        string
	ServiceDescription string
	Timestamp          float32
}

- `ServiceCall`

```
// ServiceCall defines the schema of the data to be sent to stats db when a host machine accepts a deployement request
type ServiceCall struct {
	CallID              float32
	PeerIDOfServiceHost string
	ServiceID           string
	CPUUsed             float32
	MaxRAM              float32
	MemoryUsed          float32
	NetworkBwUsed       float32
	TimeTaken           float32
	Status              string
	AmountOfNtx         int32
	Timestamp           float32
}
```

- `ServiceStatus`

```
// ServiceStatus defines the schema of update the status of service to stats db of the job being executed on host machine
type ServiceStatus struct {
	CallID              float32
	PeerIDOfServiceHost string
	ServiceID           string
	Status              string
	Timestamp           float32
}
```

- `ServiceRemove`

```
// ServiceRemove defines the schema of the data to be sent to stats db when a new service gets removed from the platform
type ServiceRemove struct {
	ServiceID string
	Timestamp float32
}
```

- `NtxPayment`

```
// NtxPayment defines the schema of the data to be sent to stats db when a payment is made to device for the completion of service.
type NtxPayment struct {
	CallID            float32
	ServiceID         string
	AmountOfNtx       int32
	PeerID            string
	SuccessFailStatus string
	Timestamp         float32
}
```

#### Executor

- `Executor`: This defines the type of executor that is used to execute the job.

```
type Executor struct {
	ExecutorType ExecutorType `json:"executor_type"`
}

type ExecutorType string
``` 


- `ExecutionRequest`: This is the input that `executor` receives to initiate a job execution. 

```
type ExecutionRequest struct {
	// ID of the job to execute
	JobID string

	// ID of the execution
	ExecutionID string

	// Engine spec for the execution
	EngineSpec models.SpecConfig

	// Resources for the execution
	Resources executor.ExecutionResources

	// Input volumes for the execution
	Inputs []storage.StorageVolume

	// Output volumes for the results
	Outputs []storage.StorageVolume

	// Directory to store the results
	ResultsDir string
}
```

- `ExecutionResult`: This contains the result of the job execution. 

```
type ExecutionResult struct {
	// STDOUT of the execution
	STDOUT string `json:"stdout"`

	// STDERR of the execution
	STDERR string `json:"stderr"`

	// Exit code of the execution
	ExitCode int `json:"exit_code"`

	// Error message if the execution failed
	ErrorMsg string `json:"error_msg"`
}
```

- `SpecConfig` `TBD`: This allows arbitrary configuration/parameters as needed during implementation of specific executor. 

```
// SpecConfig represents a configuration for a spec
// A SpecConfig can be used to define an engine spec, a storage volume, etc.
type SpecConfig struct {
	// Type of the spec (e.g. docker, firecracker, storage, etc.)
	Type string `json:"type"`

	// Params of the spec
	// This allows passing arbitrary parameters to the spec implementation
	Params map[string]interface{} `json:"params,omitempty"`
}
```

- `LogStreamRequest`: This is the input provided when a request to stream logs of an execution is made.

```
type LogStreamRequest struct {
	JobID       string // ID of the job
	ExecutionID string // ID of the execution
	Tail        bool   // Tail the logs
	Follow      bool   // Follow the logs
}
```

#### Firecracker

- `BootSource`: This contains configuration parameters for booting a Firecracker VM.

```
type BootSource struct {
	KernelImagePath string `json:"kernel_image_path"`
	BootArgs        string `json:"boot_args"`
}
```

- `Drives`: This contains properties of a virtual drive for Firecracker VM.
```
type Drives struct {
	DriveID      string `json:"drive_id"`
	PathOnHost   string `json:"path_on_host"`
	IsRootDevice bool   `json:"is_root_device"`
	IsReadOnly   bool   `json:"is_read_only"`
}
```

- `MachineConfig`: This defines the configuration parameters of the machine to be used while creating a new Firecracker VM.

```
type MachineConfig struct {
	VCPUCount  int `json:"vcpu_count"`
	MemSizeMib int `json:"mem_size_mib"`
}
```

- `NetworkInterfaces`: This defines the network configuration parameters.

```
type NetworkInterfaces struct {
	IfaceID     string `json:"iface_id"`
	GuestMac    string `json:"guest_mac"`
	HostDevName string `json:"host_dev_name"`
}
```

- `MMDSConfig`: This contains a list of the network configuration parameters defined by` NetworkInterfaces` struct.
```
type MMDSConfig struct {
	NetworkInterface []string `json:"network_interfaces"`
}
```

- `MMDSMsg` `TBD`: This contains the latest metadata of the machine.
```
type MMDSMsg struct {
	Latest struct {
		Metadata struct {
			models.MMDSMetadata
		} `json:"meta-data"`
	} `json:"latest"`
}
```

- `MMDSMetadata` `TBD`: This contains the metadata of the machine.
```
type MMDSMetadata struct {
	NodeId string `json:"node_id"`
	PKey   string `json:"pkey"`
}
```

- `Actions` `TBD`: This contains the type of action to be performed on the Firecracker VM.
```
type Actions struct {
	ActionType string `json:"action_type"`
}
```

- `VirtualMachine`: This contains the configuration parameters of Firecracker virtual machine.
```
type VirtualMachine struct {
	models.BaseDBModel
	SocketFile string `json:"socket_file"`
	BootSource string `json:"boot_source"`
	Filesystem string `json:"filesystem"`
	VCPUCount  int    `json:"vcpu_count"`
	MemSizeMib int    `json:"mem_size_mib"`
	TapDevice  string `json:"tap_device"`
	State      string `json:"state"`
}
```

#### Machine

- `IP`
```
type IP []any
```

- `PeerInfo` `TBD`: This contains parameters of the peer node.

```
type PeerInfo struct {
	models.BaseDBModel
	NodeID    string `json:"nodeID,omitempty"`
	Key       string `json:"key,omitempty"`
	Mid       string `json:"mid,omitempty"`
	PublicKey string `json:"public_key,omitempty"`
	Address   string `json:"_address,omitempty"`
}

```

- `Machine`: This contains the configuration parameters of the machine.

```
type Machine struct {
	models.BaseDBModel
	NodeId               string
	PeerInfo             int
	IpAddr               string
	AvailableResources   int
	FreeResources        int
	TokenomicsAddress    string
	TokenomicsBlockchain string
}
```

- `FreeResources`: This contains the resources currently available for a job.

```
// FreeResources are the resources free to be used by new services,
// plugins and any other processes started by DMS. It's basically
// the subtraction between AvailableResources and the amount of resources
// already used by DMS and its processes (mostly services)
type FreeResources struct {
	BaseDBModel
	TotCpuHz          int     `json:"tot_cpu_hz"`
	PriceCpu          float64 `json:"price_cpu"`
	Ram               int     `json:"ram"`
	PriceRam          float64 `json:"price_ram"`
	Vcpu              int     `json:"vcpu"`
	Disk              float64 `json:"disk"`
	PriceDisk         float64 `json:"price_disk"`
	NTXPricePerMinute float64 `json:"ntx_price"`
}
``` 

- `AvailableResources`: This contains the resources onboarded to Nunet by the user.

```
// AvailableResources are the amount of resources onboarded which
// can be used by NuNet
type AvailableResources struct {
	models.BaseDBModel
	TotCpuHz          int
	CpuNo             int
	CpuHz             float64
	PriceCpu          float64
	Ram               int
	PriceRam          float64
	Vcpu              int
	Disk              float64
	PriceDisk         float64
	NTXPricePerMinute float64
}
```

- `Services` `TBD`: This contains the details of the services running on the machine.

```
type Services struct {
	models.BaseDBModel
	TxHash               string
	TransactionType      string // transaction type can be running, done, withdraw, refund and distribute
	JobStatus            string // whether job is running or exited; one of these 'running', 'finished without errors', 'finished with errors'
	JobDuration          int64  // job duration in minutes
	EstimatedJobDuration int64  // job duration in minutes
	ServiceName          string
	ContainerID          string
	ResourceRequirements int
	ImageID              string
	LogURL               string
	LastLogFetch         time.Time
	ServiceProviderAddr  string
	ComputeProviderAddr  string
	MetadataHash         string
	WithdrawHash         string
	RefundHash           string // saving hashes for call the `/request-reward` endpoint by SPD
	Distribute_50Hash    string
	Distribute_75Hash    string
	SignatureDatum       string
	MessageHashDatum     string
	Datum                string
	SignatureAction      string // saving signatures for removing redundancy of calling Oracle
	MessageHashAction    string
	Action               string
	// TODO: Add ContainerType field

}
```

- `ServiceResourceRequirements`: This contains the resource requirements for a service.

```
type ServiceResourceRequirements struct {
	models.BaseDBModel
	CPU  int
	RAM  int
	VCPU int
	HDD  int
}

```

- `ContainerImages`: This contains parameters of a container image.

```
type ContainerImages struct {
	gorm.Model
	ImageID   string
	ImageName string
	Digest    string
}
```

- `Libp2pInfo`: This contains parameters of Libp2p node.

```
type Libp2pInfo struct {
	BaseDBModel
	PrivateKey []byte `json:"private_key"`
	PublicKey  []byte `json:"public_key"`
	ServerMode bool   `json:"server_mode"`
	Available  bool   `json:"available"`
}
```

- `MachineUUID`: This defines the unique identifier for the machine.

```
type MachineUUID struct {
	BaseDBModel
	UUID string `json:"uuid"`
}
```

- `Gpu`: This contains the GPU parameters of the machine.

```
type Gpu struct {
	Name     string `json:"name"`
	TotVram  uint64 `json:"tot_vram"`
	FreeVram uint64 `json:"free_vram"`
}
```

- `resources`: This defines the resource parameters of the machine.

```
type resources struct {
	TotCpuHz  float64
	PriceCpu  float64
	Ram       int
	PriceRam  float64
	Vcpu      int
	Disk      float64
	PriceDisk float64
}
```

- `PeerData`: This contains the details of the peer node.

```
type PeerData struct {
	PeerID               string        `json:"peer_id"`
	IsAvailable          bool          `json:"is_available"`
	HasGpu               bool          `json:"has_gpu"`
	AllowCardano         bool          `json:"allow_cardano"`
	GpuInfo              []Gpu         `json:"gpu_info"`
	TokenomicsAddress    string        `json:"tokenomics_addrs"`
	TokenomicsBlockchain string        `json:"tokenomics_blockchain"`
	AvailableResources   FreeResources `json:"available_resources"`
	Services             []Services    `json:"services"`
	Timestamp            int64         `json:"timestamp,omitempty"`
}
```

- `Connection`: `TBD`

```
type Connection struct {
	models.BaseDBModel
	PeerID     string `json:"peer_id"`
	Multiaddrs string `json:"multiaddrs"`
}
```

- `PingResult`: The contains the details of the ping result.

```
type PingResult struct {
	RTT     time.Duration
	Success bool
	Error   error
}
```

- `Machines`: `TBD`
```
type Machines map[string]PeerData
```

- `KadDHTMachineUpdate`: This contains machine info for KAD-DHT.

```
// machine info for KAD-DHT
type KadDHTMachineUpdate struct {
	Data      []byte `json:"data"`
	Signature []byte `json:"signature"`
}
```

- `ElasticToken`: `TBD`

```
type ElasticToken struct {
	models.BaseDBModel
	NodeId      string
	Token       string
	ChannelName string
}
```

#### Onboarding

- `BlockchainAddressPrivKey`

```
// BlockchainAddressPrivKey holds Ethereum wallet address and private key from which the
// address is derived.
type BlockchainAddressPrivKey struct {
	Address    string `json:"address,omitempty"`
	PrivateKey string `json:"private_key,omitempty"`
	Mnemonic   string `json:"mnemonic,omitempty"`
}
```

- `CapacityForNunet`

```
// CapacityForNunet is a struct required in request body for the onboarding
type CapacityForNunet struct {
	Memory            int64   `json:"memory,omitempty"`
	CPU               int64   `json:"cpu,omitempty"`
	NTXPricePerMinute float64 `json:"ntx_price,omitempty"`
	Channel           string  `json:"channel,omitempty"`
	PaymentAddress    string  `json:"payment_addr,omitempty"`
	Cardano           bool    `json:"cardano,omitempty"`
	ServerMode        bool    `json:"server_mode,omitempty,"`
	IsAvailable       bool    `json:"is_available"`
}
```

- `Provisioned`

```
// Provisioned struct holds data about how much total resource
// host machine is equipped with
type Provisioned struct {
	CPU      float64 `json:"cpu,omitempty"`
	Memory   uint64  `json:"memory,omitempty"`
	NumCores uint64  `json:"total_cores,omitempty"`
}
```

- `Metadata`

```
// Metadata - machine metadata of onboarding parameters
type Metadata struct {
	Name            string `json:"name,omitempty"`
	UpdateTimestamp int64  `json:"update_timestamp,omitempty"`
	Resource        struct {
		MemoryMax int64 `json:"memory_max,omitempty"`
		TotalCore int64 `json:"total_core,omitempty"`
		CPUMax    int64 `json:"cpu_max,omitempty"`
	} `json:"resource,omitempty"`
	Available struct {
		CPU    int64 `json:"cpu,omitempty"`
		Memory int64 `json:"memory,omitempty"`
	} `json:"available,omitempty"`
	Reserved struct {
		CPU    int64 `json:"cpu,omitempty"`
		Memory int64 `json:"memory,omitempty"`
	} `json:"reserved,omitempty"`
	Network           string  `json:"network,omitempty"`
	PublicKey         string  `json:"public_key,omitempty"`
	NodeID            string  `json:"node_id,omitempty"`
	AllowCardano      bool    `json:"allow_cardano,omitempty"`
	GpuInfo           []Gpu   `json:"gpu_info,omitempty"`
	Dashboard         string  `json:"dashboard,omitempty"`
	NTXPricePerMinute float64 `json:"ntx_price,omitempty"`
}
```

- `OnboardingStatus`

```
type OnboardingStatus struct {
	Onboarded    bool   `json:"onboarded"`
	Error        error  `json:"error"`
	MachineUUID  string `json:"machine_uuid"`
	MetadataPath string `json:"metadata_path"`
	DatabasePath string `json:"database_path"`
}
```

- `LogBinAuth`: This stores the authorisation token for LogBin.

```
type LogBinAuth struct {
	models.BaseDBModel
	PeerID      string `json:"peer_id"`
	MachineUUID string `json:"machine_uuid"`
	Token       string `json:"token"`
}
```

#### Resource

- `GPU`: This contains GPU parameters.

```
type GPUVendor string

type GPU struct {
	// Self-reported index of the device in the system
	Index uint64
	// Model name of the GPU e.g. Tesla T4
	Name string
	// Maker of the GPU, e.g. NVidia, AMD, Intel
	Vendor GPUVendor
	// PCI address of the device, in the format AAAA:BB:CC.C
	// Used to discover the correct device rendering cards
	PCIAddress string
}
```

- `ExecutionResources`: `TBD` 

```
type ExecutionResources struct {
	// CPU units
	CPU float64 `json:"cpu,omitempty"`
	// Memory in bytes
	Memory uint64 `json:"memory,omitempty"`
	// Disk in bytes
	Disk uint64 `json:"disk,omitempty"`
	// GPU configurations
	GPUs []models.GPU `json:"gpus,omitempty"`
}
```

#### Spec_config

- `SpecConfig`: This allows arbitrary configuration to be defined as needed.

```
// SpecConfig represents a configuration for a spec
// A SpecConfig can be used to define an engine spec, a storage volume, etc.
type SpecConfig struct {
	// Type of the spec (e.g. docker, firecracker, storage, etc.)
	Type string `json:"type"`
	// Params of the spec
	Params map[string]interface{} `json:"params,omitempty"`
}
```

#### Storage

- `StorageVolume`: This contains the parameters related to the storage volume that is created by the DMS on the local machine.

```
// StorageVolume represents a prepared storage volume that can be mounted to an execution
type StorageVolume struct {
	// Type of the volume (e.g. bind)
	Type string `json:"type"`
	// Source path of the volume on the host
	Source string `json:"source"`
	// Target path of the volume in the execution
	Target string `json:"target"`
	// ReadOnly flag to mount the volume as read-only
	ReadOnly bool `json:"readonly"`
}
```

#### Telemetry Config

- `CollectorConfig`: This contains the parameters for a collector. 

```
type CollectorConfig struct {
	CollectorType     string
	CollectorEndpoint string
}
```

- `TelemetryConfig`: This defines the telemetry parameters such as obervability level, collector configurations etc. 

```
type TelemetryConfig struct {
	ServiceName        string
	GlobalEndpoint     string
	ObservabilityLevel int
	CollectorConfigs   map[string]CollectorConfig
}
```

ObservabilityLevel is an enum that defines the level of observability. Currently logging is done at these observability levels.

```
    TRACE ObservabilityLevel = 1
	DEBUG ObservabilityLevel = 2
	INFO  ObservabilityLevel = 3
	WARN  ObservabilityLevel = 4
	ERROR ObservabilityLevel = 5
	FATAL ObservabilityLevel = 6
```

#### Types

- `BaseDBModel`

```
// BaseDBModel is a base model for all entities. It'll be mainly used for database
// records.
type BaseDBModel struct {
	ID        string `gorm:"type:uuid"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
```

### 6. Testing

Test are defined in other packages where functionality is implemented

### 7. Proposed Functionality / Requirements 

#### List of issues

All issues that are related to the implementation of `models` package can be found below. These include any proposals for modifications to the package or new data structures needed to cover the requirements of other packages.

- [models package implementation]() `TBD`

##### `proposed` Encryption interfaces

These are placeholder interface definitions which will be developed in the future.

`Encryptor` must support encryption of files/directories
```
type Encryptor interface {
	Encrypt([]byte) ([]byte, error)
}
```

`Decryptor` must support decryption of files/directories
```
type Decryptor interface {
	Decrypt([]byte) ([]byte, error)
}
```

##### `proposed` Network types and methods 

This section contains the proposed data models and methods related to network functionality.

- `NetworkSpec`
```
// NetworkSpec is a stub. Please expand based on requirements.
type NetworkSpec struct {
}
```

- `NetConfig`
```
// NetConfig is a stub. Please expand it or completely change it based on requirements.
type NetConfig struct {
	NetworkSpec SpecConfig `json:"network_spec"` // Network specification
}
```

`NetConfig` struct will implement a `GetNetworkConfig` method which returns network configuration parameters.

```
func (nc *NetConfig) GetNetworkConfig() *SpecConfig {
	return &nc.NetworkSpec
}
```

- `NetworkStats`
```
// NetworkStats should contain all network info the user is interested in.
// for now there's only peerID and listening address but reachability, local and remote addr etc...
// can be added when necessary.
type NetworkStats struct {
	ID         string `json:"id"`
	ListenAddr string `json:"listen_addr"`
}
```

- `MessageInfo`
```
// MessageInfo is a stub. Please expand it or completely change it based on requirements.
type MessageInfo struct {
	Info string `json:"info"` // Message information
}
```

##### `proposed` Network configuration data type

- `MessageEnvelope`

```
type MessageType string

type MessageEnvelope struct {
	Type MessageType
	Data []byte
}
``` 

- `NetworkConfig`

```
type NetworkType string

type NetworkConfig struct {
	Type NetworkType

	// libp2p
	models.Libp2pConfig

	// nats
	NATSUrl string
}
```

- `Libp2pConfig`

// Libp2pConfig holds the libp2p configuration
type Libp2pConfig struct {
	DHTPrefix               string
	//crypto is Go-libp2p package that implements various cryptographic utilities 
	PrivateKey              crypto.PrivKey 
	BootstrapPeers          []multiaddr.Multiaddr
	Rendezvous              string
	Server                  bool
	Scheduler               *bt.Scheduler
	CustomNamespace         string
	ListenAddress           []string
	PeerCountDiscoveryLimit int
	PrivateNetwork          models.PrivateNetworkConfig
	GracePeriodMs           int
	GossipMaxMessageSize    int
}

- `PrivateNetworkConfig`
```
type PrivateNetworkConfig struct {
	// WithSwarmKey if true, DMS will try to fetch the key from
	// `<config_path>/swarm.key`.
	WithSwarmKey bool

	// ACL defines the access control list for the private network.
	ACL []multiaddr.Multiaddr
}
```

### 8. References
