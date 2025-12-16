// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
	"go.elastic.co/apm/module/apmhttp/v2"
	"go.elastic.co/apm/transport"
	"go.elastic.co/apm/v2"

	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/types"
)

const binaryName = "dms"

// envE2ECacheKeep set to "DMS_E2E_CACHE_KEEP=1" will preserve the generated cache for post-run inspections.
const envE2ECacheKeep = "DMS_E2E_CACHE_KEEP"

// envE2ECacheKeys set to "DMS_E2E_CACHE_KEYS=1" will preserve generated keys to speed up test runs.
// Implies envE2ECacheKeep.
const envE2ECacheKeys = "DMS_E2E_CACHE_KEYS"

// envE2EDebugNodes set to "DMS_E2E_DEBUG_NODES=1,3" will start nodes with indexes 1 and 3 in debug mode (delve).
const envE2EDebugNodes = "DMS_E2E_DEBUG_NODES"

// envE2EObserveToken set to "DMS_E2E_OBSERVE_TOKEN=supersecrettoken" will enable observability for all nodes' logs
// and traces. Requires envE2EObserveAPIKey.
const envE2EObserveToken = "DMS_E2E_OBSERVE_TOKEN"

// envE2EObserveAPIKey set to "DMS_E2E_OBSERVE_API_KEY=someapikey" will enable observability for all nodes' logs. Requires envE2EObserveToken.
const envE2EObserveAPIKey = "DMS_E2E_OBSERVE_API_KEY"

// envE2EObservePrefix used to distinguish between test runs, eg `me` will result in `prefix/E2E/deployment_tests`.
const envE2EObservePrefix = "DMS_E2E_OBSERVE_PREFIX"

// SUMMARY

type SummaryNode struct {
	Onboarded bool
	Error     bool

	// TODO node summaries

	// TODO implement
	Connected bool
	// Role is this node's role, eg orchestrator TODO implement
	Role string
	// TODO implement
	Bids []any
	// errs is a list of errors for this node TODO implement
	Errs []string
	// TODO Onboarded RAM, disk, GPU
}

type Summary struct {
	// Nodes is a map of test nodes and their state.
	Nodes map[string]*SummaryNode
	// NodeConns is a map of active connections between nodes, sourced from nodes' logs.
	NodeConns map[string][]string
	// NodeIDs is a list of node indexes to peer IDs.
	NodeIDs []string
	// NodeDIDs is a list of node indexes to DIDs [userDID, dmsDID].
	NodeDIDs [][]string
	// NodeTestConns is like NodeConns, but only for test nodes. Sourced from the test runner.
	NodeTestConns map[string][]string
	Test          struct {
		NodesReady       bool
		CapsReady        bool
		NetworkCreated   bool
		NetworkReady     bool
		NetworkConnected bool
	}

	// errs is a list of errors in the test runner TODO implement
	// errs []string
}

func (s *Summary) String() string {
	f := fmt.Sprintf

	// E2E runner info
	conns := 0
	for _, targets := range s.NodeTestConns {
		conns += len(targets)
	}
	ret := f("\nSUMMARY [nodes: %d, conns: %d]\n", len(s.Nodes), conns)
	if s.Test.NodesReady {
		ret += f("(nodes ready) ")
	}
	if s.Test.CapsReady {
		ret += f("(caps ready) ")
	}
	if s.Test.NetworkCreated {
		ret += f("(network created) ")
	}
	if s.Test.NetworkReady {
		ret += f("(network ready) ")
	}
	if s.Test.NetworkConnected {
		ret += f("(network connected) ")
	}

	if len(s.NodeIDs) == 0 {
		return ret
	}
	ret += "\n"

	// nodes
	for idx, peerID := range s.NodeIDs {
		ret += f("\n  %d: %s", idx, peerID)

		// log scraped states
		tags := []string{}
		if s.Nodes[peerID].Onboarded {
			tags = append(tags, "onboarded")
		}
		if s.Nodes[peerID].Connected {
			tags = append(tags, "connected")
		}
		if s.Nodes[peerID].Error {
			tags = append(tags, "error")
		}
		if len(tags) > 0 {
			ret += "\n  - #" + strings.Join(tags, "#")
		}

		// TODO remaining node info from SummaryNode

		// DIDs
		ret += f("\n  - userDID: %s", s.NodeDIDs[idx][0])
		ret += f("\n  - dmsDID: %s", s.NodeDIDs[idx][1])
	}

	return ret
}

