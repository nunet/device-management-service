# basic_controller

- [Project README](https://gitlab.com/nunet/device-management-service/-/blob/develop/README.md)
- [Release/Build Status](https://gitlab.com/nunet/device-management-service/-/releases)
- [Changelog](https://gitlab.com/nunet/device-management-service/-/blob/develop/CHANGELOG.md)
- [License](https://www.apache.org/licenses/LICENSE-2.0.txt)
- [Contribution Guidelines](https://gitlab.com/nunet/device-management-service/-/blob/develop/CONTRIBUTING.md)
- [Code of Conduct](https://gitlab.com/nunet/device-management-service/-/blob/develop/CODE_OF_CONDUCT.md)
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

This sub package offers a default implementation of the volume controller.

### 2. Structure and Organisation

Here is quick overview of the contents of this pacakge:

* [README](https://gitlab.com/nunet/device-management-service/-/blob/develop/storage/basic_controller/README.md): Current file which is aimed towards developers who wish to use and modify the functionality.

* [basic_controller](https://gitlab.com/nunet/device-management-service/-/blob/develop/storage/basic_controller/basic_controller.go): This file implements the methods for `VolumeController` interface.

* [basic_controller_test](https://gitlab.com/nunet/device-management-service/-/blob/develop/storage/basic_controller/basic_controller_test.go): This file contains the unit tests for the methods of `VolumeController` interface.

### 3. Class Diagram

The class diagram for the `basic_controller` sub-package is shown below.

#### Source file

[basic_controller Class Diagram](https://gitlab.com/nunet/device-management-service/-/blob/develop/storage/basic_controller/specs/class_diagram.puml?ref_type=heads)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/develop"
!$packageRelativePath = "/storage/basic_controller"
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath
 
!include $packageUrlGitlab/specs/class_diagram.puml
```


### 4. Functionality

#### NewDefaultVolumeController

* signature: `NewDefaultVolumeController(db *gorm.DB, volBasePath string, fs afero.Fs) -> (storage.basic_controller.BasicVolumeController, error)` <br/>
* input #1: local database instance of type `*gorm.DB` <br/>
* input #2: base path of the volumes <br/>
* input #3: file system instance of type `afero.FS` <br/>
* output (sucess): new instance of type `storage.basic_controller.BasicVolumeController` <br/>
* output (error): error

`NewDefaultVolumeController` returns a new instance of `storage.basic_controller.BasicVolumeController` struct. 

`BasicVolumeController` is the default implementation of the `VolumeController` interface. It persists storage volumes information in the local database.

#### CreateVolume

For function signature refer to the package [readme](https://gitlab.com/nunet/device-management-service/-/blob/develop/storage/README.md#createvolume)

`CreateVolume` creates a new storage volume given a storage source (S3, IPFS, job, etc). The creation of a storage volume effectively creates an empty directory in the local filesystem and writes a record in the database.

The directory name follows the format: `<volSource> + "-" + <name>`  where `name` is random.

`CreateVolume` will return an error if there is a failure in
* creation of new directory
* creating a database entry

#### LockVolume

For function signature refer to the package [readme](https://gitlab.com/nunet/device-management-service/-/blob/develop/storage/README.md#lockvolume)

`LockVolume` makes the volume read-only, not only changing the field value but also changing file permissions. It should be used after all necessary data has been written to the volume. It optionally can also set the CID and mark the volume as private

`LockVolume` will return an error when
* No storage volume is found at the specified
* There is error in saving the updated volume in the database
* There is error in updating file persmissions

#### DeleteVolume

For function signature refer to the package [readme](https://gitlab.com/nunet/device-management-service/-/blob/develop/storage/README.md#deletevolume)

`DeleteVolume` deletes a given storage volume record from the database. The identifier can be a path of a volume or a Content ID (CID). Therefore, records for both will be deleted.

It will return an error when
* Input has incorrect identifier
* There is failure in deleting the volume
* No volume is found

#### ListVolumes

For function signature refer to the package [readme](https://gitlab.com/nunet/device-management-service/-/blob/develop/storage/README.md#listvolumes)

`ListVolumes` function returns a list of all storage volumes stored on the database.

It will return an error when no storage volumes exist.

#### GetSize

For function signature refer to the package [readme](https://gitlab.com/nunet/device-management-service/-/blob/develop/storage/README.md#getsize)

`GetSize` returns the size of a volume. The input can be a path or a Content ID (CID).

It will return an error if the operation fails due to:
* error while querying database
* volume not found for given identifier
* unsupported identifed provided as input
* error while caculating size of directory

#### Custom configuration Parameters

Both `CreateVolume` and `LockVolume` allow for custom configuration of storage volumes via optional parameters. Below is the list of available parameters that can be used:

`WithPrivate()` - Passing this as an input parameter designates a given volume as private. It can be used both when creating or locking a volume.

`WithCID(cid string)` - This can be used as an input parameter to set the CID of a given volume during the lock volume operation.

### 5. Data Types

`storage.basic_controller.BasicVolumeController`: This struct manages implementation of `VolumeController` interface methods. 

```
// BasicVolumeController is the default implementation of the VolumeController.
// It persists storage volumes information in the local database.
type BasicVolumeController struct {
	// db is where all volumes information is stored.
	db *gorm.DB

	// basePath is the base path where volumes are stored under
	basePath string

	// file system to act upon
	fs afero.Fs
}
```

Refer to package readme for other data types.


### 6. Testing
The unit tests for the package functionality are defined in `*_test.go` file.

### 7. Proposed Functionality / Requirements 

#### List of issues

All issues that are related to the implementation of `storage` package can be found below. These include any proposals for modifications to the package or new data structures needed to cover the requirements of other packages.

- [storage package implementation]() `TBD`


### 8. References







