// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package itest

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
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/dms"
	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/crypto/keystore"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
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
	re := regexp.MustCompile(`"Status":\s*"(.*?)"`)

	matches := re.FindStringSubmatch(input)

	if len(matches) < 2 {
		return ""
	}

	return matches[1]
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
			PortAvailableRangeTo:   90000,
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
		Profiler: config.Profiler{
			Enabled: false,
		},
		APM: config.APM{
			ServerURL:   "https://apm.telemetry.nunet.io",
			ServiceName: "nunet-dms",
			APIKey:      os.Getenv("ES_API"),
			Environment: "production",
		},
	}
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

// setupKeysAndCaps creates keys and capabilities for the given CLI instance.
func setupKeysAndCaps(t *testing.T, cli *Client, pass, keyType1, keyType2 string) {
	cli.newKey(t, keyType1, pass)
	cli.newKey(t, keyType2, pass)
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

func replaceHostnameInFile(filePath, hostname string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	modifiedContent := strings.ReplaceAll(string(content), "${hostname}", hostname)

	err = os.WriteFile(filePath, []byte(modifiedContent), 0o644)
	if err != nil {
		return fmt.Errorf("failed to write back to file: %w", err)
	}

	return nil
}

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

	shutdownCh chan struct{}
}

func newMockNode(
	t *testing.T, config *config.Config,
	password, rootDir string, index int,
) (*mockNode, error) {
	t.Helper()

	cliHelper := newClient(t, config)
	dmsContext := fmt.Sprintf("dms%d", index)
	userContext := fmt.Sprintf("user%d", index)
	setupKeysAndCaps(t, cliHelper, password, dmsContext, userContext)

	capCtx, err := loadCapCtx(t, cliHelper, password, dmsContext)
	if err != nil {
		return nil, fmt.Errorf("failed to load capability context: %w", err)
	}

	userDID := cliHelper.getDID(t, fmt.Sprintf("%s.cap", userContext), password)
	require.NotEmpty(t, userDID)
	dmsDID := capCtx.DID().String()
	require.NotEmpty(t, dmsDID)

	return &mockNode{
		config:      config,
		client:      cliHelper,
		password:    password,
		rootDir:     rootDir,
		index:       index,
		userDID:     userDID,
		dmsDID:      dmsDID,
		userContext: userContext,
		dmsContext:  dmsContext,
		capCtx:      capCtx,
		shutdownCh:  make(chan struct{}),
	}, nil
}

func loadCapCtx(
	t *testing.T, cliHelper *Client, password, dmsContext string,
) (ucan.CapabilityContext, error) {
	t.Helper()
	fs := afero.NewOsFs()

	keyStoreDir := filepath.Join(
		cliHelper.cfg.General.UserDir, node.KeystoreDir)
	keyStore, err := keystore.New(fs, keyStoreDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open keystore: %w", err)
	}

	privK, err := dms.GetPrivKeyFromKS(keyStore, password, dmsContext)
	if err != nil {
		return nil,
			fmt.Errorf("private key from keystore: %w", err)
	}
	pubKey := privK.GetPublic()

	dmsCtxPath := filepath.Join(
		cliHelper.cfg.General.UserDir, node.CapstoreDir, fmt.Sprintf("%s.cap", dmsContext))

	trustCtx, err := did.NewTrustContextWithPrivateKey(privK)
	if err != nil {
		return nil, fmt.Errorf("unable to create trust context: %w", err)
	}

	capCtx, err := dms.LoadOrCreateCapCtx(
		fs, dmsCtxPath, trustCtx, dmsContext, pubKey)
	if err != nil {
		return nil,
			fmt.Errorf(
				"unable to load or create capability context: %w", err)
	}

	trustCtx.Start(10 * time.Minute)
	capCtx.Start(5 * time.Minute)

	return capCtx, nil
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
