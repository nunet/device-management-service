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
	homeDir       string
}

type prefixWriter struct {
	prefix string
	w      io.Writer
}

func (pw *prefixWriter) Write(p []byte) (n int, err error) {
	lines := strings.Split(string(p), "\n")

	// different colors for each node prefix
	colorIdx, _ := strconv.Atoi(pw.prefix[len(pw.prefix)-3 : len(pw.prefix)-2])
	colorIdx++
	color := fmt.Sprintf("\x1b[3%dm", colorIdx)
	colorReset := "\x1b[0m"

	for i, line := range lines {
		if line != "" {
			if _, err := fmt.Fprintf(pw.w, "%s%s%s%s", color, pw.prefix, colorReset, line); err != nil {
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

func (s *TestSuite) startNode(index int) {
	s.T().Logf("Starting node%d", index)
	node, ok := s.nodes[index]
	s.Require().True(ok)
	s.Require().NotNil(node)

	// save config to a file
	configPath := filepath.Join(node.config.General.UserDir, "dms_config.json")
	_ = os.MkdirAll(filepath.Dir(configPath), 0o755)
	s.T().Logf("writing config to %s", configPath)
	s.T().Logf("config: %+v", node.config)
	jsonData, err := json.MarshalIndent(node.config, "", "  ")
	s.Require().NoError(err)
	err = os.WriteFile(configPath, jsonData, 0o644)
	s.Require().NoError(err)

	binaryPath := filepath.Join(s.currentDir, binaryName)
	cmd := exec.Command(binaryPath, "run", "--config", configPath, "--context", node.dmsContext)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GOLOG_LOG_LEVEL=%s,%s",
			"debug",
			"observability=info", // too verbose on debug level
		),
		fmt.Sprintf("DMS_PASSPHRASE=%s", node.password),
	)
	prefix := fmt.Sprintf("[%s-node-%d] ", s.Name, index)
	cmd.Stdout = &prefixWriter{prefix: prefix, w: os.Stdout}
	cmd.Stderr = &prefixWriter{prefix: prefix, w: os.Stderr}

	// Start the node process.
	err = cmd.Start()
	s.Require().NoError(err)

	// Write the PID file.
	err = os.WriteFile(filepath.Join(node.config.General.UserDir, "proc.pid"), []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0o700)
	s.Require().NoError(err)

	// Start a goroutine to wait for shutdown.
	go func() {
		<-node.shutdownCh
		_ = cmd.Process.Kill()
	}()

	s.T().Logf("started node %d with pid %d", index, cmd.Process.Pid)

	err = cmd.Wait()
	s.T().Logf("node %d exited with error: %v", index, err)
}

// setupTestNetwork creates a network of nodes and grants mutual access to all nodes.
func (s *TestSuite) setupTestNetwork() {
	s.T().Logf("%s: setting up %d nodes", s.Name, s.numNodes)
	// Start from a clean per‑suite sandbox, **never** the real $HOME
	_ = os.RemoveAll(filepath.Join(s.homeDir, ".nunet"))
	for i := 0; i < s.numNodes; i++ {
		rootDir := fmt.Sprintf("testdata/%s/dms%d", s.Name, i)
		password := fmt.Sprintf("password%d", i)
		userDir := filepath.Join(s.homeDir, rootDir)
		_ = os.RemoveAll(userDir)
		nodeConfig := createConfig(
			userDir,
			uint32(s.restPortIndex),
			[]string{fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", s.p2pPortIndex), fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", s.p2pPortIndex)},
			[]string{},
		)
		nodeIndex := i

		var err error
		s.nodes[nodeIndex], err = newMockNode(s.T(), nodeConfig, password, rootDir, nodeIndex)
		s.Require().NoError(err)

		s.restPortIndex++
		s.p2pPortIndex++
	}

	s.T().Logf("setting up caps")

	// grant mutual access to all nodes in the network.
	// TODO: lower the time complexity.
	for i, node := range s.nodes {
		// set the root anchor
		node.client.addRootAnchor(s.T(), node.dmsContext, node.userDID, node.password)

		// grant access to all other nodes
		for j, otherNode := range s.nodes {
			if i == j {
				continue
			}

			nodeGrantToken := node.client.grant(s.T(), node.dmsContext, otherNode.userDID, node.password)
			node.client.anchor(s.T(), nodeGrantToken, node.dmsContext, "require", node.password)
			otherNode.client.anchor(s.T(), nodeGrantToken, otherNode.userContext, "provide", otherNode.password)

			otherNodeGrantToken := otherNode.client.grant(s.T(), otherNode.dmsContext, node.userDID, otherNode.password)
			otherNode.client.anchor(s.T(), otherNodeGrantToken, otherNode.dmsContext, "require", otherNode.password)
			node.client.anchor(s.T(), otherNodeGrantToken, node.userContext, "provide", node.password)

			if s.grantTokens[node.index] == nil {
				s.grantTokens[node.index] = make(map[int]string)
			}
			if s.grantTokens[otherNode.index] == nil {
				s.grantTokens[otherNode.index] = make(map[int]string)
			}
			s.grantTokens[node.index][otherNode.index] = nodeGrantToken
			s.grantTokens[otherNode.index][node.index] = otherNodeGrantToken
		}

		// delegate to dms
		delegateToken := node.client.delegate(s.T(), node.userContext, node.dmsDID, node.password)
		node.client.anchor(s.T(), delegateToken, node.dmsContext, "provide", node.password)
	}

	s.T().Logf("creating network")
	for i := 0; i < len(s.nodes); i++ {
		node := s.nodes[i]
		node.config.BootstrapPeers = s.bootstrapPeers

		go s.startNode(i)
		// wait for the node to be ready
		var (
			networkStats types.NetworkStats
			err          error
		)
		s.Require().Eventually(func() bool {
			networkStats, err = node.client.self(s.T(), node.dmsContext, node.password)
			return err == nil && networkStats.ID != ""
		}, 15*time.Second, 3*time.Second, "Expected node %s to be ready", node.index)

		node.peerID = networkStats.ID
		s.T().Logf("node %d peerID: %s", node.index, node.peerID)

		// We add all the nodes except the last one to the bootstrap peers to ensure that the network is connected.
		if i != s.numNodes-1 {
			bootstrapAddr := make([]string, 0)
			addrs := strings.Split(networkStats.ListenAddr, ", ")
			for _, a := range addrs {
				bootstrapAddr = append(bootstrapAddr, fmt.Sprintf("%s/p2p/%s", a, networkStats.ID))
			}

			s.bootstrapPeers = append(s.bootstrapPeers, bootstrapAddr...)
		}
	}

	s.T().Logf("waiting for the network to be ready")
	s.Require().Eventually(func() bool {
		expectedPeers := make(map[string]struct{})
		for _, node := range s.nodes {
			expectedPeers[node.peerID] = struct{}{}
		}

		s.T().Logf("expected peers: %v", expectedPeers)

		for _, node := range s.nodes {
			resp, err := node.client.peers(s.T(), node.dmsContext, node.password)
			if err != nil {
				s.T().Logf("Error fetching peers for node %d: %v", node.index, err)
				return false
			}

			s.T().Logf("peers for node %d: %v", node.index, resp.Peers)

			// ensure that it has all peers by checking their peerID
			peerCount := 0
			for _, peer := range resp.Peers {
				if _, ok := expectedPeers[peer.String()]; ok {
					peerCount++
				}
			}

			if peerCount != s.numNodes {
				s.T().Logf("Node %d has not found all peers yet", node.index)
				return false
			}
		}
		return true
	}, 120*time.Second, 2*time.Second, fmt.Sprintf("Expected all %d nodes to have %d peers within timeout", s.numNodes, s.numNodes-1))

	s.T().Logf("network is ready. Onboarding ...")
	for _, node := range s.nodes {
		node.client.onboard(s.T(), node.userContext, node.password)
	}

	s.T().Logf("connecting nodes")
	// connect all the nodes in the network
	time.Sleep(5 * time.Second)
	for i, node := range s.nodes {
		for j := i + 1; j < len(s.nodes); j++ {
			s.T().Logf("connecting node %d to node %d", i, j)
			otherNode := s.nodes[j]

			otherHostID, err := otherNode.client.self(s.T(), otherNode.dmsContext, otherNode.password)
			s.Require().NoError(err)

			result := node.client.connect(s.T(), node.userContext, node.password, otherHostID.ID)
			s.Contains(result, `"Status": "CONNECTED"`)
		}
	}

	s.T().Logf("all nodes are onboarded and connected")
}

// SetupSuite runs once before the suite starts.
// Keep testdata dir to find the test artifact after execution
func (s *TestSuite) SetupSuite() {
	s.grantTokens = make(map[int]map[int]string)
	s.nodes = make(map[int]*mockNode)
	s.bootstrapPeers = []string{}
	s.currentDir = getCurrentFileDirectory()
	s.testDataDir = filepath.Join(s.currentDir, "testdata")
}

// TearDownSuite runs once after all tests are complete.
func (s *TestSuite) TearDownSuite() {
	s.T().Log("stopping nodes")
	for _, node := range s.nodes {
		data, err := os.ReadFile(filepath.Join(node.config.General.UserDir, "proc.pid"))
		if err != nil {
			s.T().Logf("failed to read pid file: %v", err)
			continue
		}

		pid, err := strconv.Atoi(string(data))
		if err != nil {
			s.T().Logf("failed to convert pid to int: %v", err)
			continue
		}

		// Get process handle
		proc := getProc(int32(pid))
		if proc == nil {
			s.T().Logf("process %d not found", pid)
			continue
		}

		// Send kill signal
		node.shutdownCh <- struct{}{}

		// Wait for process to actually terminate with timeout
		s.Require().Eventually(func() bool {
			if exists, _ := proc.IsRunning(); !exists {
				return true
			}
			return false
		}, 10*time.Second, 100*time.Millisecond, fmt.Sprintf("process %d not terminated", pid))
	}

	s.T().Logf("cleaning up directories")
	for _, node := range s.nodes {
		err := os.RemoveAll(filepath.Join(s.currentDir, node.rootDir))
		if err != nil {
			s.T().Logf("failed to remove directory %s: %v", node.rootDir, err)
		}
	}

	s.T().Logf("teardown complete")
}

// TestRevokeToken tests the token revocation.
func (s *TestSuite) RevokeTokenTests() {
	node1 := s.nodes[0]

	// revoke all the tokens granted from node1
	for _, grantToken := range s.grantTokens[node1.index] {
		revokeToken := node1.client.revokeToken(s.T(), node1.dmsContext, node1.password, grantToken)
		node1.client.anchorBehaviour(s.T(), node1.dmsContext, node1.password, revokeToken)
	}

	s.Require().Eventually(func() bool {
		// try saying hello from node2 to node1
		node2 := s.nodes[1]
		_, err := node2.client.hello(s.T(), node2.userContext, node2.password, node1.dmsDID)
		return err != nil
	}, 120*time.Second, 10*time.Second, "Expected request to fail after revoking all tokens")
}

// Test_RunSuite runs the test suite.
//
// Note: While adding new test cases, we need to ensure we use the suite's Require() method instead of using the direct package level functions.
// This is because the suite's Require() method will ensure that the test case is marked as failed and the suite's TearDownSuite() method is called.
// If we use the package level functions, the test case will be marked as PASS but the tests will ultimately FAIL since the suite can't track it.
func (s *TestSuite) Test_RunSuite() {
	gin.SetMode(gin.DebugMode)
	os.Setenv("GOLOG_LOG_LEVEL", "debug,observability=info")
	s.setupTestNetwork()
	s.runner(s)
}
