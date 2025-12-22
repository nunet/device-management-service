// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/client"
	"github.com/shirou/gopsutil/v4/process"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/dms"
	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/types"
)

type DIDData struct {
	DID struct {
		URI string `json:"uri"`
	} `json:"DID"`
	Roots   []interface{}          `json:"Roots"`
	Require map[string]interface{} `json:"Require"`
	Provide map[string]interface{} `json:"Provide"`
}

func countDIDOccurrences(input string) int {
	pattern := `"DID":\s*{[^}]*}`

	re, err := regexp.Compile(pattern)
	if err != nil {
		fmt.Println("Error compiling regex:", err)
		return 0
	}

	matches := re.FindAllString(input, -1)

	return len(matches)
}

func getCurrentFileDirectory() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("Unable to get current file info")
	}
	return filepath.Dir(file)
}

func extractURIFromFile(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	var data DIDData
	if err := json.Unmarshal(content, &data); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	return data.DID.URI, nil
}

func extractEnsembleID(input string) string {
	re := regexp.MustCompile(`"EnsembleID":\s*"(.*?)"`)

	matches := re.FindStringSubmatch(input)

	if len(matches) < 2 {
		return ""
	}

	return matches[1]
}

func extractStatus(input string) string {
	re := regexp.MustCompile(`"status":\s*"(.*?)"`)

	matches := re.FindStringSubmatch(input)

	if len(matches) < 2 {
		return ""
	}

	return matches[1]
}

func createConfig(userDir string, restPort uint32, p2pListenAddrs []string, bootstrap []string) *config.Config {
	cfg := &config.Config{
		General: config.General{
			Env:                    "test",
			UserDir:                userDir,
			WorkDir:                filepath.Join(userDir, "work_dir"),
			DataDir:                filepath.Join(userDir, "data_dir"),
			Debug:                  true,
			HostCountry:            "NL",
			HostCity:               "Amsterdam",
			HostContinent:          "Europe",
			PortAvailableRangeFrom: 1024,
			PortAvailableRangeTo:   90000,
			StorageMode:            false,
			StorageCADirectory:     filepath.Join(userDir, "storage_ca_directory"),
			StorageBricksDir:       filepath.Join(userDir, "storage_bricks_dir"),
		},
		Rest: config.Rest{
			Addr: "localhost",
			Port: restPort,
		},
		Job: config.Job{
			AllowPrivilegedDocker: false,
		},
		P2P: config.P2P{
			ListenAddress:   p2pListenAddrs,
			BootstrapPeers:  bootstrap,
			Memory:          1024,
			FileDescriptors: 10444,
		},
		Observability: config.Observability{
			LogLevel:             "debug",
			LogFile:              filepath.Join(userDir, "logs.jsonl"),
			MaxSize:              100,
			MaxBackups:           3,
			MaxAge:               28,
			ElasticsearchURL:     "https://telemetry.nunet.io",
			ElasticsearchIndex:   "nunet-dms",
			FlushInterval:        3,
			ElasticsearchEnabled: false,
			ElasticsearchAPIKey:  os.Getenv("ES_API"),
			InsecureSkipVerify:   true,
		},
		Profiler: config.Profiler{
			Enabled: false,
		},
		APM: config.APM{
			ServerURL:   "https://apm.telemetry.nunet.io",
			ServiceName: "nunet-dms",
			Environment: "production",
			APIKey:      os.Getenv("ES_API"),
		},
	}

	// observability
	apiKey := os.Getenv(envE2EObserveAPIKey)
	token := os.Getenv(envE2EObserveToken)
	if apiKey != "" {
		cfg.Observability.Elastic.Enabled = true
		cfg.Observability.Elastic.APIKey = apiKey

		// if secrettoken is set, switch to local observability
		if token != "" {
			cfg.Observability.Elastic.URL = "https://localhost:9200"
			cfg.APM.ServerURL = "http://localhost:8200"
			cfg.APM.SecretToken = token
			cfg.APM.Environment = "development"
		}
	}

	return cfg
}

// getProc finds a process by pid.
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

// initCaps creates capabilities for the given config and keys.
func initCaps(t *testing.T, cli *Client, pass, keyType1, keyType2 string) {
	cli.newCap(t, keyType1, pass)
	cli.newCap(t, keyType2, pass)
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	return nil
}

func replaceContractInFile(filePath, contractData string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	modifiedContent := strings.ReplaceAll(string(content), "{$contract}", contractData)

	err = os.WriteFile(filePath, []byte(modifiedContent), 0o644)
	if err != nil {
		return fmt.Errorf("failed to write back to file: %w", err)
	}

	return nil
}

// MOCK NODE