//nolint:revive,unparam
func (s *Summary) parseLog(nodeIdx int, isErr bool, text string) {
	// dont panic the whole test case
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered from panic in parseLog for node %d: %v\n", nodeIdx, r)
		}
	}()

	// parse lines
	if strings.Contains(text, `machine_onboarded_successfully`) {
		s.Nodes[s.NodeIDs[nodeIdx]].Onboarded = true
	}
}

// LOGS

type LogInterceptor struct {
	summary *Summary
	nodeIdx int
	isErr   bool
	fwd     io.Writer
}

var _ io.Writer = &LogInterceptor{}

func (l LogInterceptor) Write(p []byte) (n int, err error) {
	l.summary.parseLog(l.nodeIdx, l.isErr, string(p))

	return l.fwd.Write(p)
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

// TEST SUITE

// TestSuite defines a single end-to-end test suite, with a dedicated network.
// TODO rename to TestCase?
type TestSuite struct {
	suite.Suite
	runner func(*TestSuite)

	// Name is the name of this test case.
	Name       string
	numNodes   int
	currentDir string
	// rootDir is the root path for the test case.
	rootDir        string
	bootstrapPeers []string
	nodes          map[int]*mockNode
	grantTokens    map[int]map[int]string // map[nodeIndex]map[otherNodeIndex]grantToken

	restPortIndex int
	p2pPortIndex  int
	summary       *Summary
	// testDataDir is the E2E testdata dir with fixtures.
	testDataDir string
	rootTrace   *apm.Transaction
	tracer      *apm.Tracer
}

func (s *TestSuite) startNode(index int) {
	dbgNodes := strings.Split(os.Getenv(envE2EDebugNodes), ",")
	idxS := strconv.Itoa(index)

	s.T().Logf("Starting node%d", index)
	node, ok := s.nodes[index]
	s.Require().True(ok)
	s.Require().NotNil(node)

	// If the node was previously stopped, create a new shutdown channel
	if node.stopped {
		s.T().Logf("node %d was previously stopped, creating new shutdown channel", index)
		node.shutdownCh = make(chan struct{})
		node.stopped = false

		// previous config
		path := filepath.Join(node.config.General.UserDir, "dms_config.json")
		jsonData, err := os.ReadFile(path)
		s.Require().NoError(err)
		err = json.Unmarshal(jsonData, &node.config)
		s.Require().NoError(err)
		s.T().Logf("previous config: %+v", node.config)
	}

	// save config to a file
	configPath := filepath.Join(node.config.General.UserDir, "dms_config.json")
	_ = os.MkdirAll(filepath.Dir(configPath), 0o755)
	s.T().Logf("writing config to %s", configPath)
	s.T().Logf("config: %+v", node.config)
	jsonData, err := json.MarshalIndent(node.config, "", "  ")
	s.Require().NoError(err)
	err = os.WriteFile(configPath, jsonData, 0o644)
	s.Require().NoError(err)

	// run or dbg the node
	binaryPath := filepath.Join(s.currentDir, binaryName)
	var cmd *exec.Cmd
	if slices.Contains(dbgNodes, idxS) {
		cmd = exec.Command("dlv", "exec", "--headless", "--listen=:234"+idxS, "--api-version=2",
			"--accept-multiclient", binaryPath, "--", "run", "--config", configPath, "--context", node.dmsContext)
	} else {
		cmd = exec.Command(binaryPath, "run", "--config", configPath, "--context", node.dmsContext)
	}
	// config the env
	tracePrefix := os.Getenv(envE2EObservePrefix)
	if tracePrefix != "" {
		tracePrefix += "/"
	}
	// TODO observability.EnvFlightrecSec
	envFlightrec := os.Getenv("DMS_FLIGHTREC_SEC")
	cmd.Env = append(os.Environ(),
		// define a name for Kibana
		"ELASTIC_APM_SERVICE_NODE_NAME="+tracePrefix+"E2E-"+s.T().Name()+"-node-"+idxS,
		"DMS_PASSPHRASE="+node.password,
		// log levels
		"GOLOG_LOG_LEVEL=debug",
		"DMS_OBSERVE_LEVEL=debug",
		"DMS_FLIGHTREC_SEC="+envFlightrec,
		"DMS_BINARY_PATH="+binaryPath,
	)

	// nest under a test span
	if s.rootTrace != nil {
		traceCtx := s.rootTrace.TraceContext()
		cmd.Env = append(cmd.Env,
			"ELASTIC_APM_TRACEPARENT="+apmhttp.FormatTraceparentHeader(traceCtx),
			"ELASTIC_APM_TRACESTATE="+traceCtx.State.String(),
		)
	}

	// intercept log for scraping
	logPrefix := fmt.Sprintf("[%s-node-%d] ", s.Name, index)
	cmd.Stdout = &LogInterceptor{
		summary: s.summary,
		nodeIdx: index,
		fwd:     &prefixWriter{prefix: logPrefix, w: os.Stdout},
	}
	cmd.Stderr = &LogInterceptor{
		summary: s.summary,
		nodeIdx: index,
		isErr:   true,
		fwd:     &prefixWriter{prefix: logPrefix, w: os.Stderr},
	}

	// Start the node process.
	err = cmd.Start()
	s.Require().NoError(err)

	// Write the PID file.
	err = os.WriteFile(filepath.Join(node.config.General.UserDir, "proc.pid"), []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0o700)
	s.Require().NoError(err)

	// Start a goroutine to wait for shutdown.
	go func() {
		<-node.shutdownCh

		// handle flight recorder
		if secsNum, _ := strconv.Atoi(envFlightrec); secsNum > 0 {
			_, err = node.client.debugFlightrec(s.T(), node.dmsContext, node.password)
			if err != nil {
				s.T().Logf("failed to get flight recorder: %v", err)
			}
			time.Sleep(100 * time.Millisecond)
		}

		// Try graceful shutdown first
		if err := cmd.Process.Signal(os.Interrupt); err != nil {
			s.T().Logf("failed to send interrupt signal to node %d: %v", index, err)
			_ = cmd.Process.Kill()
			return
		}

		// Wait for graceful shutdown with timeout
		done := make(chan error)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-done:
			s.T().Logf("node %d shutdown gracefully", index)
			return
		case <-time.After(5 * time.Second):
			s.T().Logf("graceful shutdown timeout for node %d, forcing kill", index)
			_ = cmd.Process.Kill()
		}
	}()

	s.T().Logf("started node %d with pid %d", index, cmd.Process.Pid)

	err = cmd.Wait()
	s.T().Logf("node %d exited with error: %v", index, err)
	if err != nil && !strings.Contains(err.Error(), "signal: killed") {
		if _, ok := s.summary.Nodes[node.peerID]; ok {
			s.summary.Nodes[node.peerID].Error = true
			s.printSummary()
		} else {
			s.T().Logf("summary for node %d missing", index)
		}
	}
}

