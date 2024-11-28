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
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/cmd"
	"gitlab.com/nunet/device-management-service/dms/hardware"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"
)

type cli struct {
	afero      afero.Afero
	httpClient *utils.HTTPClient
	cfg        *config.Config
	root       *cobra.Command
}

func newCLI(cfg *config.Config) *cli {
	afs := afero.Afero{Fs: afero.NewOsFs()}

	client := utils.NewHTTPClient(
		fmt.Sprintf("http://%s:%d",
			cfg.Rest.Addr,
			cfg.Rest.Port),
		"/api/v1",
	)

	return &cli{
		afero:      afs,
		httpClient: client,
		cfg:        cfg,
		root:       cmd.NewRootCMD(client, afs, cfg),
	}
}

func (c *cli) newKey(t *testing.T, identity, passphrase string) {
	c.newCommandCtx()

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
	require.NoError(t, err)
	args := []string{"key", "new", identity}
	c.root.SetArgs(args)
	err = c.root.Execute()
	require.NoError(t, err)
}

func (c *cli) getDID(t *testing.T, context, passphrase string) string {
	err := os.Setenv("DMS_PASSPHRASE", passphrase)
	require.NoError(t, err)

	did, err := extractURIFromFile(filepath.Join(c.cfg.General.UserDir, "cap", context))
	require.NoError(t, err)
	return did
}

func (c *cli) newCap(t *testing.T, name, passphrase string) {
	c.newCommandCtx()

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
	require.NoError(t, err)
	args := []string{"cap", "new", name}
	c.root.SetArgs(args)
	err = c.root.Execute()
	require.NoError(t, err)
}

func (c *cli) addRootAnchorToDMS(t *testing.T, userDID, passphrase string) {
	c.newCommandCtx()

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
	require.NoError(t, err)
	args := []string{"cap", "anchor", "--context", "dms", "--root", userDID}
	c.root.SetArgs(args)
	err = c.root.Execute()
	require.NoError(t, err)
}

func (c *cli) grant(t *testing.T, otherDID, passphrase string) string {
	c.newCommandCtx()

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
	require.NoError(t, err)
	args := []string{"cap", "grant", "--context", "dms", "--cap", "/public", "--cap", "/dms/deployment", "--cap", "/broadcast", "--topic", "/nunet", "--expiry", "2025-12-31", otherDID}
	c.root.SetArgs(args)

	var buf bytes.Buffer
	c.root.SetOutput(&buf)
	err = c.root.Execute()
	require.NoError(t, err)
	return buf.String()
}

func (c *cli) delegateToDMS(t *testing.T, otherDID, passphrase string) string {
	c.newCommandCtx()

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
	require.NoError(t, err)
	args := []string{"cap", "delegate", "--context", "user", "--cap", "/public", "--cap", "/dms/deployment", "--cap", "/broadcast", "--topic", "/nunet", "--expiry", "2025-12-31", otherDID}
	c.root.SetArgs(args)

	var buf bytes.Buffer
	c.root.SetOutput(&buf)
	err = c.root.Execute()
	require.NoError(t, err)
	return buf.String()
}

func (c *cli) anchorDMS(t *testing.T, token, cxt, anchor, passphrase string) {
	c.newCommandCtx()

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
	require.NoError(t, err)

	args := []string{"cap", "anchor", "--context", cxt, "--" + anchor, token}
	c.root.SetArgs(args)
	err = c.root.Execute()
	require.NoError(t, err)
}

func (c *cli) onboard(t *testing.T, passphrase string) {
	c.newCommandCtx()

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
	require.NoError(t, err)

	hw := hardware.NewHardwareManager()
	mr, err := hw.GetMachineResources()
	require.NoError(t, err)
	args := []string{"actor", "cmd", "--context", "user", "/dms/node/onboarding/onboard", "--no-gpu", "--ram", fmt.Sprintf("%.2f", types.ConvertBytesToGB(mr.Resources.RAM.Size*0.2)), "--cpu", fmt.Sprintf("%2f", math.Ceil(float64(mr.Resources.CPU.Cores*0.2))), "--disk", "100"}
	c.root.SetArgs(args)
	err = c.root.Execute()
	require.NoError(t, err)
}

func (c *cli) broadcast(t *testing.T, passphrase string) string {
	c.newCommandCtx()

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", "user", "/broadcast/hello", "--timeout", "5s"}
	c.root.SetArgs(args)

	var buf bytes.Buffer
	c.root.SetOutput(&buf)
	err = c.root.Execute()
	require.NoError(t, err)
	return buf.String()
}

func (c *cli) deploy(t *testing.T, passphrase, specPath string) string {
	c.newCommandCtx()

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", "user", "/dms/node/deployment/new", "--spec-file", specPath, "--timeout", "2m"}
	c.root.SetArgs(args)

	var buf bytes.Buffer
	c.root.SetOutput(&buf)
	err = c.root.Execute()
	require.NoError(t, err)
	return buf.String()
}

func (c *cli) shutdownDeployment(t *testing.T, passphrase, deploymentID string) string {
	c.newCommandCtx()

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", "user", "/dms/node/deployment/shutdown", "--id", deploymentID}
	c.root.SetArgs(args)

	var buf bytes.Buffer
	c.root.SetOutput(&buf)
	err = c.root.Execute()
	require.NoError(t, err)
	return buf.String()
}

func (c *cli) deploymentStatus(t *testing.T, passphrase, deploymentID string) string {
	c.newCommandCtx()

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", "user", "/dms/node/deployment/status", "--id", deploymentID}
	c.root.SetArgs(args)

	var buf bytes.Buffer
	c.root.SetOutput(&buf)
	err = c.root.Execute()
	require.NoError(t, err)
	return buf.String()
}

// nunet actor cmd --context user /dms/node/peers/connect --address /p2p/<peer_id>
func (c *cli) connect(t *testing.T, passphrase, hostID string) string {
	c.newCommandCtx()

	err := os.Setenv("DMS_PASSPHRASE", passphrase)
	require.NoError(t, err)

	args := []string{"actor", "cmd", "--context", "user", "/dms/node/peers/connect", "--address", "/p2p/" + hostID}
	c.root.SetArgs(args)

	var buf bytes.Buffer
	c.root.SetOutput(&buf)
	err = c.root.Execute()
	require.NoError(t, err)
	return buf.String()
}

func (c *cli) newCommandCtx() {
	c.root = cmd.NewRootCMD(c.httpClient, c.afero, c.cfg)
}
