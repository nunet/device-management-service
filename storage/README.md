# storage

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

The storage package is responsible for disk storage management on each DMS (Device Management Service) for data related to DMS and jobs deployed by DMS. It primarily handles storage access to remote storage providers such as [AWS S3](https://aws.amazon.com/s3/), [IPFS](https://ipfs.tech/) etc. It also handles the control of storage volumes.

### 2. Structure and organisation

Here is quick overview of the contents of this pacakge:

* [README](https://gitlab.com/nunet/device-management-service/-/blob/develop/storage/README.md): Current file which is aimed towards developers who wish to use and modify the package functionality.

* [storage](https://gitlab.com/nunet/device-management-service/-/blob/develop/storage/storage.go): This file defines the interface responsible for handling input/output operations of files with remote storage providers.

* [volumes](https://gitlab.com/nunet/device-management-service/-/blob/develop/storage/volumes.go): This file contains the interfaces and structs related to storage volumes.

_subpackages_
* [basic_controller](https://gitlab.com/nunet/device-management-service/-/tree/428-implementation-of-volumecontroller-2/storage/basic_controller): This folder contains the basic implementation of `VolumeController` interface.

* [s3](s3): This contains implementation of storage functionality for S3 storage bucket.

### 3. Class Diagram

The class diagram for the storage package is shown below.

#### Source file

[storage Class Diagram](https://gitlab.com/nunet/device-management-service/-/blob/develop/storage/specs/class_diagram.puml)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/develop"
!$packageRelativePath = "/storage"
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath
 
!include $packageUrlGitlab/specs/class_diagram.puml
```

### 4. Functionality

The functionality with respect to `storage` package is offered by two main interfaces:
1. `StorageProvider`
2. `VolumeController`

These interfaces are described below.

#### StorageProvider Interface

The `StorageProvider` interface handles the input and output operations of files with remote storage providers such as AWS S3 and IPFS. Basically it provides methods to upload or download data and also to check the size of a data source.

Its functionality is coupled with local mounted volumes, meaning that implementations will rely on mounted files to upload data and downloading data will result in a local mounted volume.

*Notes:*
* If needed, the availability-checking of a storage provider should be handled druing instantiation of the implementation.

* Any necessary authentication data should be provided within the `models.SpecConfig` parameters

* The interface has been designed for file based transfer of data. It is not built with the idea of supporting streaming of data and non-file storage operations (e.g.: some databases). Assessing the feasiblity of such requirement if needed should be done while implementation.

The methods of `StorageProvider` are as follows:

##### Upload

* signature: `Upload(ctx context.Context, vol StorageVolume, target *models.SpecConfig) (*models.SpecConfig, error)` <br/>

* input #1: Context object <br/>

* input #2: storage volume from which data will be uploaded of type `storage.StorageVolume` <br/>

* input #3: configuration parameters of specified storage provider of type `models.SpecConfig` <br/>

* output (sucess): parameters related to storage provider like upload details/metadata etc of type `models.SpecConfig` <br/>

* output (error): error message

`Upload` function uploads data from the storage volume provided as input to a given remote storage provider. The configuration of the storage provider is also provided as input to the function.

##### Download

* signature: `Download(ctx context.Context, source *models.SpecConfig) (StorageVolume, error)` <br/>

* input #1: Context object <br/>

* input #2: configuration parameters of specified storage provider of type `models.SpecConfig` <br/>

* output (sucess): storage volume which has downloaded data of type `storage.StorageVolume` <br/>

* output (error): error message

`Download` function downloads data from a given source, mounting it to a certain local path. The input configuration received will vary from provider to provider and hence it is left to be detailed during implementation.

It will return an error if the operation fails. Note that this can also happen if the user running DMS does not have access permission to the given path.

##### Size

* signature: `Size(ctx context.Context, source *models.SpecConfig) (uint64, error)` <br/>

* input #1: Context object <br/>

* input #2: configuration parameters of specified storage provider of type `models.SpecConfig` <br/>

* output (sucess): size of the storage in Megabytes of type `uint64` <br/>

* output (error): error message

`Size` function returns the size of a given storage provider provided as input. It will return an error if the operation fails.

Note that this method may also be useful to check if a given source is available.

#### VolumeController Interface

The `VolumeController` interface manages operations related to storage volumes which are data mounted to files/directories.

The methods of `VolumeController` are as follows:

##### CreateVolume

* signature: `CreateVolume(volSource storage.VolumeSource, opts ...storage.CreateVolOpt) -> (storage.StorageVolume, error)` <br/>
* input #1: predefined values of type `string` which specify the source of data (ex. IPFS etc)  <br/>
* input #2: optional parameter which can be passsed to set attributes or perform an operation on the storage volume<br/>
* output (sucess): storage volume of type `storage.StorageVolume` <br/>
* output (error): error message

`CreateVolume` creates a directory where data can be stored, and returns a `StorageVolume` which contains the path to the directory. Note that `CreateVolume` does not insert any data within the directory. It's up to the caller to do that.

`VolumeSource` contains predefined constants to specify common sources like S3 but it's extensible if new sources need to be supported. 

`CreateVolOpt` is a function type that modifies `storageVolume` object. It allows for arbitrary operations to be performed while creating volume like setting permissions, encryption etc.

`CreateVolume` will return an error if the operation fails. Note that this can also happen if the user running DMS does not have access permission to create volume at the given path.

##### LockVolume

* signature: `LockVolume(pathToVol string, opts ...storage.LockVolOpt) -> error` <br/>
* input #1: path to the volume  <br/>
* input #2: optional parameter which can be passsed to set attributes or perform an operation on the storage volume<br/>
* output (sucess): None <br/>
* output (error): error message

`LockVolume` makes the volume read-only. It should be used after all necessary data has been written to the volume. It also makes clear whether a volume will change state or not. This is very useful when we need to retrieve volume's CID which is immutable given a certain data.

`LockVolOpt` is a function type that modifies `storageVolume` object. It allows for arbitrary operations to be performed while locking the volume like setting permissions, encryption etc.

`LockVolume` will return an error if the operation fails.

##### DeleteVolume

* signature: `DeleteVolume(identifier string, idType storage.IDType) -> error` <br/>
* input #1: path to the volume or CID  <br/>
* input #2: integer value associated with the type of identifier<br/>
* output (error): error message

`DeleteVolume` function deletes everything within the root directory of a storage volume. It will return an error if the operation fails. Note that this can also happen if the user running DMS does not have the requisite access permissions.

The input can be a path or a Content ID (CID) depending on the identifier type passed.

`IDType` contains predefined integer values for different types of identifiers. 

##### ListVolumes

* signature: `ListVolumes() -> ([]storage.StorageVolume, error)` <br/>
* input: None <br/>
* output (sucess): List of existing storage volumes of type `storage.StorageVolume` <br/>
* output (error): error message

`ListVolumes` function fetches the list of existing storage volumes. It will return an error if the operation fails or if the user running DMS does not have the requisite access permissions.

##### GetSize

* signature: `GetSize(identifier string, idType storage.IDType) -> (int64, error)` <br/>
* input #1: path to the volume or CID  <br/>
* input #2: integer value associated with the type of identifier<br/>
* output (success): size of the volume
* output (error): error message

`GetSize` returns the size of a volume. The input can be a path or a Content ID (CID) depending on the identifier type passed. It will return an error if the operation fails.

`IDType` contains predefined integer values for different types of identifiers. 

### 5. Data Types

- `storage.StorageVolume`: This struct contains parameters related to a storage volume such as path, CID etc.

```
import "time"

// StorageVolume contains the location (FS path) of a directory where certain data may be stored
// and metadata about the volume itself + data (if any).
type StorageVolume struct {
	// CID is the content identifier of the storage volume.
	//
	// Warning: CID must be updated ONLY when locking volume (aka when volume was
	// is set to read-only)
	//
	// Be aware: Before relying on data's CID, be aware that it might be encrypted (
	// EncryptionType might be checked first if needed)
	CID string

	// Path points to the root of a DIRECTORY where data may be stored.
	Path string

	// ReadOnly indicates whether the storage volume is read-only or not.
	ReadOnly bool

	// Size is the size of the storage volume
	// Size int64

	// Private indicates whether the storage volume is private or not.
	// If it's private, it shouldn't be shared with other nodes and it shouldn't
	// be persisted after the job is finished.
	// Practical application: if private, peer maintaining it shouldn't publish
	// its CID as if it was available to be worked on by other jobs.
	Private bool

	// EncryptionType indicates the type of encryption used for the storage volume.
	// In case no encryption is used, the value will be EncryptionTypeNull
    EncryptionType models.EncryptionType 

	// CreatedAt represents the creation timestamp of the storage volume
	CreatedAt time.Time

	// UpdatedAt represents the last update timestamp of the storage volume
	UpdatedAt time.Time
}
```
`TBD` **Note: EncryptionType is not yet defined in models package**

- `models.SpecConfig`: This allows arbitrary configuration/parameters as needed during implementation of a specific storage provider. The parameters include authentication related data (if applicable).

- `storage.VolumeSource`: This represents the source of data for a storage volume, for example IPFS, S3 etc. 

```
// VolumeSource contains the source of data for a storage volume
type VolumeSource string

const (
	VolumeSourceUndefined VolumeSource = "volume-source-undefined"
	VolumeSourceS3        VolumeSource = "s3"
	VolumeSourceIPFS      VolumeSource = "ipfs"
	VolumeSourceJob       VolumeSource = "job" // when data is generated by a job
)
```

- `storage.CreateVolOpt`: This allows arbitrary operations on `storage.StorageVolume` to passed as input during volume creation. 

```
type CreateVolOpt func(*storage.StorageVolume)
```

`storage.LockVolOpt`: This allows arbitrary operations on `StorageVolume` to passed as input during locking of volume. 

```
// LockVolumeOpt allows arbitrary operation on the storage volume
// while making the volume read-only
type LockVolOpt func(*storage.StorageVolume)
```

`storage.IDType`: This defines integer values for different types of identifiers of a storage volume. 

```
type IDType int

const (
	IDTypeUndefined IDType = iota
	IDTypePath
	IDTypeCID
)
```

`models.EncryptionType`: `TBD`

**Note: The definition below should be moved to models package** 
```
type EncryptionType int

const (
	EncryptionTypeNull EncryptionType = iota
)
```

### 6. Testing
`TBD`

### 7. Proposed Functionality / Requirements 

#### List of issues

All issues that are related to the implementation of `storage` package can be found below. These include any proposals for modifications to the package or new data structures needed to cover the requirements of other packages.

- [storage package implementation]() `TBD`


### 8. References




