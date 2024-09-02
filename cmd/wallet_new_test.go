package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWalletNewCmdCardano(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	buf := new(bytes.Buffer)

	cmd := newWalletNewCmd(ts.client)

	cmd.SetOut(buf)
	cmd.SetErr(buf)
	// testing with --cardano flag
	cmd.SetArgs([]string{"--cardano"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("error executing command: %v", err)
	}

	cardano, _ := cmd.Flags().GetBool("cardano")
	assert.True(t, cardano)
}

func TestWalletNewCmdEthereum(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	buf := new(bytes.Buffer)
	cmd := newWalletNewCmd(ts.client)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--ethereum"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("error executing command: %v", err)
	}

	cardano, _ := cmd.Flags().GetBool("cardano")
	assert.True(t, cardano)
}
