// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cmdUtils "gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/lib/crypto/keystore"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/utils"
)

func TestGenerateNewKey(t *testing.T) {
	t.Parallel()

	dmsCli := cmdUtils.NewTestCli()
	cfg, err := dmsCli.Config()
	require.NoError(t, err)
	fs := dmsCli.FS()

	cmd := newKeyNewCmd(dmsCli)
	testKeyName := "testkey"
	out, _, err := cmdUtils.ExecuteCommand(cmd, testKeyName)
	require.NoError(t, err)

	ks, err := keystore.New(fs, filepath.Join(cfg.General.UserDir, node.KeystoreDir), false)
	require.NoError(t, err)

	ok := ks.Exists(testKeyName)
	assert.True(t, ok)

	// Verify DID output
	didOutput := strings.TrimSpace(out)
	_, err = did.FromString(didOutput)
	require.NoError(t, err, "did should be valid")
}

func TestOverwriteExistingKeyWithConfirmation(t *testing.T) {
	t.Parallel()

	dmsCli := cmdUtils.NewTestCli()

	// Create a key first
	cmd1 := newKeyNewCmd(dmsCli)
	existingKeyName := "existingkey"
	out1, _, err := cmdUtils.ExecuteCommand(cmd1, existingKeyName)
	require.NoError(t, err)

	firstDID := strings.TrimSpace(out1)

	// Now try to create another key with the same name
	cmd2 := newKeyNewCmd(dmsCli)
	out2, _, err := cmdUtils.ExecuteCommandWithInput(cmd2, [][]byte{[]byte("y\n")}, existingKeyName)
	require.NoError(t, err)

	secondDID := strings.TrimSpace(out2)

	// The DIDs should be different since we generated a new key
	assert.NotEqual(t, firstDID, secondDID)
}

func TestCancelOverwriteOfExistingKey(t *testing.T) {
	t.Parallel()

	dmsCli := cmdUtils.NewTestCli()

	// Create a key first
	cmd1 := newKeyNewCmd(dmsCli)
	cancelKeyName := "cancelkey"

	_, _, err := cmdUtils.ExecuteCommand(cmd1, cancelKeyName)
	require.NoError(t, err)

	// Now try to create another key with the same name
	cmd2 := newKeyNewCmd(dmsCli)
	_, _, err = cmdUtils.ExecuteCommandWithInput(cmd2, [][]byte{[]byte("n\n")}, cancelKeyName)
	assert.Error(t, err, "expected an error when user cancels overwrite")
	assert.ErrorIs(t, err, utils.ErrOperationCancelled)
}

func TestGetKeyDID(t *testing.T) {
	t.Parallel()

	dmsCli := cmdUtils.NewTestCli()

	// First create a key
	newCmd := newKeyNewCmd(dmsCli)
	testKeyName := "testdidkey"

	newOut, _, err := cmdUtils.ExecuteCommand(newCmd, testKeyName)
	require.NoError(t, err)

	// Get the DID from the output of the new command
	expectedDID := strings.TrimSpace(newOut)
	require.NotEmpty(t, expectedDID)

	// Now test the DID command
	didCmd := newKeyDIDCmd(dmsCli)

	didOut, _, err := cmdUtils.ExecuteCommand(didCmd, testKeyName)
	require.NoError(t, err)

	// Verify the output matches the expected DID
	actualDID := strings.TrimSpace(didOut)
	assert.Equal(t, expectedDID, actualDID)
}

func TestGetKeyDIDNonExistentKey(t *testing.T) {
	t.Parallel()

	dmsCli := cmdUtils.NewTestCli()

	didCmd := newKeyDIDCmd(dmsCli)

	nonExistentKeyName := "nonexistentkey"
	_, _, err := cmdUtils.ExecuteCommand(didCmd, nonExistentKeyName)
	assert.Error(t, err)
}

// Ensure `nunet key ledger-alias set <alias> <index>` writes the alias file and
// that ResolveLedgerIndex picks it up.
func TestLedgerAliasSetCommand(t *testing.T) {
	t.Parallel()

	dmsCli := cmdUtils.NewTestCli()

	var stdout bytes.Buffer
	aliasCmd := newKeyLedgerAliasCmd(dmsCli)
	aliasCmd.SetOut(&stdout)
	aliasCmd.SetArgs([]string{"set", "biz", "5"})

	require.NoError(t, aliasCmd.Execute())

	cfg, err := dmsCli.Config()
	require.NoError(t, err)

	idx, err := node.ResolveLedgerIndex(dmsCli.FS(), cfg.General.UserDir, "biz")
	require.NoError(t, err)
	require.Equal(t, 5, idx)

	require.Contains(t, stdout.String(), `Alias "biz"`)
	require.Contains(t, stdout.String(), "account 5")
}
