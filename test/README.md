# test

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

## Specification

### 1. Description

This package contains any tests that are not unit tests (at least this is how it was defined when Obsidian built these tests)

_Note: position these tests as per our test-matrix and triggers in to the correct place attn: @gabriel_

_Note by Dagim: suggest if security.go is deleted entirely with utils/cardano because those tests do not apply to the new dms._

`./test` directory of the package contains full test suite of DMS. 

### 2. Structure and organisation

Here is quick overview of the contents of this pacakge:


* [README](README.md): Current file which is aimed towards developers who wish to use and modify the package functionality.

* [run_all](run_all.sh): This file contains script to run CLI test and security test.

* [script](script.json): The file is a Plutus script encoded in CBOR hexadecimal format.

* [security](security.go): `TBD`

* [tester.addr](tester.addr): `TBD`

* [tester.sk](tester.sk): `TBD`

* [tester.vk](tester.vk): `TBD`


### 3. Functionality

#### Run CLI Test Suite

This command will run the Command Line Interface Test suite inside the `./test` directory:
`go test -run `

#### Run Security Test Suite

This command will run the Security Test suite inside the `./test` directory:
`go test -ldflags="-extldflags=-Wl,-z,lazy" -run=TestSecurity`

#### Run all tests

This command will run all tests from root directory:
`sh run_all.sh`

After developing a new test suite or a test make sure that they are properly included with approprate flags and parameters into the `run_all.sh` file.
