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
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	cmd := newResourceConfigCmd(ts.client)

	assert.True(t, cmd.HasAvailableFlags())

	expectedFlags := []string{"memory", "cpu", "ntx-price"}
	flags := cmd.Flags()
	flags.VisitAll(func(f *flag.Flag) {
		assert.Contains(t, expectedFlags, f.Name)
	})
}

func Test_ResourceConfigCmdMissingFlags(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	buf := new(bytes.Buffer)
	cmd := newResourceConfigCmd(ts.client)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"--memory=0", "--cpu=0"})

	err := cmd.Execute()
	assert.Error(t, err)

	assert.Contains(t, buf.String(), "all flag values must be specified")
}

func Test_ResourceConfigCmdSuccess(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	buf := new(bytes.Buffer)
	cmd := newResourceConfigCmd(ts.client)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--memory=5000", "--cpu=4500"})

	err := cmd.Execute()
	assert.NoError(t, err)

	assert.Contains(t, buf.String(), "Resources updated successfully!")
}

func Test_ResourceConfigCmdErrorMessage(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	buf := new(bytes.Buffer)

	cmd := newResourceConfigCmd(ts.client)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--memory=5000", "--cpu=4500"})

	err := cmd.Execute()
	assert.Error(t, err)

	assert.Contains(t, buf.String(), "bad error")
}
