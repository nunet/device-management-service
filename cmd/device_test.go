package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_DeviceCmdSubCommands(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	cmd := newDeviceCmd(ts.client)

	assert.True(t, cmd.HasAvailableSubCommands())

	subcmd := []string{"status", "set"}

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

func Test_DeviceStatusCmd(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	buf := new(bytes.Buffer)
	cmd := newDeviceStatusCmd(ts.client)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	assert.NoError(t, err)

	assert.Contains(t, buf.String(), "Status: offline")
}

func Test_DeviceSetCmd(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	// no argument
	buf := new(bytes.Buffer)
	cmd := newDeviceSetCmd(ts.client)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, buf.String(), "Error: invalid number of arguments")

	// with argument
	buf = new(bytes.Buffer)
	cmd = newDeviceSetCmd(ts.client)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"online"})
	err = cmd.Execute()
	assert.NoError(t, err)

	assert.Contains(t, buf.String(), "Device status successfully changed to online")
}
