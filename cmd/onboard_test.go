package cmd

import (
	"bytes"
	"testing"

	flag "github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func Test_OnboardCmdFlags(t *testing.T) {
	conns := GetMockConn(true)
	mockConn := &MockConnection{conns: conns}
	mockUtils := &MockUtilsService{}

	cmd := NewOnboardCmd(mockConn, mockUtils)

	assert := assert.New(t)
	assert.True(cmd.HasAvailableFlags())

	expectedFlags := []string{"memory", "cpu", "nunet-channel", "address", "plugin", "local-enable", "cardano", "unavailable", "ntx-price"}
	flags := cmd.Flags()
	flags.VisitAll(func(f *flag.Flag) {
		assert.Contains(expectedFlags, f.Name)
	})
}

func Test_OnboardCmdMissingMemory(t *testing.T) {
	conns := GetMockConn(true)
	mockConn := &MockConnection{conns: conns}
	mockUtils := &MockUtilsService{}

	assert := assert.New(t)

	buf := new(bytes.Buffer)

	cmd := NewOnboardCmd(mockConn, mockUtils)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"--memory=0", "--cpu=5000", "--nunet-channel=nunet-test", "--address=addr1_qtest123"})

	err := cmd.Execute()
	assert.ErrorContains(err, "memory must be provided and greater than 0")
}

func Test_OnboardCmdMissingCpu(t *testing.T) {
	conns := GetMockConn(true)
	mockConn := &MockConnection{conns: conns}
	mockUtils := &MockUtilsService{}

	assert := assert.New(t)

	buf := new(bytes.Buffer)

	cmd := NewOnboardCmd(mockConn, mockUtils)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"--memory=3000", "--cpu=0", "--nunet-channel=nunet-test", "--address=addr1_qtest123"})

	err := cmd.Execute()
	assert.ErrorContains(err, "cpu must be provided and greater than 0")
}

func Test_OnboardCmdMissingAddress(t *testing.T) {
	conns := GetMockConn(true)
	mockConn := &MockConnection{conns: conns}
	mockUtils := &MockUtilsService{}

	assert := assert.New(t)

	buf := new(bytes.Buffer)

	cmd := NewOnboardCmd(mockConn, mockUtils)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"--memory=3000", "--cpu=5000", "--nunet-channel=nunet-test", "--address="})

	err := cmd.Execute()
	assert.ErrorContains(err, "address must be provided and non-empty")
}

func Test_OnboardCmdSuccess(t *testing.T) {
	conns := GetMockConn(true)
	mockConn := &MockConnection{conns: conns}
	mockUtils := &MockUtilsService{}
	mockUtils.SetResponseFor("POST", "/api/v1/onboarding/onboard", []byte(`{ "message": "test" }`))

	cmd := NewOnboardCmd(mockConn, mockUtils)

	outBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)
	cmd.SetErr(outBuf)

	cmd.SetArgs([]string{"--memory=3000", "--cpu=5000", "--nunet-channel=nunet-test", "--address=addr1_qtest123"})

	// answer if it prompts for reonboard
	inBuf := bytes.NewBufferString("y\n")
	cmd.SetIn(inBuf)

	assert := assert.New(t)

	err := cmd.Execute()
	assert.NoError(err)

	assert.Contains(outBuf.String(), "Successfully onboarded!")
}

func Test_OnboardNegativeNtxValue(t *testing.T) {
	conns := GetMockConn(true)
	mockConn := &MockConnection{conns: conns}
	mockUtils := &MockUtilsService{}

	assert := assert.New(t)

	buf := new(bytes.Buffer)

	cmd := NewOnboardCmd(mockConn, mockUtils)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{
		"--ntx-price=-1",
		"--memory=3000",
		"--cpu=5000",
		"--nunet-channel=nunet-test",
		"--address=addr1_qtest123",
	})

	err := cmd.Execute()
	assert.NotNil(err, "expected ''ntx-price' must be a positive value' error")
	assert.ErrorContains(err, "'ntx-price' must be a positive value")
}
