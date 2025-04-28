// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package itest

import (
	"bytes"
	"encoding/json"
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
	"gitlab.com/nunet/device-management-service/dms/hardware"
	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/types"
)

type Client struct {
	afero afero.Afero
	cfg   *config.Config
}

func newClient(t *testing.T, cfg *config.Config) *Client {
	t.Helper()
	afs := afero.Afero{Fs: afero.NewOsFs()}

	return &Client{
		afero: afs,
		cfg:   cfg,
	}
}

func (c *Client) newCommandCtx() *cobra.Command {
	return cmd.NewRootCMD(c.afero, c.cfg)
}

func (c *Client) newKey(t *testing.T, identity, passphrase string) {
	root := c.newCommandCtx()

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
	require.NoError(t, err)
	args := []string{"key", "new", identity}
	root.SetArgs(args)
	err = root.Execute()
	require.NoError(t, err)
}

func (c *Client) getDID(t *testing.T, context, passphrase string) string {
	err := os.Setenv("DMS_PASSPHRASE", passphrase)
	require.NoError(t, err)

	did, err := extractURIFromFile(filepath.Join(c.cfg.General.UserDir, "cap", context))
	require.NoError(t, err)
	return did
}

func (c *Client) newCap(t *testing.T, name, passphrase string) {
	root := c.newCommandCtx()

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
	require.NoError(t, err)
	args := []string{"cap", "new", name}
	root.SetArgs(args)
	err = root.Execute()
	require.NoError(t, err)
}

func (c *Client) addRootAnchor(t *testing.T, context, rootDID, passphrase string) {
	root := c.newCommandCtx()

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
	require.NoError(t, err)
	args := []string{"cap", "anchor", "--context", context, "--root", rootDID}
	root.SetArgs(args)
	err = root.Execute()
	require.NoError(t, err)
}

func (c *Client) grant(t *testing.T, context, otherDID, passphrase string) string {
	root := c.newCommandCtx()

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
	require.NoError(t, err)
	args := []string{"cap", "grant", "--context", context, "--cap", "/dms/volume/create", "--cap", "/public", "--cap", "/dms/deployment", "--cap", "/broadcast", "--topic", "/nunet", "--expiry", "2025-12-31", otherDID}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	require.NoError(t, err)
	return buf.String()
}

func (c *Client) delegate(t *testing.T, context, otherDID, passphrase string) string {
	root := c.newCommandCtx()

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
	require.NoError(t, err)
	args := []string{"cap", "delegate", "--context", context, "--cap", "/dms/volume/create", "--cap", "/public", "--cap", "/dms/deployment", "--cap", "/broadcast", "--topic", "/nunet", "--expiry", "2025-12-31", otherDID}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	require.NoError(t, err)
	return buf.String()
}

func (c *Client) anchor(t *testing.T, token, cxt, anchor, passphrase string) {
	root := c.newCommandCtx()

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
	require.NoError(t, err)

	args := []string{"cap", "anchor", "--context", cxt, "--" + anchor, token}
	root.SetArgs(args)
	err = root.Execute()
	require.NoError(t, err)
}

func (c *Client) onboard(t *testing.T, context, passphrase string) {
	root := c.newCommandCtx()

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
	require.NoError(t, err)

	hw := hardware.NewHardwareManager()
	mr, err := hw.GetMachineResources()
	require.NoError(t, err)
	// onboard with 40% of available ram and cpu
	ram := types.ConvertBytesToGB(uint64(float64(mr.Resources.RAM.Size) * 0.4))
	args := []string{"actor", "cmd", "--context", context, "/dms/node/onboarding/onboard", "--no-gpu", "--ram", fmt.Sprintf("%d GB", ram), "--cpu", fmt.Sprintf("%.2f", math.Ceil(float64(mr.Resources.CPU.Cores*0.4))), "--disk", "10GiB"}
	root.SetArgs(args)
	err = root.Execute()
	require.NoError(t, err)
}

func (c *Client) getOnboarded(t *testing.T, context, passphrase string) string {
	root := c.newCommandCtx()

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
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

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
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

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", context, "/public/hello", "--timeout", "5s", "--dest", dest}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	fmt.Println("hello response: ", buf.String())
	return buf.String(), err
}

func (c *Client) createVolume(t *testing.T, volName, pemFilePath, outputDir, context, passphrase, dest string) (string, error) {
	root := c.newCommandCtx()

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
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

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", context, "/dms/volume/start", "--name", volName, "--timeout", "5s", "--dest", dest}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	fmt.Println("createStart response: ", buf.String())
	return buf.String(), err
}

func (c *Client) deploy(t *testing.T, context, passphrase, specPath string) string {
	root := c.newCommandCtx()

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", context, "/dms/node/deployment/new", "--spec-file", specPath, "--timeout", "2m"}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	require.NoError(t, err)
	return buf.String()
}

func (c *Client) shutdownDeployment(t *testing.T, context, passphrase, deploymentID string) string {
	root := c.newCommandCtx()

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", context, "/dms/node/deployment/shutdown", "--id", deploymentID}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	require.NoError(t, err)
	return buf.String()
}

func (c *Client) revokeToken(t *testing.T, context, passphrase, token string) string {
	root := c.newCommandCtx()

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
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

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
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

func (c *Client) deploymentStatus(t *testing.T, context, passphrase, deploymentID string) string {
	root := c.newCommandCtx()

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", context, "/dms/node/deployment/status", "--id", deploymentID}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	require.NoError(t, err)
	return buf.String()
}

// nunet actor cmd --context user /dms/node/peers/connect --address /p2p/<peer_id>
func (c *Client) connect(t *testing.T, context, passphrase, hostID string) string {
	root := c.newCommandCtx()

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
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

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
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

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
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

func (c *Client) freeResources(t *testing.T, context, passphrase string) (types.Resources, error) {
	root := c.newCommandCtx()

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", context, "/dms/node/resources/free"}
	root.SetArgs(args)

	var buf bytes.Buffer
	root.SetOutput(&buf)
	err = root.Execute()
	if err != nil {
		return types.Resources{}, fmt.Errorf("failed to execute free resources command: %w", err)
	}

	res := node.ResourcesResponse{}
	if err = json.Unmarshal(buf.Bytes(), &res); err != nil {
		return types.Resources{}, fmt.Errorf("unmarshal res: %w", err)
	}

	if !res.OK {
		return types.Resources{}, fmt.Errorf("get free resources: %s", res.Error)
	}

	return res.Resources, nil
}
