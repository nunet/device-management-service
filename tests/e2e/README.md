# Device Management Service Test Suite

## Introduction

This directory contains the test suite for the DMS.
The tests in this directory verify the functionality of the DMS by creating a network of nodes and testing their interactions
. These tests ensure that the core features of the DMS are working correctly in a multi-node environment.

## Prerequisites

Before running the tests, ensure you have the following prerequisites installed:

- GlusterFS
- Docker
- The DMS binary built and available in the test directory (optional for make rule usage)

### GlusterFS Setup

Ensure GlusterFS is installed and pull the container.

```bash
sudo modprobe fuse
sudo apt install glusterfs-client
docker pull ghcr.io/gluster/gluster-containers:fedora
```

## How to Run

Using Make:

```bash
sudo make itest
```

Using Go:

```bash
go test -tags=e2e ./...
```

To run a specific test:

```bash
go test -tags=e2e -run TestE2E/BasicTests
```

Available test suites:

- `BasicTests`: Tests basic node communication
- `DeploymentTests`: Tests deployment functionality
- `DeploymentWithVolumesTests`: Tests deployment with storage volumes
- `StorageTests`: Tests storage functionality

## Structure

The test suite is organized as follows:

- `e2e_test.go`: Entry point for all tests
- `suite_test.go`: Defines the test suite structure and common functionality
- `client_test.go`: Client implementation for interacting with DMS nodes
- `basic_test.go`: Basic communication tests
- `deployment_test.go`: Tests for deployment functionality
- `glusterfs_test.go`: GlusterFS setup for storage tests
- `storage_test.go`: Tests for storage functionality
- `volume_test.go`: Tests for volume management
- `utils_test.go`: Utility functions for tests
- `testdata/`: Test deployment ensembles

### Key Components

- **TestSuite**: The main test suite that sets up a network of nodes
- **Client**: A wrapper around the DMS CLI for testing
- **prefixWriter**: Used to prefix node logs with node identifiers

## Best Practices

### 1. Parallelism

Tests should be run in parallel to speed up the test suite.

To add a test for a new feature in parallel, here's the suggested workflow:

1. Create a new test file in the `tests/e2e` directory.
2. Define a runner function that takes a `*TestSuite` parameter:
   ```go
   func NewFeatureTest(suite *TestSuite) {
       // New feature test implementation
   }
   ```
3. Add your test to the `TestE2E` function in `e2e_test.go`:

   ```go
   t.Run("NewFeatureTests", func(t *testing.T) {
       t.Parallel()

       newFeatureTests := &TestSuite{
           numNodes:      3,  // Adjust as needed
           Name:          "new_feature_tests",
           restPortIndex: 8100,  // Use unique port ranges
           p2pPortIndex:  10700,  // Use unique port ranges
           runner:        NewFeatureTest,
       }
       suite.Run(t, newFeatureTests)
   })
   ```

### 2. Port Allocation

Each test suite must use unique port ranges to avoid conflicts:

- Allocate unique `restPortIndex` and `p2pPortIndex` for each test suite
- Increment by at least 3 (for a 3-node test) from the previous test suite's ports
- Document the port ranges used in comments to avoid future conflicts

### 3. Resource Management

Tests should properly clean up resources:

- Use `t.Cleanup()` for Docker containers and other external resources
- Ensure all nodes are properly shut down in the `TearDownSuite` method
- Verify resource allocation and deallocation in deployment tests

### 4. Test Data Organization

- Store test ensembles in `testdata/ensembles/`
- Use descriptive names for test files
- When using dynamic hostnames, use the `replaceHostnameInFile` utility

### 5. Error Handling

- Use `suite.Require()` instead of package-level assertions to ensure proper test failure tracking
- Add descriptive failure messages to assertions
- Use `suite.T().Logf()` for detailed logging during test execution

### 6. Test Isolation

- Each test suite should be completely independent
- Do not share state between test suites
- Use unique node directories and configurations

### 7. Timeouts and Retries

- Use `suite.Require().Eventually()` for operations that may take time to complete
- Set appropriate timeouts based on operation complexity
- Include descriptive timeout messages

### 7. Assertions

What and how you should assert deployments:

#### Before deploying:

1. (CP): resources
   1. free resources == onboarded resources
   2. allocated resources == 0
2. (CP): allocations/list is empty

#### After deploying:

> Note: it mostly does not apply for transient allocations as
> we usually use executions of short periods..
> to assert the following with task allocations,
> we need to use transients allocations that run for more than 2 minutes

1. (CP) assert allocation running (cmd: /dms/node/allocations)
   1. (CP) assert container running if possible
2. (CP) assert resources
   1. allocated resources increased
   2. free resources decreased
3. (Orchestrator) assert deployment status depending on allocations type
4. (Orchestrator) assert manifest
5. (CP) assert conn between containers is working for tests with multiple allocations!!!

#### After completed (if only tasks) or shutting it down:

1. (CP) assert allocation NOT listed (cmd: /dms/node/allocations)
   1. (CP) assert container not running if possible
2. (CP) assert resources
   1. allocated resources decreased
   2. free resources increased
3. (Orchestrator) assert deployment status depending on allocations types
4. (CP) assert subnet deleted (including tunneling iface)
