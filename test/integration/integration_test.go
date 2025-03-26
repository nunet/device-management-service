// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed
// under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
// either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build integration || !unit

package itest

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"

	"gitlab.com/nunet/device-management-service/types"

	"github.com/stretchr/testify/suite"

	"github.com/gin-gonic/gin"
)

const (
	binaryName = "dms"
	NumNodes   = 3
)

// TestSuite defines our end-to-end test suite.
type TestSuite struct {
	suite.Suite

	currentDir     string
	testDataDir    string
	bootstrapPeers []string
	nodes          map[int]*mockNode
	grantTokens    map[int]map[int]string // map[nodeIndex]map[otherNodeIndex]grantToken
}

var dmsTestSuite = new(TestSuite)

type prefixWriter struct {
	prefix string
	w      io.Writer
}

func (pw *prefixWriter) Write(p []byte) (n int, err error) {
	lines := strings.Split(string(p), "\n")
	for i, line := range lines {
		if line != "" {
			if _, err := fmt.Fprintf(pw.w, "%s%s", pw.prefix, line); err != nil {
				return 0, err
			}
		}
		if i < len(lines)-1 {
			if _, err := fmt.Fprintln(pw.w); err != nil {
				return 0, err
			}
		}
	}
	return len(p), nil
}

func (suite *TestSuite) startNode(index int) {
	suite.T().Logf("Starting node%d", index)
	node, ok := suite.nodes[index]
	suite.Require().True(ok)
	suite.Require().NotNil(node)

	// save config to a file
	configPath := filepath.Join(suite.currentDir, node.rootDir, "dms_config.json")
	suite.T().Logf("writing config to %s", configPath)
	suite.T().Logf("config: %+v", node.config)
	jsonData, err := json.MarshalIndent(node.config, "", "  ")
	suite.Require().NoError(err)
	err = os.WriteFile(configPath, jsonData, 0o644)
	suite.Require().NoError(err)

	binaryPath := filepath.Join(suite.currentDir, binaryName)
	cmd := exec.Command(binaryPath, "run", "--config", filepath.Join(suite.currentDir, node.rootDir, "dms_config.json"), "--context", node.dmsContext)
	cmd.Env = append(os.Environ(), fmt.Sprintf("GOLOG_LOG_LEVEL=%s", "debug"), fmt.Sprintf("DMS_PASSPHRASE=%s", node.password))
	prefix := fmt.Sprintf("[node-%d] ", index)
	cmd.Stdout = &prefixWriter{prefix: prefix, w: os.Stdout}
	cmd.Stderr = &prefixWriter{prefix: prefix, w: os.Stderr}

	// Start the node process.
	err = cmd.Start()
	suite.Require().NoError(err)

	// Write the PID file.
	err = os.WriteFile(filepath.Join(suite.currentDir, node.rootDir, "proc.pid"), []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0o700)
	suite.Require().NoError(err)

	// Start a goroutine to wait for shutdown.
	go func() {
		<-node.shutdownCh
		_ = cmd.Process.Kill()
	}()

	suite.T().Logf("started node %d with pid %d", index, cmd.Process.Pid)

	err = cmd.Wait()
	suite.T().Logf("node %d exited with error: %v", index, err)
}

// setupGlusterfsServer creates a glusterfs server env
func (suite *TestSuite) setupGlusterfsServer() {
	createDirectories()
	err := pullGlusterImage()
	suite.Require().NoError(err)
	err = runGlusterContainer()
	suite.Require().NoError(err)
	err = runGlusterCommands()
	suite.Require().NoError(err)
}