// stopNode stops a specific node by sending a shutdown signal
func (s *TestSuite) stopNode(index int) {
	node, ok := s.nodes[index]
	s.Require().True(ok)
	s.Require().NotNil(node)

	if node.stopped {
		return
	}

	// Read the PID file
	data, err := os.ReadFile(filepath.Join(node.config.General.UserDir, "proc.pid"))
	if err != nil {
		return
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return
	}

	// Get process handle
	proc := getProc(int32(pid))
	if proc == nil {
		node.stopped = true
		return
	}

	// Send shutdown signal
	node.shutdownCh <- struct{}{}

	// Wait for process to actually terminate with timeout
	s.Require().Eventually(func() bool {
		if exists, _ := proc.IsRunning(); !exists {
			node.stopped = true
			return true
		}
		return false
	}, 10*time.Second, 100*time.Millisecond, fmt.Sprintf("process %d not terminated for node %d", pid, index))
}

// killNode kills a specific node by killing the dms process
func (s *TestSuite) killNode(index int) {
	node, ok := s.nodes[index]
	s.Require().True(ok)
	s.Require().NotNil(node)

	if node.stopped {
		return
	}

	// Read the PID file
	data, err := os.ReadFile(filepath.Join(node.config.General.UserDir, "proc.pid"))
	if err != nil {
		return
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return
	}

	// Get process handle
	proc := getProc(int32(pid))
	if proc == nil {
		node.stopped = true
		return
	}

	err = proc.Kill()
	if err != nil {
		s.T().Logf("failed to kill process %d: %v", pid, err)
	}

	// Wait for process to actually terminate with timeout
	s.Require().Eventually(func() bool {
		if exists, _ := proc.IsRunning(); !exists {
			node.stopped = true
			return true
		}
		return false
	}, 10*time.Second, 100*time.Millisecond, fmt.Sprintf("process %d not terminated for node %d", pid, index))
}

