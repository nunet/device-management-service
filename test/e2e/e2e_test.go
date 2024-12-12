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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/dms"
	"gitlab.com/nunet/device-management-service/internal"
	"gitlab.com/nunet/device-management-service/internal/config"
)

func TestDMS(t *testing.T) {
	os.Setenv("GOLOG_LOG_LEVEL", "info")

	// setup the temporary data directories for each dms
	const dms1RootDir = "dms1_root_dir"
	const dms2RootDir = "dms2_root_dir"
	currentDir := getCurrentFileDirectory()
	assert.NotEmpty(t, currentDir)

	// first clear any temp data directory
	// since we run 2 processes, delete only in main process
	if os.Getenv("NEW_PROCESS") != "node2" {
		os.RemoveAll(filepath.Join(currentDir, dms1RootDir))
		os.RemoveAll(filepath.Join(currentDir, dms2RootDir))
	}

	dms1Pass := "1234"
	dms2Pass := "4567"

	// due to an issue related to broadcasting, which most probably is from the pubsub package
	// we run the second dms instance in another process and this fixes the issue.
	// please dont move this section and leave it here
	if os.Getenv("NEW_PROCESS") == "node2" {
		dms2Config := createConfig(dms2RootDir, 8091, "/ip4/127.0.0.1/tcp/10350", []string{})
		// clidms2 := newCMD(dms2Config)
		dms2Config.BootstrapPeers = []string{os.Getenv("BOOTSTRAP_NODE")}
		dms2, err := dms.NewDMS(dms2Config, dms2Pass, "dms")
		require.NoError(t, err)
		assert.NotNil(t, dms2)

		err = dms2.Run()
		require.NoError(t, err)
		select {}
	}

	dms1Config := createConfig(dms1RootDir, 8089, "/ip4/127.0.0.1/tcp/10349", []string{})
	clidms1 := newCLI(dms1Config)

	// create keys and caps for dms1
	clidms1.newKey(t, "dms", dms1Pass)
	clidms1.newKey(t, "user", dms1Pass)
	clidms1.newCap(t, "dms", dms1Pass)
	clidms1.newCap(t, "user", dms1Pass)
	dms1DID := clidms1.getDID(t, "dms.cap", dms1Pass)
	dms1UserDID := clidms1.getDID(t, "user.cap", dms1Pass)
	clidms1.addRootAnchorToDMS(t, dms1UserDID, dms1Pass)

	dms2Config := createConfig(dms2RootDir, 8091, "/ip4/127.0.0.1/tcp/10350", []string{})
	clidms2 := newCLI(dms2Config)

	// create keys and caps for dms2
	clidms2.newKey(t, "dms", dms2Pass)
	clidms2.newKey(t, "user", dms2Pass)
	clidms2.newCap(t, "dms", dms2Pass)
	clidms2.newCap(t, "user", dms2Pass)
	dms2DID := clidms2.getDID(t, "dms.cap", dms2Pass)
	dms2UserDID := clidms2.getDID(t, "user.cap", dms2Pass)
	clidms2.addRootAnchorToDMS(t, dms2UserDID, dms2Pass)

	// grant each other
	token1 := clidms1.grant(t, dms2UserDID, dms1Pass)
	token2 := clidms2.grant(t, dms1UserDID, dms2Pass)

	clidms1.anchorDMS(t, token1, "dms", "require", dms1Pass)
	clidms2.anchorDMS(t, token2, "dms", "require", dms2Pass)
	clidms1.anchorDMS(t, token2, "user", "provide", dms1Pass)
	clidms2.anchorDMS(t, token1, "user", "provide", dms2Pass)

	token1 = clidms1.delegateToDMS(t, dms1DID, dms1Pass)
	token2 = clidms2.delegateToDMS(t, dms2DID, dms2Pass)

	fmt.Println("DELEGATE TO DMS TOKEN 1", token1)
	fmt.Println("DELEGATE TO DMS TOKEN 2", token2)

	clidms1.anchorDMS(t, token1, "dms", "provide", dms1Pass)
	clidms2.anchorDMS(t, token2, "dms", "provide", dms2Pass)

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

	go func() {
		<-internal.ShutdownChan
		_ = cmd.Process.Kill()
		os.Exit(1)
	}()

	require.Eventually(t, func() bool {
		knownPeers, err := dms1.P2P.KnownPeers()
		require.NoError(t, err)
		return len(knownPeers) == 2
	}, 10*time.Second, 100*time.Millisecond, "Known peers is not 2 within the expected time")

	// broadcast
	result := clidms1.broadcast(t, dms1Pass)
	// should have both DIDs
	assert.Equal(t, 2, countDIDOccurrences(result))

	// onboard
	clidms1.onboard(t, dms1Pass)
	clidms2.onboard(t, dms2Pass)
	// check onboarded status

	pingResult := clidms2.connect(t, dms2Pass, dms1.P2P.Host.ID().String())
	assert.Contains(t, pingResult, `"Status": "CONNECTED"`)
	deploymentResult := clidms2.deploy(t, dms2Pass, filepath.Join(currentDir, "nginx.yaml"))
	assert.Contains(t, deploymentResult, `"Status": "OK"`)
	manifestID := extractEnsembleID(deploymentResult)

	// check if we have got the bid request
	require.Eventually(t, func() bool {
		bidRequests := dms1.Node.GetBidRequests()
		return len(bidRequests) == 1
	}, 60*time.Second, 100*time.Millisecond, "Bid requests were not populated within the expected time")

	// check the status of deployment until its running
	require.Eventually(t, func() bool {
		status := clidms2.deploymentStatus(t, dms2Pass, manifestID)
		extractedStatus := extractStatus(status)

		return extractedStatus == "Running"
	}, 180*time.Second, 100*time.Millisecond, "failed to run within the expected time")

	allAllocations := dms1.Node.GetAllocations()
	require.Len(t, allAllocations, 1)

	// check the node allocations and inspect the status until its running
	require.Eventually(t, func() bool {
		allocStatus := allAllocations[0].Status(context.TODO())
		return string(allocStatus.Status) == "running"
	}, 120*time.Second, 100*time.Millisecond, "Local allocation status not running in expected time")

	// finally kill the container
	shutdownRes := clidms2.shutdownDeployment(t, dms2Pass, manifestID)
	assert.Contains(t, shutdownRes, `"Error": ""`)

	// manually shut down the allocation
	err = allAllocations[0].Stop(context.TODO())
	require.NoError(t, err)
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
			ElasticsearchEnabled: true,
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
