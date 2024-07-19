# Introduction

The storage package is responsible for disk storage management on each DMS (Device Management Service) for data related to DMS and jobs deployed by DMS. It primarily handles storage access to remote storage providers such as [AWS S3](https://aws.amazon.com/s3/), [IPFS](https://ipfs.tech/) etc. It also handles the control of storage volumes.

# Stucture and organisation

Here is quick overview of the contents of this pacakge:

* [README](https://gitlab.com/nunet/device-management-service/-/blob/develop/storage/README.md): Current file which is aimed towards developers who wish to use and modify the storage functionality.

* [storage](https://gitlab.com/nunet/device-management-service/-/blob/develop/storage/storage.go): This file defines the interface responsible for handling input/output operations of files with remote storage providers.

* [volumes](https://gitlab.com/nunet/device-management-service/-/blob/develop/storage/volumes.go): This file contains the interfaces and structs related to storage volumes.

* [basic_controller](https://gitlab.com/nunet/device-management-service/-/tree/428-implementation-of-volumecontroller-2/storage/basic_controller): This folder contains the basic implementation of `VolumeController` interface.

# Contributing

For guidelines of how to contribute, install and test the `device-management-service` component which contains `storage` package, please refer to package level documentation:

* DMS component level [../README.md](https://gitlab.com/nunet/device-management-service/-/blob/develop/README.md)
* Contribution guidelines [../CONTRIBUTING.md](https://gitlab.com/nunet/device-management-service/-/blob/develop/CONTRIBUTING.md)
* Code of conduct [../CODE_OF_CONDUCT.md](https://gitlab.com/nunet/device-management-service/-/blob/develop/CODE_OF_CONDUCT.md)
* [Secure coding guidelines](https://gitlab.com/nunet/documentation/-/wikis/secure-coding-guidelines)

# Specifications overview

* The specification of package functionality is described as test case definitions, maintained in repository [test-suite](https://gitlab.com/nunet/test-suite).
* The associated data models are specified and maintained in repository [open-api/platform-data-model/device-management-service/](https://gitlab.com/nunet/open-api/platform-data-model/-/tree/develop/device-management-service/).

Versioning and lifecycle of above mentioned specifications is aligned to the lifecycle and branching of the platform code (see [branching strategy](https://gitlab.com/nunet/documentation/-/wikis/GIT-Workflows#git-workflow-branching-strategy)):

* `develop` branches contain specifications of the functionality of current unstable branch of development at any given moment;
* `main` branches contain specifications of the current production version of the platform at any given moment in time;
* `proposed` branches contain new functionality and data model specifications, accepted for development, but not yet implemented.

The procedure to update the specifications is described in [Specification And Documentation Procedure](https://gitlab.com/nunet/team-processes-and-guidelines/-/blob/main/specification_and_documentation/README.md).


# Storage Functionality

The functionality with respect to `storage` package is offered by two main interfaces:
1. `StorageProvider`
2. `VolumeController`

These interfaces are described below.

## StorageProvider Interface

The `StorageProvider` interface handles the input and output operations of files with remote storage providers such as AWS S3 and IPFS. Basically it provides methods to upload or download data and also to check the size of a data source.

Its functionality is coupled with local mounted volumes, meaning that implementations will rely on mounted files to upload data and downloading data will result in a local mounted volume.

*Notes:*
* If needed, the availability-checking of a storage provider should be handled druing instantiation of the implementation.

* Any necessary authentication data should be provided within the `dms.models.SpecConfig` parameters

* The interface has been designed for file based transfer of data. It is not built with the idea of supporting streaming of data and non-file storage operations (e.g.: some databases). Assessing the feasiblity of such requirement if needed should be done while implementation.

The methods of `StorageProvider` are as follows:

### Upload

* signature: `Upload(vol dms.storage.StorageVolume, target dms.models.SpecConfig) -> (dms.models.SpecConfig, error)` <br/>
* input #1: storage volume from which data will be uploaded of type `dms.storage.StorageVolume` <br/>
* input #2: configuration parameters of specified storage provider of type `dms.models.SpecConfig` <br/>
* output (sucess): parameters related to storage provider like upload details/metadata etc of type `dms.models.SpecConfig` <br/>
* output (error): error message

`Upload` function uploads data from the storage volume provided as input to a given remote storage provider. The configuration of the storage provider is also provided as input to the function.

### Download

* signature: `Download(source dms.models.SpecConfig, outputPath string) -> (dms.storage.StorageVolume, error)` <br/>
* input #1: configuration parameters of specified storage provider of type `dms.models.SpecConfig` <br/>
* input #2: output path where downloaded data should be stored <br/>
* output (sucess): storage volume which has downloaded data of type `dms.storage.StorageVolume` <br/>
* output (error): error message

`Download` function downloads data from a given source, mounting it to a certain local path. The input configuration received will vary from provider to provider and hence it is left to be detailed during implementation.

It will return an error if the operation fails. Note that this can also happen if the user running DMS does not have access permission to the given path.

### Size

* signature: `Size(source dms.models.SpecConfig) -> (MiB, error)` <br/>
* input: configuration parameters of specified storage provider of type `dms.models.SpecConfig` <br/>
* output (sucess): size of the storage in Megabytes of type `uint64` <br/>
* output (error): error message

`Size` function returns the size of a given storage provider provided as input. It will return an error if the operation fails.

Note that this method may also be useful to check if a given source is available.

## VolumeController Interface

The `VolumeController` interface manages operations related to storage volumes which are data mounted to files/directories.

The methods of `VolumeController` are as follows:

### CreateVolume

* signature: `CreateVolume(volSource dms.storage.VolumeSource, opts ...dms.storage.CreateVolOpt) -> (dms.storage.StorageVolume, error)` <br/>
* input #1: predefined values of type `string` which specify the source of data (ex. IPFS etc)  <br/>
* input #2: optional parameter which can be passsed to set attributes or perform an operation on the storage volume<br/>
* output (sucess): storage volume of type `dms.storage.StorageVolume` <br/>
* output (error): error message

`CreateVolume` creates a directory where data can be stored, and returns a `StorageVolume` which contains the path to the directory. Note that `CreateVolume` does not insert any data within the directory. It's up to the caller to do that.

`VolumeSource` contains predefined constants to specify common sources like S3 but it's extensible if new sources need to be supported. Refer to [volumeSource.data.go](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/develop/device-management-service/storage/data/volumeSource.data.go) for the list of sources considered.

`CreateVolOpt` is a function type that modifies `storageVolume` object. It allows for arbitrary operations to be performed while creating volume like setting permissions, encryption etc.

`CreateVolume` will return an error if the operation fails. Note that this can also happen if the user running DMS does not have access permission to create volume at the given path.

### LockVolume

* signature: `LockVolume(pathToVol string, opts ...dms.storage.LockVolOpt) -> error` <br/>
* input #1: path to the volume  <br/>
* input #2: optional parameter which can be passsed to set attributes or perform an operation on the storage volume<br/>
* output (sucess): None <br/>
* output (error): error message

`LockVolume` makes the volume read-only. It should be used after all necessary data has been written to the volume. It also makes clear whether a volume will change state or not. This is very useful when we need to retrieve volume's CID which is immutable given a certain data.

`LockVolOpt` is a function type that modifies `storageVolume` object. It allows for arbitrary operations to be performed while locking the volume like setting permissions, encryption etc.

`LockVolume` will return an error if the operation fails.

### DeleteVolume

* signature: `DeleteVolume(identifier string, idType dms.storage.IDType) -> error` <br/>
* input #1: path to the volume or CID  <br/>
* input #2: integer value associated with the type of identifier<br/>
* output (error): error message

`DeleteVolume` function deletes everything within the root directory of a storage volume. It will return an error if the operation fails. Note that this can also happen if the user running DMS does not have the requisite access permissions.

The input can be a path or a Content ID (CID) depending on the identifier type passed.

`IDType` contains predefined integer values for different types of identifiers. Refer to [idType.data.go](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/develop/device-management-service/storage/data/idType.data.go) for the list of identifiers considered.

### ListVolumes

* signature: `ListVolumes() -> ([]dms.storage.StorageVolume, error)` <br/>
* input: None <br/>
* output (sucess): List of existing storage volumes of type `dms.storage.StorageVolume` <br/>
* output (error): error message

`ListVolumes` function fetches the list of existing storage volumes. It will return an error if the operation fails or if the user running DMS does not have the requisite access permissions.

### GetSize

* signature: `GetSize(identifier string, idType dms.storage.IDType) -> (int64, error)` <br/>
* input #1: path to the volume or CID  <br/>
* input #2: integer value associated with the type of identifier<br/>
* output (success): size of the volume
* output (error): error message

`GetSize` returns the size of a volume. The input can be a path or a Content ID (CID) depending on the identifier type passed. It will return an error if the operation fails.

`IDType` contains predefined integer values for different types of identifiers. Refer to [idType.data.go](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/develop/device-management-service/storage/data/idType.data.go) for reference data model.

## List of Data Types

`dms.storage.StorageVolume`: This struct contains parameters related to a storage volume such as path, CID etc. See [storageVolume.data.go](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/develop/device-management-service/storage/data/storageVolume.data.go) for reference data model.

`dms.models.SpecConfig`: This allows arbitrary configuration/parameters as needed during implementation of a specific storage provider. The parameters include authentication related data (if applicable). See [specConfig.data.go](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/develop/device-management-service/models/data/specConfig.data.go) for reference data model.

`dms.storage.VolumeSource`: This represents the source of data for a storage volume, for example IPFS, S3 etc. See [volumeSource.data.go](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/develop/device-management-service/storage/data/volumeSource.data.go) for reference data model.

`dms.storage.CreateVolOpt`: This allows arbitrary operations on `StorageVolume` to passed as input during volume creation. See [createVolOpt.data.go](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/develop/device-management-service/storage/data/createVolOpt.data.go) for reference data model.

`dms.storage.LockVolOpt`: This allows arbitrary operations on `StorageVolume` to passed as input during locking of volume. See [lockVolOpt.data.go](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/develop/device-management-service/storage/data/lockVolOpt.data.go) for reference data model.

`dms.storage.IDType`: This defines integer values for different types of identifiers of a storage volume. See [idType.data.go](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/develop/device-management-service/storage/data/idType.data.go) for reference data model.

`dms.models.EncryptionType`:  See [encryptionType.data.go](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/develop/device-management-service/models/data/encryptionType.data.go) for reference data model.