// setupTestNetwork creates a network of nodes and grants mutual access to all nodes.
func (s *TestSuite) setupTestNetwork() {
	cacheKeys := os.Getenv(envE2ECacheKeys) == "1"
	summ := s.summary

	s.T().Logf("%s: setting up %d nodes", s.Name, s.numNodes)
	for i := 0; i < s.numNodes; i++ {
		nodeName := fmt.Sprintf("dms%d", i)
		password := "pass1234"
		// lock the node in this dir
		nodeRoot := filepath.Join(s.rootDir, nodeName)
		cfg := createConfig(
			nodeRoot,
			uint32(s.restPortIndex),
			[]string{fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", s.p2pPortIndex), fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", s.p2pPortIndex)},
			[]string{},
		)

		// remove old files
		if cacheKeys {
			_ = os.RemoveAll(cfg.DataDir)
			_ = os.RemoveAll(cfg.WorkDir)
			_ = os.RemoveAll(filepath.Join(cfg.General.UserDir, node.CapstoreDir))
			_ = os.Remove(filepath.Join(cfg.General.UserDir, "logs.jsonl"))
		} else {
			_ = os.RemoveAll(nodeRoot)
		}
		nodeIndex := i

		fmt.Println("setting up for test", s.Name)

		// for contracts test set the 4th node as a payment validator
		switch s.Name {
		case "deployment_with_contracts_tests":
			if nodeIndex == 3 {
				cfg.PaymentProvider.EthereumRPCURL = "http://localhost:9421/"
				cfg.PaymentProvider.Mode = true
			}
		case "deployment_with_contracts_pay_per_deployment_tests":
			if nodeIndex == 3 {
				cfg.PaymentProvider.EthereumRPCURL = "http://localhost:9422/"
				cfg.PaymentProvider.Mode = true
			}
		case "deployment_with_contracts_pay_per_time_utilization_tests":
			if nodeIndex == 3 {
				cfg.PaymentProvider.EthereumRPCURL = "http://localhost:9423/"
				cfg.PaymentProvider.Mode = true
			}
		case "deployment_with_contracts_pay_per_resource_utilization_tests":
			if nodeIndex == 3 {
				cfg.PaymentProvider.EthereumRPCURL = "http://localhost:9424/"
				cfg.PaymentProvider.Mode = true
			}
		case "deployment_with_contracts_fixed_rental_tests":
			if nodeIndex == 3 {
				cfg.PaymentProvider.EthereumRPCURL = "http://localhost:9425"
				cfg.PaymentProvider.Mode = true
			}
		case "deployment_with_contracts_periodic_tests":
			if nodeIndex == 3 {
				cfg.PaymentProvider.EthereumRPCURL = "http://localhost:9426"
			}
		case "deployment_with_ondemand_provisioner_tests":
			if nodeIndex == 0 {
				cfg.General.ComputeGateway = true
				cfg.General.Providers = []config.ProviderConfig{
					{
						Name:   "local-incus",
						Type:   "local-incus",
						Config: map[string]interface{}{},
					},
				}
			}
		case "deployment_with_contracts_collect_after_pay_tests":
			if nodeIndex == 3 {
				cfg.PaymentProvider.EthereumRPCURL = "http://localhost:9425/"
				cfg.PaymentProvider.Mode = true
			}
		}

		var err error
		s.nodes[nodeIndex], err = newMockNode(s.T(), cfg, password, nodeRoot, nodeIndex)
		s.Require().NoError(err)

		s.restPortIndex++
		s.p2pPortIndex++
	}

	s.T().Logf("setting up caps")
	summ.Test.NodesReady = true
	s.printSummary()

	// grant mutual access to all nodes in the network.
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

	summ.Test.CapsReady = true
	s.printSummary()
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
		}, 30*time.Second, 5*time.Second, "Expected node %s to be ready", node.index)

		node.peerID = networkStats.ID
		s.T().Logf("node %d peerID: %s", node.index, node.peerID)

		summ.Nodes[node.peerID] = &SummaryNode{}
		summ.NodeIDs = append(summ.NodeIDs, node.peerID)
		summ.NodeDIDs = append(summ.NodeDIDs, []string{node.userDID, node.dmsDID})

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

	summ.Test.NetworkCreated = true
	s.printSummary()
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

	summ.Test.NetworkReady = true
	s.printSummary()
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
			summ.NodeTestConns[node.peerID] = append(summ.NodeTestConns[node.peerID], otherHostID.ID)
			summ.NodeTestConns[otherHostID.ID] = append(summ.NodeTestConns[otherHostID.ID], node.peerID)
		}
	}

	summ.Test.NetworkConnected = true
	s.printSummary()
	s.T().Logf("all nodes are onboarded and connected")
}

