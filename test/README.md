# Introduction

These are any tests that are not unit tests (at least this is how it was defined when Obsidian built these tests)

_Note: position these tests as per our test-matrix and triggers in to the correct place attn: @gabriel_

_Note by Dagim: suggest if security.go is deleted entirely with utils/cardano because those tests do not apply to the new dms._

`./test` directory of the package contains full test suite of DMS. 

## Run CLI Test Suite

This command will run the Command Line Interface Test suite inside the `./test` directory:
`go test -run `

## Run Security Test Suite

This command will run the Security Test suite inside the `./test` directory:
`go test -ldflags="-extldflags=-Wl,-z,lazy" -run=TestSecurity`

## Run all tests

This command will run all tests from root directory:
`sh run_all.sh`

After developing a new test suite or a test make sure that they are properly included with approprate flags and parameters into the `run_all.sh` file.

