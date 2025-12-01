// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package e2e

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/cmd"
	"gitlab.com/nunet/device-management-service/cmd/cli"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/crypto/keystore"
	"gitlab.com/nunet/device-management-service/lib/env"
	"gitlab.com/nunet/device-management-service/lib/hardware"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
	"gitlab.com/nunet/device-management-service/types"
)

type Client struct {
	fs  afero.Fs
	ks  keystore.KeyStore
	cfg *config.Config
	env env.EnvironmentProvider
}

func newClient(t *testing.T, cfg *config.Config) (*Client, error) {
	t.Helper()

	fs := afero.NewOsFs()
	ks, err := keystore.New(fs, filepath.Join(cfg.General.UserDir, node.KeystoreDir), true)
	if err != nil {
		return nil, fmt.Errorf("failed to create keystore: %w", err)
	}

	return &Client{
		fs:  fs,
		cfg: cfg,
		env: env.NewOSEnvironment(),
		ks:  ks,
	}, nil
}

// TODO rename to newCommandContainer?
func (c *Client) newCommandCtx() *cobra.Command {
	dmsCLI := cli.New(
		cli.WithConfig(c.cfg),
		cli.WithFS(c.fs),
		cli.WithEnv(env.NewOSEnvironment()),
		cli.WithKeystoreProvider(c.ks),
	)
	return cmd.NewRootCMD(dmsCLI)
}

func (c *Client) newKey(t *testing.T, identity, passphrase string) {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)
	args := []string{"key", "new", identity}
	root.SetArgs(args)
	err = root.Execute()
	require.NoError(t, err)
}

func (c *Client) getDID(t *testing.T, context, passphrase string) string {
	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	did, err := extractURIFromFile(filepath.Join(c.cfg.General.UserDir, "cap", context))
	require.NoError(t, err)
	return did
}

func (c *Client) newCap(t *testing.T, name, passphrase string) {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)
	args := []string{"cap", "new", "-f", name}
	root.SetArgs(args)
	err = root.Execute()
	require.NoError(t, err)
}

func (c *Client) addRootAnchor(t *testing.T, context, rootDID, passphrase string) {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)
	args := []string{"cap", "anchor", "--context", context, "--root", rootDID}
	root.SetArgs(args)
	err = root.Execute()
	require.NoError(t, err)
}

func (c *Client) grant(t *testing.T, context, otherDID, passphrase string) string {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)
	args := []string{"cap", "grant", "--context", context, "--cap", "/dms/tokenomics", "--cap", "/dms/tokenomics/contract/propose", "--cap", "/dms/tokenomics/contract/state", "--cap", "/dms/volume/create", "--cap", "/public", "--cap", "/dms/deployment", "--cap", "/broadcast", "--topic", "/nunet", "--expiry", "2025-12-31", otherDID}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	require.NoError(t, err)
	return buf.String()
}

func (c *Client) delegate(t *testing.T, context, otherDID, passphrase string) string {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)
	args := []string{"cap", "delegate", "--context", context, "--cap", "/dms/tokenomics", "--cap", "/dms/tokenomics/contract/propose", "--cap", "/dms/tokenomics/contract/state", "--cap", "/dms/volume/create", "--cap", "/public", "--cap", "/dms/deployment", "--cap", "/broadcast", "--topic", "/nunet", "--expiry", "2025-12-31", otherDID}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	require.NoError(t, err)
	return buf.String()
}

func (c *Client) anchor(t *testing.T, token, cxt, anchor, passphrase string) {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	args := []string{"cap", "anchor", "--context", cxt, "--" + anchor, token}
	root.SetArgs(args)
	err = root.Execute()
	require.NoError(t, err)
}

func (c *Client) onboard(t *testing.T, context, passphrase string) {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	hw := hardware.NewHardwareManager()
	mr, err := hw.GetMachineResources()
	require.NoError(t, err)

	// onboard with 40% of available ram and cpu
	ram := types.ConvertBytesToGB(uint64(float64(mr.Resources.RAM.Size) * 0.4))
	args := []string{
		"actor", "cmd", "--context", context, "/dms/node/onboarding/onboard",
		"--no-gpu", "--ram", fmt.Sprintf("%d GB", ram), "--cpu",
		fmt.Sprintf("%.2f", math.Ceil(float64(mr.Resources.CPU.Cores*0.4))),
		"--disk", "10GiB",
	}
	root.SetArgs(args)
	err = root.Execute()
	require.NoError(t, err)
}