// SetupSuite runs once before the suite starts.
// Keep testdata dir to find the test artifact after execution
func (s *TestSuite) SetupSuite() {
	s.grantTokens = make(map[int]map[int]string)
	s.nodes = make(map[int]*mockNode)
	s.bootstrapPeers = []string{}
	s.currentDir = getCurrentFileDirectory()
	// TODO should be nested under "testdata/artifacts" to diff from fixtures
	s.testDataDir = filepath.Join(s.currentDir, "testdata")
	s.rootDir = filepath.Join(s.testDataDir, s.Name)

	// Initialize the APM tracer
	cfg := createConfig("/tmp/fake", 0, []string{}, []string{})
	if !cfg.Observability.ElasticsearchEnabled {
		return
	}
	s.T().Logf("initializing APM tracer")

	// Create a new APM transport
	tr, err := transport.NewHTTPTransport()
	if err != nil {
		s.T().Logf("Failed to create APM transport: %v", err)
		return
	}

	// Parse and set the APM Server URL
	serverURL, err := url.Parse(cfg.APM.ServerURL)
	if err != nil {
		s.T().Logf("Failed to parse APM server URL: %v", err)
		return
	}
	tr.SetServerURL(serverURL)

	// Set API key if provided
	if cfg.APM.SecretToken != "" {
		tr.SetSecretToken(cfg.APM.SecretToken)
	} else if cfg.APM.APIKey != "" {
		tr.SetAPIKey(cfg.APM.APIKey)
	}

	tracer, err := apm.NewTracerOptions(apm.TracerOptions{
		ServiceName:    cfg.APM.ServiceName,
		ServiceVersion: "1.0.0",
		Transport:      tr,
	})
	if err != nil {
		s.T().Logf("Failed to initialize APM tracer: %v", err)
		return
	}
	apm.SetDefaultTracer(tracer)
	s.tracer = tracer

	// compose name and the top trace
	prefix := os.Getenv(envE2EObservePrefix)
	name := "E2E/" + s.Name
	if prefix != "" {
		name = prefix + "/" + name
	}
	s.rootTrace = tracer.StartTransaction(name, "request")
	if prefix != "" {
		s.rootTrace.Context.SetLabel("prefix", prefix)
	}
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

	// clean up
	if os.Getenv(envE2ECacheKeep) != "1" && os.Getenv(envE2ECacheKeys) != "1" {
		s.T().Logf("cleaning up directories")
		for _, node := range s.nodes {
			// safety
			if !strings.HasPrefix(s.currentDir, node.rootDir) {
				s.T().Logf("skipping external directory %s", node.rootDir)
				continue
			}

			err := os.RemoveAll(node.rootDir)
			if err != nil {
				s.T().Logf("failed to remove directory %s: %v", node.rootDir, err)
			}
		}
	}

	if s.rootTrace != nil {
		s.rootTrace.End()
		s.tracer.Flush(nil)
		s.tracer.Close()
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
	os.Setenv("GOLOG_LOG_LEVEL", "debug")
	// os.Setenv("DMS_OBSERVE_LEVEL", "debug")
	s.summary = &Summary{
		Nodes:         make(map[string]*SummaryNode),
		NodeConns:     make(map[string][]string),
		NodeTestConns: make(map[string][]string),
	}
	s.setupTestNetwork()
	s.runner(s)
}

func (s *TestSuite) printSummary() {
	if s.summary == nil {
		return
	}
	s.T().Log(s.summary.String())
}
