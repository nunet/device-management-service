package cmd

import (
	"bytes"
	"testing"

	flag "github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func Test_OnboardCmdFlags(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	cmd := newOnboardCmd(ts.client)

	assert.True(t, cmd.HasAvailableFlags())

	expectedFlags := []string{"memory", "cpu", "nunet-channel", "address", "plugin", "local-enable", "cardano", "unavailable", "ntx-price"}
	flags := cmd.Flags()
	flags.VisitAll(func(f *flag.Flag) {
		assert.Contains(t, expectedFlags, f.Name)
	})
}

func Test_OnboardCmdMissingMemory(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	buf := new(bytes.Buffer)

	cmd := newOnboardCmd(ts.client)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"--memory=0", "--cpu=5000", "--nunet-channel=nunet-test", "--address=addr1_qtest123"})

	err := cmd.Execute()
	assert.ErrorContains(t, err, "memory must be provided and greater than 0")
}

func Test_OnboardCmdMissingCpu(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	buf := new(bytes.Buffer)

	cmd := newOnboardCmd(ts.client)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"--memory=3000", "--cpu=0", "--nunet-channel=nunet-test", "--address=addr1_qtest123"})

	err := cmd.Execute()
	assert.ErrorContains(t, err, "cpu must be provided and greater than 0")
}

func Test_OnboardCmdMissingAddress(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	buf := new(bytes.Buffer)

	cmd := newOnboardCmd(ts.client)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cmd.SetArgs([]string{"--memory=3000", "--cpu=5000", "--nunet-channel=nunet-test", "--address="})

	err := cmd.Execute()
	assert.ErrorContains(t, err, "address must be provided and non-empty")
}

func Test_OnboardCmdSuccess(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	cmd := newOnboardCmd(ts.client)

	outBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)
	cmd.SetErr(outBuf)

	cmd.SetArgs([]string{"--memory=3000", "--cpu=5000", "--nunet-channel=nunet-test", "--address=addr1_qtest123"})

	// answer if it prompts for reonboard
	inBuf := bytes.NewBufferString("y\n")
	cmd.SetIn(inBuf)

	err := cmd.Execute()
	assert.NoError(t, err)

	assert.Contains(t, outBuf.String(), "Successfully onboarded!")
}

func Test_OnboardNegativeNtxValue(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	buf := new(bytes.Buffer)

	cmd := newOnboardCmd(ts.client)
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
	assert.NotNil(t, err, "expected ''ntx-price' must be a positive value' error")
	assert.ErrorContains(t, err, "'ntx-price' must be a positive value")
}