func (c *Client) offboard(t *testing.T, context, passphrase string) {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)
	require.NoError(t, err)
	args := []string{"actor", "cmd", "--context", context, "/dms/node/onboarding/offboard"}
	root.SetArgs(args)
	err = root.Execute()
	require.NoError(t, err)
}

func (c *Client) getOnboarded(t *testing.T, context, passphrase string) string {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	onboardedArgs := []string{"actor", "cmd", "--context", context, "/dms/node/resources/onboarded"}
	root.SetArgs(onboardedArgs)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	require.NoError(t, err)
	return buf.String()
}

func (c *Client) broadcast(t *testing.T, context, passphrase string) string {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", context, "/broadcast/hello", "--timeout", "5s"}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	require.NoError(t, err)
	return buf.String()
}

func (c *Client) hello(t *testing.T, context, passphrase, dest string) (string, error) {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", context, "/public/hello", "--timeout", "5s", "--dest", dest}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	fmt.Println("hello response: ", buf.String())
	return buf.String(), err
}

func (c *Client) createContract(t *testing.T, contractFilePath, context, passphrase string) (string, error) {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", context, "/dms/tokenomics/contract/create", "--contract-file", contractFilePath, "--timeout", "5s"}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	return buf.String(), err
}

func (c *Client) listLocalTransactions(t *testing.T, context, passphrase string) (string, error) {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", context, "/dms/tokenomics/contract/transactions/list", "--timeout", "5s"}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	fmt.Println("listLocalTransactions response: ", buf.String())
	return buf.String(), err
}

func (c *Client) paymentStatus(t *testing.T, context, passphrase, uniqueID, dest string) (string, error) {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", context, "/dms/tokenomics/contract/payment/status", "--unique-id", uniqueID, "--timeout", "5s", "--dest", dest}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	fmt.Println("paymentStatus response: ", buf.String())
	return buf.String(), err
}

func (c *Client) terminateContract(t *testing.T, context, passphrase, contractDID, contractHostDID string) (string, error) { //nolint:unparam
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", context, "/dms/tokenomics/contract/terminate", "--contract-did", contractDID, "--contract-host-did", contractHostDID, "--timeout", "5s"}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	fmt.Println("terminateContract response: ", buf.String())
	return buf.String(), err
}

func (c *Client) validateContract(t *testing.T, context, passphrase, contractDID, contractHostDID string) (string, error) {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", context, "/dms/tokenomics/contract/validate", "--contract-did", contractDID, "--contract-host-did", contractHostDID, "--timeout", "5s"}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	fmt.Println("validateContract response: ", buf.String())
	return buf.String(), err
}

func (c *Client) settleContract(t *testing.T, context, passphrase, contractDID, contractHostDID string) (string, error) { //nolint:unparam
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", context, "/dms/tokenomics/contract/settle", "--contract-did", contractDID, "--contract-host-did", contractHostDID, "--timeout", "5s"}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	fmt.Println("settleContract response: ", buf.String())
	return buf.String(), err
}

func (c *Client) confirmLocalTransaction(t *testing.T, context, passphrase, uniqueID, txHash string) (string, error) { //nolint:unparam
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", context, "/dms/tokenomics/contract/transactions/confirm", "--unique-id", uniqueID, "--tx-hash", txHash, "--blockchain", "ETHEREUM", "--timeout", "5s"}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	fmt.Println("confirmLocalTransaction response: ", buf.String())
	return buf.String(), err
}

func (c *Client) calculateContractUsages(t *testing.T, context, passphrase string, contractDID ...string) (string, error) {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", context, "/dms/tokenomics/contract/usages/calculate", "--timeout", "5s"}
	if len(contractDID) > 0 && contractDID[0] != "" {
		args = append(args, "--contract-did", contractDID[0])
	}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	fmt.Println("calculateContractUsages response: ", buf.String())
	return buf.String(), err
}

func (c *Client) approveContracts(t *testing.T, contractDID, context, passphrase string) (string, error) {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", context, "/dms/tokenomics/contract/approve_local", "--contract-did", contractDID, "--timeout", "5s"}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	fmt.Println("approveContracts response: ", buf.String())
	return buf.String(), err
}

func (c *Client) listIncomingContracts(t *testing.T, context, passphrase string) (string, error) {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	args := []string{"contracts", "list", "--context", context, "incoming", "--timeout", "5s"}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	return buf.String(), err
}

func (c *Client) listOutgoingContracts(t *testing.T, context, passphrase string) ([]*contracts.Contract, error) {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	args := []string{"contracts", "list", "--context", context, "outgoing", "--timeout", "5s"}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	require.NoError(t, err)

	// parse json response
	var resp contracts.ContractListIncomingResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cmd output: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("failed to list outgoing contracts: %s", resp.Error)
	}
	return resp.Contracts, nil
}

