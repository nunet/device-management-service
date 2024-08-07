package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/cmd/backend"
)

func Test_DeviceCmdSubCommands(t *testing.T) {
	conns := GetMockConn(true)
	mockConn := &MockConnection{conns: conns}

	cmd := NewDeviceCmd(mockConn)

	assert := assert.New(t)
	assert.True(cmd.HasAvailableSubCommands())

	subcmd := []string{"status", "set"}

	cmds := cmd.Commands()
	for _, child := range cmds {
		assert.Contains(subcmd, child.Name())
	}

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	assert.NoError(err)
}

func Test_DeviceStatusCmd(t *testing.T) {
	assert := assert.New(t)

	err := setupMockDB()
	assert.NoError(err)

	mockUtils := &MockUtilsService{}

	selfResponse := []byte(`{"online":false}`)
	mockUtils.SetResponseFor("GET", "/api/v1/device/status", selfResponse)

	buf := new(bytes.Buffer)
	cmd := NewDeviceStatusCmd(&backend.Utils{})
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err = cmd.Execute()
	assert.NoError(err)

	assert.Contains(buf.String(), "Status: offline")
}

func Test_DeviceSetCmd(t *testing.T) {
	assert := assert.New(t)

	err := setupMockDB()
	assert.NoError(err)

	mockUtils := &MockUtilsService{}

	postResponse := []byte(`{"message":"Device status successfully changed to online"}`)
	mockUtils.SetResponseFor("POST", "/api/v1/device/status", postResponse)

	// no argument
	buf := new(bytes.Buffer)
	cmd := NewDeviceSetCmd(&backend.Utils{})
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	err = cmd.Execute()
	assert.Error(err)
	assert.Contains(buf.String(), "Error: invalid number of arguments")

	// with argument
	buf = new(bytes.Buffer)
	cmd = NewDeviceSetCmd(&backend.Utils{})
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"online"})
	err = cmd.Execute()
	assert.NoError(err)

	assert.Contains(buf.String(), "Device status successfully changed to online")
}