type mockNode struct {
	index       int
	config      *config.Config
	client      *Client
	password    string
	rootDir     string
	peerID      string
	userDID     string
	dmsDID      string
	userContext string
	dmsContext  string
	capCtx      ucan.CapabilityContext

	privKey    crypto.PrivKey
	shutdownCh chan struct{}
	stopped    bool // tracks if the node has been stopped
}

func newMockNode(
	t *testing.T, cfg *config.Config, password, rootDir string, index int,
) (*mockNode, error) {
	t.Helper()

	cliHelper, err := newClient(t, cfg)
	require.NoError(t, err)

	dmsContext := fmt.Sprintf("dms%d", index)
	userContext := fmt.Sprintf("user%d", index)
	initCaps(t, cliHelper, password, dmsContext, userContext)

	capCtx, pkey, err := loadCapCtx(t, cliHelper, password, dmsContext)
	if err != nil {
		return nil, fmt.Errorf("failed to load capability context: %w", err)
	}

	userDID := cliHelper.getDID(t, fmt.Sprintf("%s.cap", userContext), password)
	require.NotEmpty(t, userDID)
	dmsDID := capCtx.DID().String()
	require.NotEmpty(t, dmsDID)

	return &mockNode{
		config:      cfg,
		client:      cliHelper,
		password:    password,
		rootDir:     rootDir,
		index:       index,
		userDID:     userDID,
		dmsDID:      dmsDID,
		userContext: userContext,
		dmsContext:  dmsContext,
		capCtx:      capCtx,
		privKey:     pkey,
		shutdownCh:  make(chan struct{}),
		stopped:     false,
	}, nil
}

func loadCapCtx(
	t *testing.T, cliHelper *Client, password, dmsContext string,
) (ucan.CapabilityContext, crypto.PrivKey, error) {
	t.Helper()

	privK, err := dms.GetPrivKeyFromKS(cliHelper.ks, password, dmsContext)
	if err != nil {
		return nil, nil,
			fmt.Errorf("private key from keystore: %w", err)
	}
	pubKey := privK.GetPublic()

	dmsCtxPath := filepath.Join(
		cliHelper.cfg.General.UserDir, node.CapstoreDir, fmt.Sprintf("%s.cap", dmsContext))

	trustCtx, err := did.NewTrustContextWithPrivateKey(privK)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create trust context: %w", err)
	}

	capCtx, err := dms.LoadOrCreateCapCtx(
		cliHelper.fs, dmsCtxPath, trustCtx, dmsContext, pubKey)
	if err != nil {
		return nil, nil,
			fmt.Errorf(
				"unable to load or create capability context: %w", err)
	}

	trustCtx.Start(10 * time.Minute)
	capCtx.Start(5 * time.Minute)

	return capCtx, privK, nil
}

func isContainerRunning(name string) (bool, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return false,
			fmt.Errorf("failed to create Docker Client: %w", err)
	}
	defer cli.Close()

	containerInfo, err := cli.ContainerInspect(
		context.Background(), name)
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to inspect container: %w", err)
	}

	return containerInfo.State.Running, nil
}

// checkResourcesDecreased verifies that resources have decreased on a node
func checkResourcesDecreased(t *testing.T, node *mockNode, before types.Resources) bool {
	after, err := node.client.freeResources(t, node.dmsContext, node.password)
	if err != nil {
		t.Logf("Error getting free resources: %v", err)
		return false
	}

	comparison, err := after.Compare(before)
	if err != nil {
		t.Logf("Error comparing resources: %v", err)
		return false
	}

	result := comparison == types.Worse
	if !result {
		t.Logf("Expected %s, but got %v. Before: %v, After: %v", types.Worse, comparison, before, after)
	}
	return result
}

// checkResourcesIncreased verifies that resources have increased on a node
func checkResourcesIncreased(t *testing.T, node *mockNode, before types.Resources) bool {
	after, err := node.client.freeResources(t, node.dmsContext, node.password)
	if err != nil {
		t.Logf("Error getting free resources: %v", err)
		return false
	}

	comparison, err := after.Compare(before)
	if err != nil {
		t.Logf("Error comparing resources: %v", err)
		return false
	}

	result := comparison == types.Better
	if !result {
		t.Logf("Expected %s, but got %v. Before: %v, After: %v", types.Better, comparison, before, after)
	}
	return result
}

// checkResourcesEqual verifies that resources are equal on a node
func checkResourcesEqual(t *testing.T, node *mockNode, expected types.Resources) bool {
	actual, err := node.client.freeResources(t, node.dmsContext, node.password)
	if err != nil {
		t.Logf("Error getting free resources: %v", err)
		return false
	}

	result := actual.Equal(expected)
	if !result {
		t.Logf("Expected resources to be equal, but got %v", actual)
	}
	return result
}
