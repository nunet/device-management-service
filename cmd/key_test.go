package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/crypto/keystore"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/env"
	"gitlab.com/nunet/device-management-service/utils"
)

const (
	passphrase = "testpass"
)

// keyCmdDependencies holds all the necessary components for key command tests
type keyCmdDependencies struct {
	fs  afero.Afero
	cfg *config.Config
	env *env.MockEnvironment
}

// newKeyCmdDependencies creates a new test dependencies and returns it
func newKeyCmdDependencies(t *testing.T) *keyCmdDependencies {
	fs := afero.Afero{Fs: afero.NewMemMapFs()}
	cfg := &config.Config{
		General: config.General{
			UserDir: "/test/user",
		},
	}

	err := fs.MkdirAll(filepath.Join(cfg.General.UserDir, node.KeystoreDir), 0o755)
	require.NoError(t, err)

	mockEnv := env.NewMockEnvironment()

	deps := &keyCmdDependencies{
		fs:  fs,
		cfg: cfg,
		env: mockEnv,
	}

	return deps
}

func TestGenerateNewKey(t *testing.T) {
	t.Parallel()
	deps := newKeyCmdDependencies(t)

	var stdout, stdin bytes.Buffer
	cmd := newKeyNewCmd(deps.fs, deps.env, deps.cfg)
	cmd.SetOut(&stdout)
	cmd.SetIn(&stdin)

	testKeyName := "testkey"

	cmd.SetArgs([]string{testKeyName})

	err := deps.env.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	err = cmd.Execute()
	require.NoError(t, err)

	ks, err := keystore.New(deps.fs, filepath.Join(deps.cfg.General.UserDir,
		node.KeystoreDir))
	require.NoError(t, err)

	ok := ks.Exists(testKeyName)
	assert.True(t, ok)

	// Verify DID output
	didOutput := strings.TrimSpace(stdout.String())
	_, err = did.FromString(didOutput)
	require.NoError(t, err, "did should be valid")
}

func TestOverwriteExistingKeyWithConfirmation(t *testing.T) {
	t.Parallel()
	deps := newKeyCmdDependencies(t)

	// Create a key first
	var stdout1, stdin1 bytes.Buffer
	cmd1 := newKeyNewCmd(deps.fs, deps.env, deps.cfg)
	cmd1.SetOut(&stdout1)
	cmd1.SetIn(&stdin1)

	existingKeyName := "existingkey"

	cmd1.SetArgs([]string{existingKeyName})

	err := deps.env.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	err = cmd1.Execute()
	require.NoError(t, err)
	firstDID := strings.TrimSpace(stdout1.String())

	// Now try to create another key with the same name
	var stdout2, stdin2 bytes.Buffer
	// Provide "y" to the confirmation prompt
	stdin2.WriteString("y\n")

	cmd2 := newKeyNewCmd(deps.fs, deps.env, deps.cfg)
	cmd2.SetOut(&stdout2)
	cmd2.SetIn(&stdin2)
	cmd2.SetArgs([]string{existingKeyName})

	// Execute command again with the same key name
	err = cmd2.Execute()
	require.NoError(t, err)

	secondDID := strings.TrimSpace(stdout2.String())

	// The DIDs should be different since we generated a new key
	assert.NotEqual(t, firstDID, secondDID)
}

func TestCancelOverwriteOfExistingKey(t *testing.T) {
	t.Parallel()
	deps := newKeyCmdDependencies(t)

	// Create a key first
	var stdout1, stdin1 bytes.Buffer
	cmd1 := newKeyNewCmd(deps.fs, deps.env, deps.cfg)
	cmd1.SetOut(&stdout1)
	cmd1.SetIn(&stdin1)

	cancelKeyName := "cancelkey"

	cmd1.SetArgs([]string{cancelKeyName})

	err := deps.env.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	err = cmd1.Execute()
	require.NoError(t, err)

	// Now try to create another key with the same name
	var stdout2, stdin2 bytes.Buffer
	// Provide "n" to the confirmation prompt
	stdin2.WriteString("n\n")

	cmd2 := newKeyNewCmd(deps.fs, deps.env, deps.cfg)
	cmd2.SetOut(&stdout2)
	cmd2.SetIn(&stdin2)
	cmd2.SetArgs([]string{cancelKeyName})

	// Execute command again with the same key name
	err = cmd2.Execute()
	assert.Error(t, err, "expected an error when user cancels overwrite")
	assert.ErrorIs(t, err, utils.ErrOperationCancelled)
}

func TestGetKeyDID(t *testing.T) {
	t.Parallel()
	deps := newKeyCmdDependencies(t)

	// First create a key
	var newStdout, newStdin bytes.Buffer
	newCmd := newKeyNewCmd(deps.fs, deps.env, deps.cfg)
	newCmd.SetOut(&newStdout)
	newCmd.SetIn(&newStdin)

	testKeyName := "testdidkey"

	newCmd.SetArgs([]string{testKeyName})
	err := deps.env.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)

	err = newCmd.Execute()
	require.NoError(t, err)

	// Get the DID from the output of the new command
	expectedDID := strings.TrimSpace(newStdout.String())
	require.NotEmpty(t, expectedDID)

	// Now test the DID command
	var didStdout, didStdin bytes.Buffer
	didCmd := newKeyDIDCmd(deps.fs, deps.env, deps.cfg)
	didCmd.SetOut(&didStdout)
	didCmd.SetIn(&didStdin)

	didCmd.SetArgs([]string{testKeyName})

	err = didCmd.Execute()
	require.NoError(t, err)

	// Verify the output matches the expected DID
	actualDID := strings.TrimSpace(didStdout.String())
	assert.Equal(t, expectedDID, actualDID)
}

func TestGetKeyDIDNonExistentKey(t *testing.T) {
	t.Parallel()
	deps := newKeyCmdDependencies(t)

	var didStdout, didStdin bytes.Buffer
	didCmd := newKeyDIDCmd(deps.fs, deps.env, deps.cfg)
	didCmd.SetOut(&didStdout)
	didCmd.SetIn(&didStdin)

	nonExistentKeyName := "nonexistentkey"

	err := deps.env.Setenv(node.DMSPassphraseEnv, passphrase)
	require.NoError(t, err)
	didCmd.SetArgs([]string{nonExistentKeyName})

	// Execute the DID command with a non-existent key
	err = didCmd.Execute()
	assert.Error(t, err)
}

// Ensure `nunet key ledger-alias set <alias> <index>` writes the alias file and
// that ResolveLedgerIndex picks it up.
func TestLedgerAliasSetCommand(t *testing.T) {
	t.Parallel()

	deps := newKeyCmdDependencies(t)

	var stdout bytes.Buffer
	aliasCmd := newKeyLedgerAliasCmd(deps.fs, deps.cfg)
	aliasCmd.SetOut(&stdout)
	aliasCmd.SetArgs([]string{"set", "biz", "5"})

	require.NoError(t, aliasCmd.Execute())

	idx, err := node.ResolveLedgerIndex(deps.fs, deps.cfg.General.UserDir, "biz")
	require.NoError(t, err)
	require.Equal(t, 5, idx)

	require.Contains(t, stdout.String(), `Alias "biz"`)
	require.Contains(t, stdout.String(), "account 5")
}