// setupTestNetwork creates a network of nodes and grants mutual access to all nodes.
func (suite *TestSuite) setupTestNetwork() {
	var (
		restPortIndex = 8090
		p2pPortIndex  = 10689
	)

	for i := 0; i < NumNodes; i++ {
		rootDir := fmt.Sprintf("testdata/dms%d", i)
		password := fmt.Sprintf("password%d", i)
		nodeConfig := createConfig(rootDir, uint32(restPortIndex), fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", p2pPortIndex), []string{})
		nodeIndex := i
		suite.nodes[nodeIndex] = newMockNode(suite.T(), nodeConfig, password, rootDir, nodeIndex)

		restPortIndex++
		p2pPortIndex++
	}

	suite.T().Logf("setting up caps")

	// grant mutual access to all nodes in the network.
	// TODO: lower the time complexity.
	for i, node := range suite.nodes {
		// set the root anchor
		node.client.addRootAnchor(suite.T(), node.dmsContext, node.userDID, node.password)

		// grant access to all other nodes
		for j, otherNode := range suite.nodes {
			if i == j {
				continue
			}
			nodeGrantToken := node.client.grant(suite.T(), node.dmsContext, otherNode.userDID, node.password)
			node.client.anchor(suite.T(), nodeGrantToken, node.dmsContext, "require", node.password)
			otherNode.client.anchor(suite.T(), nodeGrantToken, otherNode.userContext, "provide", otherNode.password)

			otherNodeGrantToken := otherNode.client.grant(suite.T(), otherNode.dmsContext, node.userDID, otherNode.password)
			otherNode.client.anchor(suite.T(), otherNodeGrantToken, otherNode.dmsContext, "require", otherNode.password)
			node.client.anchor(suite.T(), otherNodeGrantToken, node.userContext, "provide", node.password)

			if suite.grantTokens[node.index] == nil {
				suite.grantTokens[node.index] = make(map[int]string)
			}
			if suite.grantTokens[otherNode.index] == nil {
				suite.grantTokens[otherNode.index] = make(map[int]string)
			}
			suite.grantTokens[node.index][otherNode.index] = nodeGrantToken
			suite.grantTokens[otherNode.index][node.index] = otherNodeGrantToken
		}

		// delegate to dms
		delegateToken := node.client.delegate(suite.T(), node.userContext, node.dmsDID, node.password)
		node.client.anchor(suite.T(), delegateToken, node.dmsContext, "provide", node.password)
	}

	suite.T().Logf("creating network")
	for i := 0; i < len(suite.nodes); i++ {
		node := suite.nodes[i]
		node.config.BootstrapPeers = suite.bootstrapPeers

		go suite.startNode(i)
		// wait for the node to be ready
		var (
			networkStats types.NetworkStats
			err          error
		)
		suite.Require().Eventually(func() bool {
			networkStats, err = node.client.self(suite.T(), node.dmsContext, node.password)
			return err == nil && networkStats.ID != ""
		}, 15*time.Second, 3*time.Second, "Expected node %s to be ready", node.index)

		node.peerID = networkStats.ID
		suite.T().Logf("node %d peerID: %s", node.index, node.peerID)

		// We add all the nodes except the last one to the bootstrap peers to ensure that the network is connected.
		if i != NumNodes-1 {
			bootstrapAddr := make([]string, 0)
			addrs := strings.Split(networkStats.ListenAddr, ", ")
			for _, a := range addrs {
				bootstrapAddr = append(bootstrapAddr, fmt.Sprintf("%s/p2p/%s", a, networkStats.ID))
			}

			suite.bootstrapPeers = append(suite.bootstrapPeers, bootstrapAddr...)
		}
	}

	suite.T().Logf("waiting for the network to be ready")
	suite.Require().Eventually(func() bool {
		expectedPeers := make(map[string]struct{})
		for _, node := range suite.nodes {
			expectedPeers[node.peerID] = struct{}{}
		}

		suite.T().Logf("expected peers: %v", expectedPeers)

		for _, node := range suite.nodes {
			resp, err := node.client.peers(suite.T(), node.dmsContext, node.password)
			if err != nil {
				suite.T().Logf("Error fetching peers for node %d: %v", node.index, err)
				return false
			}

			suite.T().Logf("peers for node %d: %v", node.index, resp.Peers)

			// ensure that it has all peers by checking their peerID
			peerCount := 0
			for _, peer := range resp.Peers {
				if _, ok := expectedPeers[peer.String()]; ok {
					peerCount++
				}
			}

			if peerCount != NumNodes {
				suite.T().Logf("Node %d has not found all peers yet", node.index)
				return false
			}
		}
		return true
	}, 120*time.Second, 2*time.Second, fmt.Sprintf("Expected all %d nodes to have %d peers within timeout", NumNodes, NumNodes-1))

	suite.T().Logf("network is ready. Connecting peers ...")

	// connect all the nodes in the network
	time.Sleep(5 * time.Second)
	for i, node := range suite.nodes {
		for j := i + 1; j < len(suite.nodes); j++ {
			suite.T().Logf("connecting node %d to node %d", i, j)
			otherNode := suite.nodes[j]

			otherHostID, err := otherNode.client.self(suite.T(), otherNode.dmsContext, otherNode.password)
			suite.Require().NoError(err)

			result := node.client.connect(suite.T(), node.userContext, node.password, otherHostID.ID)
			suite.Contains(result, `"Status": "CONNECTED"`)
		}
	}

	suite.T().Logf("all nodes are connected")
}

// SetupSuite runs once before the suite starts.
func (suite *TestSuite) SetupSuite() {
	suite.grantTokens = make(map[int]map[int]string)
	suite.nodes = make(map[int]*mockNode)
	suite.currentDir = getCurrentFileDirectory()
	suite.testDataDir = filepath.Join(suite.currentDir, "testdata")
	suite.Require().NotEmpty(suite.currentDir)
}

// TearDownSuite runs once after all tests are complete.
func (suite *TestSuite) TearDownSuite() {
	suite.T().Log("stopping nodes")
	for _, node := range suite.nodes {
		data, err := os.ReadFile(filepath.Join(suite.currentDir, node.rootDir, "proc.pid"))
		if err != nil {
			suite.T().Logf("failed to read pid file: %v", err)
			continue
		}

		pid, err := strconv.Atoi(string(data))
		if err != nil {
			suite.T().Logf("failed to convert pid to int: %v", err)
			continue
		}

		// Get process handle
		proc := getProc(int32(pid))
		if proc == nil {
			suite.T().Logf("process %d not found", pid)
			continue
		}

		// Send kill signal
		node.shutdownCh <- struct{}{}

		// Wait for process to actually terminate with timeout
		suite.Require().Eventually(func() bool {
			if exists, _ := proc.IsRunning(); !exists {
				return true
			}
			return false
		}, 10*time.Second, 100*time.Millisecond, fmt.Sprintf("process %d not terminated", pid))
	}

	suite.T().Logf("cleaning up directories")
	for _, node := range suite.nodes {
		err := os.RemoveAll(filepath.Join(suite.currentDir, node.rootDir))
		if err != nil {
			suite.T().Logf("failed to remove directory %s: %v", node.rootDir, err)
		}
	}

	_ = deleteGlusterContainer()

	suite.T().Logf("teardown complete")
}

// TestBasic performs basic tests to ensure the setup is correct.
func (suite *TestSuite) BasicTests() {
	// every node in the network must be able to broadcast
	for _, node := range suite.nodes {
		result := node.client.broadcast(suite.T(), node.userContext, node.password)
		suite.Equal(NumNodes, countDIDOccurrences(result))
	}

	// every node in the network must be able to onboard
	for _, node := range suite.nodes {
		node.client.onboard(suite.T(), node.userContext, node.password)
	}
}

// DeploymentTest runs the deployment tests.
func (suite *TestSuite) DeploymentTest() {
	suite.Run("must be able to deploy nginx demo with storage", func() {
		// deploy nginx.yaml to node1 using node2's orchestrator
		node1 := suite.nodes[0]
		hostname, err := os.Hostname()
		suite.Require().NoError(err)
		srcFile := filepath.Join(suite.testDataDir, "ensembles", "nginx-storage.yaml")
		destinationFile := filepath.Join(node1.config.WorkDir, "nginx-storage.yaml")
		err = copyFile(srcFile, destinationFile)
		suite.Require().NoError(err)
		err = replaceHostnameInFile(destinationFile, hostname)
		suite.Require().NoError(err)

		freeResourcesBefore, err := node1.client.freeResources(suite.T(), node1.dmsContext, node1.password)
		suite.Require().NoError(err)

		// Deploy nginx.yaml from node2's orchestrator.
		node2 := suite.nodes[1]
		deploymentResult := node2.client.deploy(suite.T(), node2.userContext, node2.password, filepath.Join(node1.config.WorkDir, "nginx-storage.yaml"))
		suite.Contains(deploymentResult, `"Status": "OK"`)
		manifestID := extractEnsembleID(deploymentResult)

		// Wait until the deployment status is "Running".
		suite.Require().Eventually(func() bool {
			status := node2.client.deploymentStatus(suite.T(), node2.userContext, node2.password, manifestID)
			suite.T().Log("Deployment status:", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusString(jobtypes.DeploymentStatusRunning)
		}, 60*time.Second, 5*time.Second, "Deployment did not reach Running status")
		time.Sleep(2 * time.Second)

		// Ensure resources are allocated.
		freeResourcesDuring, err := node1.client.freeResources(suite.T(), node1.dmsContext, node1.password)
		suite.Require().NoError(err)
		suite.False(freeResourcesDuring.Equal(freeResourcesBefore))

		// TODO: check port mapping

		// Shutdown the nginx deployment.
		shutdownRes := node2.client.shutdownDeployment(suite.T(), node2.userContext, node2.password, manifestID)
		suite.Contains(shutdownRes, `"Error": ""`)

		// wait for the ensemble to be shutdown
		suite.Require().Eventually(func() bool {
			status := node2.client.deploymentStatus(suite.T(), node2.dmsContext, node2.password, manifestID)
			suite.T().Log("deployment status:", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusString(jobtypes.DeploymentStatusCompleted)
		}, 60*time.Second, 5*time.Second, "deployment did not reach Completed status")

		// Ensure resources are freed.
		freeResourcesAfter, err := node1.client.freeResources(suite.T(), node1.dmsContext, node1.password)
		suite.Require().NoError(err)
		suite.True(freeResourcesAfter.Equal(freeResourcesBefore))
	})

	suite.Run("must be able to deploy nginx demo", func() {
		// deploy nginx.yaml to node1 using node2's orchestrator
		node1 := suite.nodes[0]
		hostname, err := os.Hostname()
		suite.Require().NoError(err)
		srcFile := filepath.Join(suite.testDataDir, "ensembles", "nginx.yaml")
		destinationFile := filepath.Join(node1.config.WorkDir, "nginx.yaml")
		err = copyFile(srcFile, destinationFile)
		suite.Require().NoError(err)
		err = replaceHostnameInFile(destinationFile, hostname)
		suite.Require().NoError(err)

		freeResourcesBefore, err := node1.client.freeResources(suite.T(), node1.dmsContext, node1.password)
		suite.Require().NoError(err)

		// Deploy nginx.yaml from node2's orchestrator.
		node2 := suite.nodes[1]
		deploymentResult := node2.client.deploy(suite.T(), node2.userContext, node2.password, filepath.Join(node1.config.WorkDir, "nginx.yaml"))
		suite.Contains(deploymentResult, `"Status": "OK"`)
		manifestID := extractEnsembleID(deploymentResult)

		// Wait until the deployment status is "Running".
		suite.Require().Eventually(func() bool {
			status := node2.client.deploymentStatus(suite.T(), node2.userContext, node2.password, manifestID)
			suite.T().Log("Deployment status:", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusString(jobtypes.DeploymentStatusRunning)
		}, 60*time.Second, 5*time.Second, "Deployment did not reach Running status")
		time.Sleep(2 * time.Second)

		// Ensure resources are allocated.
		freeResourcesDuring, err := node1.client.freeResources(suite.T(), node1.dmsContext, node1.password)
		suite.Require().NoError(err)
		suite.False(freeResourcesDuring.Equal(freeResourcesBefore))

		// TODO: check port mapping

		// Shutdown the nginx deployment.
		shutdownRes := node2.client.shutdownDeployment(suite.T(), node2.userContext, node2.password, manifestID)
		suite.Contains(shutdownRes, `"Error": ""`)

		// wait for the ensemble to be shutdown
		suite.Require().Eventually(func() bool {
			status := node2.client.deploymentStatus(suite.T(), node2.dmsContext, node2.password, manifestID)
			suite.T().Log("deployment status:", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusString(jobtypes.DeploymentStatusCompleted)
		}, 60*time.Second, 5*time.Second, "deployment did not reach Completed status")

		// Ensure resources are freed.
		freeResourcesAfter, err := node1.client.freeResources(suite.T(), node1.dmsContext, node1.password)
		suite.Require().NoError(err)
		suite.True(freeResourcesAfter.Equal(freeResourcesBefore))
	})

	suite.Run("must be able to deploy docker hello-world", func() {
		node2 := suite.nodes[1]
		deployment2Result := node2.client.deploy(suite.T(), node2.userContext, node2.password, filepath.Join(suite.testDataDir, "ensembles", "hello.yaml"))
		suite.Contains(deployment2Result, `"Status": "OK"`)
		manifestID := extractEnsembleID(deployment2Result)
		suite.Require().Eventually(func() bool {
			status := node2.client.deploymentStatus(suite.T(), node2.userContext, node2.password, manifestID)
			suite.T().Log("Second deployment status:", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusString(jobtypes.DeploymentStatusRunning)
		}, 60*time.Second, 5*time.Second, "Hello-world deployment did not reach Running status")

		// Shutdown the hello-world deployment.
		shutdownRes := node2.client.shutdownDeployment(suite.T(), node2.dmsContext, node2.password, manifestID)
		suite.Require().Contains(shutdownRes, `"Error": ""`)

		// wait for the ensemble to be shutdown
		suite.Require().Eventually(func() bool {
			status := node2.client.deploymentStatus(suite.T(), node2.dmsContext, node2.password, manifestID)
			suite.T().Log("deployment status:", extractStatus(status))
			return extractStatus(status) == jobtypes.DeploymentStatusString(jobtypes.DeploymentStatusCompleted)
		}, 60*time.Second, 5*time.Second, "deployment did not reach Completed status")
	})
}

// TestRevokeToken tests the token revocation.
func (suite *TestSuite) RevokeTokenTests() {
	node1 := suite.nodes[0]

	// revoke all the tokens granted from node1
	for _, grantToken := range suite.grantTokens[node1.index] {
		revokeToken := node1.client.revokeToken(suite.T(), node1.dmsContext, node1.password, grantToken)
		node1.client.anchorBehaviour(suite.T(), node1.dmsContext, node1.password, revokeToken)
	}

	suite.Require().Eventually(func() bool {
		// try saying hello from node2 to node1
		node2 := suite.nodes[1]
		_, err := node2.client.hello(suite.T(), node2.userContext, node2.password, node1.dmsDID)
		return err != nil
	}, 120*time.Second, 10*time.Second, "Expected request to fail after revoking all tokens")
}

// Test_RunSuite runs the test suite.
//
// Note: While adding new test cases, we need to ensure we use the suite's Require() method instead of using the direct package level functions.
// This is because the suite's Require() method will ensure that the test case is marked as failed and the suite's TearDownSuite() method is called.
// If we use the package level functions, the test case will be marked as PASS but the tests will ultimately FAIL since the suite can't track it.
func (suite *TestSuite) Test_RunSuite() {
	gin.SetMode(gin.DebugMode)
	os.Setenv("GOLOG_LOG_LEVEL", "debug")

	suite.setupTestNetwork()

	suite.setupGlusterfsServer()

	suite.Run("must pass basic tests", func() {
		suite.BasicTests()
	})

	suite.Run("must be able to deploy ensembles", func() {
		suite.DeploymentTest()
	})
	suite.Run("must be able to revoke tokens", func() {
		suite.T().Skip("Skipping token revocation test since it is failing")
		suite.RevokeTokenTests()
	})

	suite.Run("dms creates a volume on storage node", func() {
		suite.T().Skip("Skipping unimplemeted test")

		// glusterfs container is running in host mode
		// we can directly use the bootstrap nodes here
		// peers := strings.Join(suite.bootstrapPeers, ",")
		// envVars := []string{
		//	"DMS_PASSPHRASE=password3",
		//	"GOLOG_LOG_LEVEL=debug",
		//	"BOOTSTRAP_PEERS=" + peers,
		// }
		// err := runBinaryInContainer(glusterContainerName, "/home/dms", []string{"run", "--data-dir", "/home/data"}, envVars, "/home/output.log")
		// require.NoError(t, err)
		// firstNodeClient := suite.nodes[0]
		// suite.T().Logf("createVolume: glusterDMSDID: %s", suite.glusterDMSDID)
		// time.Sleep(20 * time.Second)
		// _, err = firstNodeClient.client.createVolume(t, firstNodeClient.userContext, firstNodeClient.password, suite.glusterDMSDID)
		// require.NoError(t, err)
	})
}

// TestIntegration is the entry point for the integration tests.
func TestIntegration(t *testing.T) {
	suite.Run(t, dmsTestSuite)
}
