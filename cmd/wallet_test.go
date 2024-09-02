package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_WalletCmd(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	cmd := newWalletCmd(ts.client)

	assert.True(t, cmd.HasAvailableSubCommands())

	subcmd := []string{"new"}

	cmds := cmd.Commands()
	for _, child := range cmds {
		assert.Contains(t, subcmd, child.Name())
	}

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	assert.NoError(t, err)
}
