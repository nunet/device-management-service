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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
	"gitlab.com/nunet/device-management-service/types"
)

const binaryName = "dms"

// TestSuite defines our end-to-end test suite.
type TestSuite struct {
	suite.Suite
	runner func(*TestSuite)

	Name           string
	numNodes       int
	currentDir     string
	testDataDir    string
	bootstrapPeers []string
	nodes          map[int]*mockNode
	grantTokens    map[int]map[int]string // map[nodeIndex]map[otherNodeIndex]grantToken

	restPortIndex int
	p2pPortIndex  int
}

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
	prefix := fmt.Sprintf("[%s-node-%d] ", suite.Name, index)
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

// setupTestNetwork creates a network of nodes and grants mutual access to all nodes.
func (suite *TestSuite) setupTestNetwork() {
	suite.T().Logf("%s: setting up %d nodes", suite.Name, suite.numNodes)
	for i := 0; i < suite.numNodes; i++ {
		rootDir := fmt.Sprintf("testdata/%s/dms%d", suite.Name, i)
		password := fmt.Sprintf("password%d", i)
		nodeConfig := createConfig(rootDir, uint32(suite.restPortIndex), fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", suite.p2pPortIndex), []string{})
		nodeIndex := i
		suite.nodes[nodeIndex] = newMockNode(suite.T(), nodeConfig, password, rootDir, nodeIndex)

		suite.restPortIndex++
		suite.p2pPortIndex++
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
		if i != suite.numNodes-1 {
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

			if peerCount != suite.numNodes {
				suite.T().Logf("Node %d has not found all peers yet", node.index)
				return false
			}
		}
		return true
	}, 120*time.Second, 2*time.Second, fmt.Sprintf("Expected all %d nodes to have %d peers within timeout", suite.numNodes, suite.numNodes-1))

	suite.T().Logf("network is ready. Onboarding ...")
	for _, node := range suite.nodes {
		node.client.onboard(suite.T(), node.userContext, node.password)
	}

	suite.T().Logf("connecting nodes")
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

	suite.T().Logf("all nodes are onboarded and connected")
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

	suite.T().Logf("teardown complete")
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
	suite.runner(suite)
}