func (c *Client) contractStatus(t *testing.T, context, passphrase, contractDID, contractHostDID string) (string, error) {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", context, "/dms/tokenomics/contract/state", "--contract-did", contractDID, "--contract-host-did", contractHostDID, "--timeout", "25s"}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	fmt.Println("contractStatus response: ", buf.String())
	return buf.String(), err
}

func (c *Client) createVolume(t *testing.T, volName, pemFilePath, outputDir, context, passphrase, dest string) (string, error) {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", context, "/dms/volume/create", "--name", volName, "--client-pem-file", pemFilePath, "--ca-output-dir", outputDir, "--timeout", "5s", "--dest", dest}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	fmt.Println("createVolume response: ", buf.String())
	return buf.String(), err
}

func (c *Client) startVolume(t *testing.T, volName, context, passphrase, dest string) (string, error) {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", context, "/dms/volume/start", "--name", volName, "--timeout", "5s", "--dest", dest}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	fmt.Println("createStart response: ", buf.String())
	return buf.String(), err
}

func (c *Client) deploy(t *testing.T, context, passphrase, specPath, timeout string) string {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", context, "/dms/node/deployment/new", "--spec-file", specPath, "--timeout", timeout}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	require.NoError(t, err)
	return buf.String()
}

func (c *Client) update(t *testing.T, context, passphrase, specPath, ensembleID string) string {
	root := c.newCommandCtx()

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
	require.NoError(t, err)

	args := []string{
		"actor", "cmd", "--context",
		context, "/dms/node/deployment/update", "--spec-file",
		specPath, "--timeout", "2m",
		"--id", ensembleID,
	}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	require.NoError(t, err)
	return buf.String()
}

func (c *Client) shutdownDeployment(t *testing.T, context, passphrase, deploymentID string) string {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", context, "/dms/node/deployment/shutdown", "--id", deploymentID, "--timeout", "15m"}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	require.NoError(t, err)
	return buf.String()
}

func (c *Client) revokeToken(t *testing.T, context, passphrase, token string) string {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	args := []string{"cap", "revoke", "--context", context, token}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	fmt.Println("revokeToken output", buf.String())
	err = root.Execute()
	require.NoError(t, err)
	return buf.String()
}

func (c *Client) anchorBehaviour(t *testing.T, context, passphrase, token string) string {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", context, "/dms/cap/anchor", "--revoke", "--data", strings.TrimSpace(token)}
	fmt.Println("args", args)
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	require.NoError(t, err)
	return buf.String()
}

func (c *Client) deploymentStatus(t *testing.T, context, passphrase, deploymentID string) (string, error) {
	t.Helper()
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	if err != nil {
		return "", fmt.Errorf("failed to set env: %w", err)
	}

	args := []string{"actor", "cmd", "--context", context, "/dms/node/deployment/status", "--id", deploymentID, "--timeout", "10m"}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	if err != nil {
		return "", fmt.Errorf("failed to execute deployment status command: %w", err)
	}
	return buf.String(), nil
}

func (c *Client) deploymentManifest(
	context, passphrase, deploymentID string,
) (jobtypes.EnsembleManifest, error) {
	var resp node.DeploymentManifestResponse

	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	if err != nil {
		return jobtypes.EnsembleManifest{}, fmt.Errorf("failed to set env: %w", err)
	}

	args := []string{
		"actor", "cmd", "--context",
		context, "/dms/node/deployment/manifest", "--id", deploymentID,
	}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	if err != nil {
		return jobtypes.EnsembleManifest{},
			fmt.Errorf("failed to execute manifest command: %w", err)
	}

	err = json.Unmarshal(buf.Bytes(), &resp)
	if err != nil {
		return jobtypes.EnsembleManifest{},
			fmt.Errorf("unmarshal manifest: %w", err)
	}

	if resp.Error != "" {
		return jobtypes.EnsembleManifest{},
			errors.New(resp.Error)
	}

	return resp.Manifest, nil
}

func (c *Client) allocationsList(context, passphrase string) ([]jobs.AllocationInfo, error) {
	var resp node.AllocationsListResponse

	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	if err != nil {
		return []jobs.AllocationInfo{},
			fmt.Errorf("failed to set env: %w", err)
	}

	args := []string{"actor", "cmd", "--context", context, "/dms/node/allocations/list"}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	if err != nil {
		return []jobs.AllocationInfo{},
			fmt.Errorf("failed to execute allocations list command: %w", err)
	}

	err = json.Unmarshal(buf.Bytes(), &resp)
	if err != nil {
		return []jobs.AllocationInfo{},
			fmt.Errorf("unmarshal res: %w", err)
	}

	if resp.Error != "" {
		return []jobs.AllocationInfo{},
			errors.New(resp.Error)
	}

	return resp.Allocations, nil
}

