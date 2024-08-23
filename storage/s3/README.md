# s3

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

This sub package is an implementation of `StorageProvider` interface for S3 storage.

### 2. Structure and Organisation

Here is quick overview of the contents of this pacakge:

* [README](https://gitlab.com/nunet/device-management-service/-/blob/develop/storage/s3/README.md): Current file which is aimed towards developers who wish to use and modify the functionality.

* [download](https://gitlab.com/nunet/device-management-service/-/blob/develop/storage/s3/download.go): This file defines functionality to allow download of files and directories from S3 buckets.

* [helpers](https://gitlab.com/nunet/device-management-service/-/blob/develop/storage/s3/helpers.go): This file defines helper functions for working with AWS S3 storage.

* [init](https://gitlab.com/nunet/device-management-service/-/blob/develop/storage/s3/init.go): This file initializes an Open Telemetry logger for the package.

* [s3_test](https://gitlab.com/nunet/device-management-service/-/blob/develop/storage/s3/s3_test.go): This file contains unit tests for the package functionality.

* [s3](https://gitlab.com/nunet/device-management-service/-/blob/develop/storage/s3/s3.go): This file defines methods to interact with S3 buckets using the AWS SDK. 

* [specs_decoder](https://gitlab.com/nunet/device-management-service/-/blob/develop/storage/s3/specs_decoder.go): This file defines S3 input source configuration with validation and decoding logic.

* [upload](https://gitlab.com/nunet/device-management-service/-/blob/develop/storage/s3/upload.go): This file defines S3 storage implementation for uploading files (including directories) from a local volume to an S3 bucket.


### 3. Class Diagram

The class diagram for the `s3` sub-package is shown below.

#### Source file

[s3 Class Diagram](https://gitlab.com/nunet/device-management-service/-/blob/develop/storage/s3/specs/class_diagram.puml)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/develop"
!$packageRelativePath = "/storage/s3"
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath
 
!include $packageUrlGitlab/specs/class_diagram.puml
```


### 4. Functionality

#### NewClient

* signature: `NewClient(config aws.Config, volController storage.VolumeController) (*S3Storage, error)` <br/>
* input #1:  `AWS configuration object` from AWS SDK <br/>
* input #2:  `Volume Controller object` <br/>
* output (sucess): new instance of type `storage.s3.S3Storage` <br/>
* output (error): error

`NewClient` is a constructor method. It creates a new instance of `storage.s3.S3Storage` struct.


#### Upload

For function signature refer to the package [readme](https://gitlab.com/nunet/device-management-service/-/blob/develop/storage/README.md#upload)

`Upload` uploads all files (recursively) from a local volume to an S3 bucket. It returns an error when
- there is error in decoding input specs
- there is an error in file system operations (ex - opening files on the local file system etc)
- there are errors from the AWS SDK during the upload process

#### Download

For function signature refer to the package [readme](https://gitlab.com/nunet/device-management-service/-/blob/develop/storage/README.md#download)

`Download` fetches files from a given S3 bucket. It can handle folders but except for x-directory. It depends on the file system provided by `storage.VolumeController`. It will return an error when
* there is error in decoding input specs
* there is error in creating a storage volume
* there is an issue in determining the target S3 objects based on the provided key
* there are errors in downloading individual S3 objects
* there are issues in creating directories or writing files to the local file system 
* the `storage.VolumeController` fails to lock the newly created storage volume

#### Size

For function signature refer to the package [readme](https://gitlab.com/nunet/device-management-service/-/blob/develop/storage/README.md#size)

`Size` determines the size of an object stored in an S3 bucket.

It will return an error when
- there is error in decoding input specs
- there is an issue in AWS API call to retrieve the object size

### 5. Data Types

- `storage.s3.S3Storage`: `TBD`
```
type S3Storage struct {
	*s3.Client
	volController storage.VolumeController
	downloader    *s3Manager.Downloader
	uploader      *s3Manager.Uploader
}
```

- `storage.s3.s3Object`: `TBD`
```
type s3Object struct {
	key       *string
	eTag      *string
	versionID *string
	size      int64
	isDir     bool
}
```

- `storage.s3.S3InputSource`: `TBD`
```
type S3InputSource struct {
	Bucket   string
	Key      string
	Filter   string
	Region   string
	Endpoint string
}
```

Refer to package readme for other data types.


### 6. Testing
The various unit tests for the package functionality are defined in `s3_test.go` file.

### 7. Proposed Functionality / Requirements 

#### List of issues

All issues that are related to the implementation of `storage` package can be found below. These include any proposals for modifications to the package or new data structures needed to cover the requirements of other packages.

- [storage package implementation]() `TBD`


### 8. References





