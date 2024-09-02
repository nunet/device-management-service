package cmd

import (
	"bytes"
	"testing"

	flag "github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

// checks if command has expected flags
func Test_PeerListCmdHasFlags(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	cmd := newPeerListCmd(ts.client)

	assert.True(t, cmd.HasAvailableFlags())

	expectedFlags := []string{"dht"}

	flags := cmd.Flags()
	flags.VisitAll(func(f *flag.Flag) {
		assert.Contains(t, expectedFlags, f.Name)
	})
}

// command output without passing flags
func Test_PeerListCmdNoFlag(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	buf := new(bytes.Buffer)
	cmd := newPeerListCmd(ts.client)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	assert.NoError(t, err)
}

// command output when passing all flags
func Test_PeerListCmdWithFlags(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	buf := new(bytes.Buffer)
	cmd := newPeerListCmd(ts.client)
	cmd.SetArgs([]string{"--dht"})
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	assert.NoError(t, err)
}

// command output when received message 'no peers found'
func Test_PeerListCmdWithMessage(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	buf := new(bytes.Buffer)
	cmd := newPeerListCmd(ts.client)
	cmd.SetArgs([]string{"--dht"})
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	assert.ErrorContains(t, err, "No peers found")
}

// command output if DHT array is empty
func Test_PeerListCmdEmptyDHTArray(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	buf := new(bytes.Buffer)
	cmd := newPeerListCmd(ts.client)
	cmd.SetArgs([]string{"--dht"})
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	assert.ErrorContains(t, err, "no DHT peers available")
}