// nunet actor cmd --context user /dms/node/peers/connect --address /p2p/<peer_id>
func (c *Client) connect(t *testing.T, context, passphrase, hostID string) string {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", context, "/dms/node/peers/connect", "--address", "/p2p/" + hostID}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	require.NoError(t, err)
	return buf.String()
}

func (c *Client) self(t *testing.T, context, passphrase string) (types.NetworkStats, error) {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", context, "/dms/node/peers/self"}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	if err != nil {
		return types.NetworkStats{}, fmt.Errorf("failed to execute self command: %w", err)
	}

	stats := types.NetworkStats{}
	err = json.Unmarshal(buf.Bytes(), &stats)
	return stats, err
}

func (c *Client) peers(t *testing.T, context, passphrase string) (node.PeersListResponse, error) {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", context, "/dms/node/peers/list"}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	if err != nil {
		return node.PeersListResponse{}, fmt.Errorf("failed to execute peers list command: %w", err)
	}

	resp := node.PeersListResponse{}
	err = json.Unmarshal(buf.Bytes(), &resp)
	return resp, err
}

func (c *Client) getResources(
	resourceType, context, passphrase string,
) (types.Resources, error) {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	if err != nil {
		return types.Resources{}, fmt.Errorf("failed to set env: %w", err)
	}

	args := []string{"actor", "cmd", "--context", context, "/dms/node/resources/" + resourceType}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	if err != nil {
		return types.Resources{}, fmt.Errorf("failed to execute %s resources command: %w", resourceType, err)
	}

	res := node.ResourcesResponse{}
	if err = json.Unmarshal(buf.Bytes(), &res); err != nil {
		return types.Resources{}, fmt.Errorf("unmarshal res: %w", err)
	}

	if !res.OK {
		return types.Resources{}, fmt.Errorf("get %s resources: %s", resourceType, res.Error)
	}

	return res.Resources, nil
}

func (c *Client) freeResources(_ *testing.T, context, passphrase string) (types.Resources, error) {
	return c.getResources("free", context, passphrase)
}

func (c *Client) onboardedResources(_ *testing.T, context, passphrase string) (types.Resources, error) {
	return c.getResources("onboarded", context, passphrase)
}

func (c *Client) allocatedResources(_ *testing.T, context, passphrase string) (types.Resources, error) {
	return c.getResources("allocated", context, passphrase)
}

func (c *Client) deploymentList(t *testing.T, context, passphrase string) (map[string]string, error) {
	t.Helper()
	var resp node.DeploymentListResponse

	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	if err != nil {
		return map[string]string{}, fmt.Errorf("failed to set env: %w", err)
	}

	args := []string{"actor", "cmd", "--context", context, "/dms/node/deployment/list"}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	if err != nil {
		return map[string]string{}, fmt.Errorf("failed to execute deployment list command: %w", err)
	}

	err = json.Unmarshal(buf.Bytes(), &resp)
	if err != nil {
		return map[string]string{}, fmt.Errorf("unmarshal deployment list response: %w", err)
	}

	return resp.Deployments, nil
}

func (c *Client) deploymentLogs(context, passphrase, deploymentID, allocationName string) (node.DeploymentLogsResponse, error) {
	var resp node.DeploymentLogsResponse

	root := c.newCommandCtx()

	if err := os.Setenv(node.DMSPassphraseEnv, passphrase); err != nil {
		return node.DeploymentLogsResponse{}, fmt.Errorf("failed to set env: %w", err)
	}

	args := []string{
		"actor", "cmd", "--context", context,
		"/dms/node/deployment/logs",
		"--id", deploymentID,
		"--allocation", allocationName,
	}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	if err := root.Execute(); err != nil {
		return node.DeploymentLogsResponse{}, fmt.Errorf("failed to execute deployment logs command: %w", err)
	}

	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		return node.DeploymentLogsResponse{}, fmt.Errorf("unmarshal deployment logs response: %w", err)
	}

	if resp.Error != "" {
		return node.DeploymentLogsResponse{}, fmt.Errorf("deployment logs error: %s", resp.Error)
	}

	return resp, nil
}

func (c *Client) debugFlightrec(t *testing.T, context, passphrase string) (string, error) {
	root := c.newCommandCtx()

	err := os.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", context, "/dms/debug/flightrec"}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	fmt.Println("createVolume response: ", buf.String())
	return buf.String(), err
}
