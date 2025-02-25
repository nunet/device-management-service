// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build e2e || !unit

package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/shirou/gopsutil/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/dms"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	"gitlab.com/nunet/device-management-service/internal"
	"gitlab.com/nunet/device-management-service/internal/config"
)

func TestDMS(t *testing.T) {
	os.Setenv("GOLOG_LOG_LEVEL", "info")

	// setup the temporary data directories for each dms
	const dms1RootDir = "dms1_root_dir"
	const dms2RootDir = "dms2_root_dir"
	const dms3RootDir = "dms3_root_dir"
	currentDir := getCurrentFileDirectory()
	assert.NotEmpty(t, currentDir)

	// first clear any temp data directory
	// since we run 3 processes, delete only in main process
	if os.Getenv("NEW_PROCESS") != "node2" && os.Getenv("NEW_PROCESS") != "node3" {
		// kill previous processes if any
		data, err := os.ReadFile(filepath.Join(currentDir, dms2RootDir, "proc.pid"))
		if err == nil {
			pid, err := strconv.Atoi(string(data))
			if err == nil {
				err := killProcess(int32(pid))
				if err != nil {
					t.Logf("no process to kill: %d : %v", pid, err)
				}
			}
		}

		data, err = os.ReadFile(filepath.Join(currentDir, dms3RootDir, "proc.pid"))
		if err == nil {
			pid, err := strconv.Atoi(string(data))
			if err == nil {
				err := killProcess(int32(pid))
				if err != nil {
					t.Logf("no process to kill: %d : %v", pid, err)
				}
			}
		}

		os.RemoveAll(filepath.Join(currentDir, dms1RootDir))
		os.RemoveAll(filepath.Join(currentDir, dms2RootDir))
		os.RemoveAll(filepath.Join(currentDir, dms3RootDir))
	}

	dms1Pass := "1234"
	dms2Pass := "4567"
	dms3Pass := "7890"

	// due to an issue related to broadcasting, which most probably is from the pubsub package
	// we run the second dms instance in another process and this fixes the issue.
	// please dont move this section and leave it here
	dms3Config := createConfig(dms3RootDir, 8092, "/ip4/127.0.0.1/tcp/10684", []string{})
	dms2Config := createConfig(dms2RootDir, 8091, "/ip4/127.0.0.1/tcp/10683", []string{})
	dms1Config := createConfig(dms1RootDir, 8089, "/ip4/127.0.0.1/tcp/10689", []string{})

	if os.Getenv("NEW_PROCESS") == "node2" {
		dms2Config.BootstrapPeers = []string{os.Getenv("BOOTSTRAP_NODE")}
		dms2, err := dms.NewDMS(dms2Config, dms2Pass, "dms")
		require.NoError(t, err)
		assert.NotNil(t, dms2)

		err = dms2.Run()
		require.NoError(t, err)
		select {}
	}

	if os.Getenv("NEW_PROCESS") == "node3" {
		dms3Config.BootstrapPeers = []string{os.Getenv("BOOTSTRAP_NODE")}
		dms3, err := dms.NewDMS(dms3Config, dms3Pass, "dms")
		require.NoError(t, err)
		assert.NotNil(t, dms3)

		err = dms3.Run()
		require.NoError(t, err)

		select {}
	}

	clidms1 := newCLI(dms1Config)

	// create keys and caps for dms1
	clidms1.newKey(t, "dms", dms1Pass)
	clidms1.newKey(t, "user", dms1Pass)
	clidms1.newCap(t, "dms", dms1Pass)
	clidms1.newCap(t, "user", dms1Pass)
	dms1DID := clidms1.getDID(t, "dms.cap", dms1Pass)
	dms1UserDID := clidms1.getDID(t, "user.cap", dms1Pass)
	clidms1.addRootAnchorToDMS(t, dms1UserDID, dms1Pass)

	clidms2 := newCLI(dms2Config)

	// create keys and caps for dms2
	clidms2.newKey(t, "dms", dms2Pass)
	clidms2.newKey(t, "user", dms2Pass)
	clidms2.newCap(t, "dms", dms2Pass)
	clidms2.newCap(t, "user", dms2Pass)
	dms2DID := clidms2.getDID(t, "dms.cap", dms2Pass)
	dms2UserDID := clidms2.getDID(t, "user.cap", dms2Pass)
	clidms2.addRootAnchorToDMS(t, dms2UserDID, dms2Pass)

	clidms3 := newCLI(dms3Config)

	// create keys and caps for dms3
	clidms3.newKey(t, "dms", dms3Pass)
	clidms3.newKey(t, "user", dms3Pass)
	clidms3.newCap(t, "dms", dms3Pass)
	clidms3.newCap(t, "user", dms3Pass)
	dms3DID := clidms3.getDID(t, "dms.cap", dms3Pass)
	dms3UserDID := clidms3.getDID(t, "user.cap", dms3Pass)
	clidms3.addRootAnchorToDMS(t, dms3UserDID, dms3Pass)

	// grant each other
	grantToken12 := clidms1.grant(t, dms2UserDID, dms1Pass)
	grantToken13 := clidms1.grant(t, dms3UserDID, dms1Pass)
	clidms1.anchorDMS(t, grantToken12, "dms", "require", dms1Pass)
	clidms1.anchorDMS(t, grantToken13, "dms", "require", dms1Pass)
	clidms2.anchorDMS(t, grantToken12, "user", "provide", dms2Pass)
	clidms3.anchorDMS(t, grantToken13, "user", "provide", dms3Pass)

	grantToken21 := clidms2.grant(t, dms1UserDID, dms2Pass)
	grantToken23 := clidms2.grant(t, dms3UserDID, dms2Pass)
	clidms1.anchorDMS(t, grantToken21, "user", "provide", dms1Pass)
	clidms2.anchorDMS(t, grantToken21, "dms", "require", dms2Pass)
	clidms2.anchorDMS(t, grantToken23, "dms", "require", dms2Pass)
	clidms3.anchorDMS(t, grantToken23, "dms", "provide", dms3Pass)

	grantToken32 := clidms3.grant(t, dms2UserDID, dms3Pass)
	clidms2.anchorDMS(t, grantToken32, "dms", "provide", dms2Pass)
	clidms3.anchorDMS(t, grantToken32, "dms", "require", dms3Pass)

	grantToken31 := clidms3.grant(t, dms1UserDID, dms3Pass)
	clidms1.anchorDMS(t, grantToken31, "dms", "provide", dms1Pass)
	clidms3.anchorDMS(t, grantToken31, "dms", "require", dms3Pass)

	delegToken1 := clidms1.delegateToDMS(t, dms1DID, dms1Pass)
	delgToken2 := clidms2.delegateToDMS(t, dms2DID, dms2Pass)
	delgToken3 := clidms3.delegateToDMS(t, dms3DID, dms3Pass)

	fmt.Println("DELEGATE TO DMS TOKEN 1", delegToken1)
	fmt.Println("DELEGATE TO DMS TOKEN 2", delgToken2)
	fmt.Println("DELEGATE TO DMS TOKEN 3", delgToken3)

	clidms1.anchorDMS(t, delegToken1, "dms", "provide", dms1Pass)
	clidms2.anchorDMS(t, delgToken2, "dms", "provide", dms2Pass)
	clidms3.anchorDMS(t, delgToken3, "dms", "provide", dms3Pass)

	// create a dms instance
	dms1, err := dms.NewDMS(dms1Config, dms1Pass, "dms")
	require.NoError(t, err)
	assert.NotNil(t, dms1)
	// run dms
	err = dms1.Run()
	require.NoError(t, err)
	time.Sleep(5 * time.Second)

	// get dms1 multiaddr and construct a bootstrap so dms2 can connect to
	multiAddr, err := dms1.P2P.GetMultiaddr()
	require.NoError(t, err)

	bootstrap := []string{}
	for _, v := range multiAddr {
		bootstrap = append(bootstrap, v.String())
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestDMS")
	cmd.Env = append(os.Environ(), "NEW_PROCESS=node2", "BOOTSTRAP_NODE="+bootstrap[0])
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start Node 2 subprocess: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
	}()

	// write the pid to a file and kill the process if its running
	err = os.WriteFile(filepath.Join(currentDir, dms2RootDir, "proc.pid"), []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0o700)
	require.NoError(t, err)

	go func() {
		<-internal.ShutdownChan
		_ = cmd.Process.Kill()
		os.Exit(1)
	}()

	cmd3 := exec.Command(os.Args[0], "-test.run=TestDMS")
	cmd3.Env = append(os.Environ(), "NEW_PROCESS=node3", "BOOTSTRAP_NODE="+bootstrap[0])
	cmd3.Stdout = os.Stdout
	cmd3.Stderr = os.Stderr
	if err := cmd3.Start(); err != nil {
		t.Fatalf("Failed to start Node 3 subprocess: %v", err)
	}
	defer func() {
		_ = cmd3.Process.Kill()
	}()

	// write the pid to a file and kill the process if its running
	err = os.WriteFile(filepath.Join(currentDir, dms3RootDir, "proc.pid"), []byte(fmt.Sprintf("%d", cmd3.Process.Pid)), 0o700)
	require.NoError(t, err)

	go func() {
		<-internal.ShutdownChan
		_ = cmd3.Process.Kill()
		os.Exit(1)
	}()

	time.Sleep(10 * time.Second)

	require.Eventually(t, func() bool {
		knownPeers, err := dms1.P2P.KnownPeers()
		require.NoError(t, err)
		return len(knownPeers) == 3
	}, 15*time.Second, 100*time.Millisecond, "Known peers is not 3 within the expected time")

	// broadcast
	result := clidms1.broadcast(t, dms1Pass)
	fmt.Println("BROADCAST RESULT", result)
	assert.Equal(t, 3, countDIDOccurrences(result))

	// onboard
	clidms1.onboard(t, dms1Pass)
	clidms2.onboard(t, dms2Pass)
	clidms3.onboard(t, dms3Pass)
	// check onboarded status

	// get the free resources before we deploy anything
	freeResourcesBeforeDeplyment, err := dms1.Node.ResourceManager().GetFreeResources(context.Background())
	require.NoError(t, err)

	pingResult := clidms2.connect(t, dms2Pass, dms1.P2P.Host.ID().String())
	assert.Contains(t, pingResult, `"Status": "CONNECTED"`)

	deploymentResult := clidms2.deploy(t, dms2Pass, filepath.Join(currentDir, "nginx.yaml"))
	assert.Contains(t, deploymentResult, `"Status": "OK"`)
	manifestID := extractEnsembleID(deploymentResult)

	// check if we have got the bid request
	require.Eventually(t, func() bool {
		bidRequests := dms1.Node.GetBidRequests()
		return len(bidRequests) == 1
	}, 60*time.Second, 100*time.Millisecond, "Bid requests for nginx.yaml deployment were not populated within the expected time")

	// check the status of deployment until its running
	require.Eventually(t, func() bool {
		status := clidms2.deploymentStatus(t, dms2Pass, manifestID)
		extractedStatus := extractStatus(status)

		return extractedStatus == "Running"
	}, 60*time.Second, 100*time.Millisecond, "Deployment of nginx.yaml is not running within expected time")

	time.Sleep(2 * time.Second)

	allAllocations := dms1.Node.Allocator().GetAllocations()
	require.Len(t, allAllocations, 1)

	// get the only running allocation
	var alloc *jobs.Allocation
	for _, v := range allAllocations {
		alloc = v
		break
	}

	// check the node allocations and inspect the status until its running
	require.Eventually(t, func() bool {
		allocStatus := alloc.Status(context.Background())
		return string(allocStatus.Status) == "running"
	}, 60*time.Second, 100*time.Millisecond, "Allocation for nginx.yaml status is still not running  after 60 seconds")

	// free resources should be different after we deployed
	freeResourceWhileDeplymentRunning, err := dms1.Node.ResourceManager().GetFreeResources(context.Background())
	require.NoError(t, err)
	assert.False(t, freeResourceWhileDeplymentRunning.Equal(freeResourcesBeforeDeplyment.Resources))

	// alloc1 should not have port mapping - alloc2 should have 1
	ports := alloc.GetPortMapping()
	if strings.Contains(alloc.ID, "alloc1") {
		require.Empty(t, ports)
	} else {
		require.Equal(t, 1, len(ports))
	}

	// finally shutdown the deployment
	shutdownRes := clidms2.shutdownDeployment(t, dms2Pass, manifestID)
	assert.Contains(t, shutdownRes, `"Error": ""`)

	// TODO: fix this logic once we handle status properly
	// require.Eventually(t, func() bool {
	// 	allocStatus := allAllocations[0].Status(context.TODO())
	// 	return string(allocStatus.Status) == "stopped"
	// }, 20*time.Second, 100*time.Millisecond, "Local allocation should not be running")

	freeResourcesAfterDeplyment, err := dms1.Node.ResourceManager().GetFreeResources(context.Background())
	require.NoError(t, err)

	// free resources after deployment is shutdown should be equal to before running the deployment
	assert.True(t, freeResourcesAfterDeplyment.Equal(freeResourcesBeforeDeplyment.Resources))

	time.Sleep(1 * time.Second)

	// check if the allocation was properlly removed
	allAllocations = dms1.Node.Allocator().GetAllocations()
	require.Len(t, allAllocations, 0)

	// run hello world that exists
	deployment2Result := clidms2.deploy(t, dms2Pass, filepath.Join(currentDir, "hello.yaml"))
	assert.Contains(t, deployment2Result, `"Status": "OK"`)
	manifest2ID := extractEnsembleID(deployment2Result)
	require.Eventually(t, func() bool {
		status := clidms2.deploymentStatus(t, dms2Pass, manifest2ID)
		extractedStatus := extractStatus(status)
		t.Log("second ensemble status", extractedStatus)
		// TODO: change the following to Completed
		// after we add the completed status
		allAllocations = dms1.Node.Allocator().GetAllocations()
		return extractedStatus == "Running" && len(allAllocations) == 1
	}, 60*time.Second, 100*time.Millisecond, "failed to run second deployment hello.yaml within the expected time")

	// get the only running allocation
	var helloAlloc *jobs.Allocation
	for _, v := range allAllocations {
		helloAlloc = v
		break
	}

	// get allocations port mappings
	ports = helloAlloc.GetPortMapping()
	require.NotEmpty(t, ports)
	require.Equal(t, ports[8080], 80)

	// shutdown previous allocations
	shutdownRes = clidms2.shutdownDeployment(t, dms2Pass, manifest2ID)
	require.Contains(t, shutdownRes, `"Error": ""`)

	time.Sleep(2 * time.Second)

	revokeToken := clidms1.revokeToken(t, dms1Pass, grantToken12)
	clidms1.anchorBehaviour(t, dms1Pass, revokeToken)

	time.Sleep(3 * time.Second)

	_, err = clidms2.hello(t, dms2Pass, dms1DID)
	require.Error(t, err, "request failed with status code: 408")

	dms1.Stop()
}

