package cmd

import (
	"bytes"
	"testing"

	gonet "github.com/shirou/gopsutil/net"
	flag "github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/internal/config"
)

func GetMockConn(open bool) []gonet.ConnectionStat {
	dmsPort := config.GetConfig().Rest.Port

	conns := []gonet.ConnectionStat{
		{
			Laddr:  gonet.Addr{Port: dmsPort},
			Status: "",
		},
	}

	if open {
		conns[0].Status = "LISTEN"
	} else {
		conns[0].Status = "CLOSE"
	}

	return conns
}

func Test_ResourceConfigCmdHasFlags(t *testing.T) {
	conns := GetMockConn(true)
	mockConn := &MockConnection{conns: conns}
	mockUtils := &MockUtilsService{}

	cmd := NewResourceConfigCmd(mockConn, mockUtils)

	assert := assert.New(t)
	assert.True(cmd.HasAvailableFlags())

	expectedFlags := []string{"memory", "cpu", "ntx-price"}
	flags := cmd.Flags()
	flags.VisitAll(func(f *flag.Flag) {
		assert.Contains(expectedFlags, f.Name)
	})
}

func Test_ResourceConfigCmdMissingFlags(t *testing.T) {
	conns := GetMockConn(true)
	mockConn := &MockConnection{conns: conns}
	mockUtils := &MockUtilsService{}

	buf := new(bytes.Buffer)

	cmd := NewResourceConfigCmd(mockConn, mockUtils)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"--memory=0", "--cpu=0"})

	assert := assert.New(t)

	err := cmd.Execute()
	assert.Error(err)

	assert.Contains(buf.String(), "all flag values must be specified")
}

func Test_ResourceConfigCmdSuccess(t *testing.T) {
	conns := GetMockConn(true)
	mockConn := &MockConnection{conns: conns}

	mockUtils := &MockUtilsService{}
	mockUtils.SetResponseFor("POST", "/api/v1/onboarding/resource-config", []byte(`{ "message": "resources updated" }`))

	buf := new(bytes.Buffer)

	cmd := NewResourceConfigCmd(mockConn, mockUtils)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--memory=5000", "--cpu=4500"})

	assert := assert.New(t)

	err := cmd.Execute()
	assert.NoError(err)

	assert.Contains(buf.String(), "Resources updated successfully!")
}

func Test_ResourceConfigCmdErrorMessage(t *testing.T) {
	conns := GetMockConn(true)
	mockConn := &MockConnection{conns: conns}

	mockUtils := &MockUtilsService{}
	mockUtils.SetResponseFor("POST", "/api/v1/onboarding/resource-config", []byte(`{ "error": "bad error" }`))

	buf := new(bytes.Buffer)

	cmd := NewResourceConfigCmd(mockConn, mockUtils)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--memory=5000", "--cpu=4500"})

	assert := assert.New(t)

	err := cmd.Execute()
	assert.Error(err)

	assert.Contains(buf.String(), "bad error")
}