func createConfig(dmsRootDir string, restPort uint32, p2pListenAddr string, bootstrap []string) *config.Config {
	currentDir := getCurrentFileDirectory()

	return &config.Config{
		General: config.General{
			UserDir:                filepath.Join(currentDir, dmsRootDir),
			WorkDir:                filepath.Join(currentDir, dmsRootDir, "work_dir"),
			DataDir:                filepath.Join(currentDir, dmsRootDir, "data_dir"),
			HostCountry:            "NL",
			HostCity:               "Amsterdam",
			HostContinent:          "Europe",
			PortAvailableRangeFrom: 1024,
			PortAvailableRangeTo:   10000,
			Debug:                  true,
		},
		Rest: config.Rest{
			Addr: "localhost",
			Port: restPort,
		},
		Job: config.Job{
			AllowPrivilegedDocker: false,
		},
		P2P: config.P2P{
			ListenAddress:   []string{p2pListenAddr},
			BootstrapPeers:  bootstrap,
			Memory:          1024,
			FileDescriptors: 10444,
		},
		Observability: config.Observability{
			FlushInterval:        3,
			ElasticsearchEnabled: false,
			ElasticsearchURL:     "https://telemetry.nunet.io",
			ElasticsearchIndex:   "nunet-dms",
			LogLevel:             "debug",
			ElasticsearchAPIKey:  os.Getenv("ES_API"),
			LogFile:              filepath.Join(currentDir, dmsRootDir, "logs.txt"),
		},
		Profiler: config.Profiler{},
		APM: config.APM{
			ServerURL:   "https://apm.telemetry.nunet.io",
			ServiceName: "nunet-dms",
			APIKey:      os.Getenv("ES_API"),
			Environment: "production",
		},
	}
}

func killProcess(pid int32) error {
	p := getProc(pid)
	if p != nil {
		pname, err := p.Name()
		if err == nil && pname == "testbinary" {
			return p.Kill()
		}
	}

	return fmt.Errorf("process not found")
}

func getProc(pid int32) *process.Process {
	processes, err := process.Processes()
	if err != nil {
		return nil
	}
	for _, p := range processes {
		if p.Pid == pid {
			return p
		}
	}

	return nil
}
