package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_InfoCmd(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	// TODO: get info from onboarding db repo
	expectedResponse := `+----------------------+----------+\n
|         INFO         |  VALUE   |\n"
+----------------------+----------+\n"
| Name                 | metadata |\n"
| Update Timestamp     |        0 |\n"
| Memory Max           |      256 |\n"
| Total Core           |        4 |\n"
| CPU Max              |      700 |\n"
| Available CPU        |      690 |\n"
| Available Memory     |      246 |\n"
| Reserved CPU         |       10 |\n"
| Reserved Memory      |       10 |\n"
| Network              | tcp      |\n"
| Public Key           | abc123   |\n"
| Node ID              |          |\n"
| Allow Cardano        | false    |\n"
| Dashboard            |          |\n"
| NTX Price Per Minute | 0.000000 |\n"
+----------------------+----------+\n"`

	cmd := newInfoCmd(ts.client)

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("error executing command: %v", err)
	}

	expected := new(bytes.Buffer)
	assert.Equal(t, expectedResponse, buf.String())

	buf.Reset()
	expected.Reset()
}

func Test_InfoCmdNotOnboarded(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	buf := new(bytes.Buffer)
	cmd := newInfoCmd(ts.client)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	assert.ErrorContains(t, err, "not onboarded")
}

func Test_InfoCmdInvalidMetadata(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	buf := new(bytes.Buffer)
	cmd := newInfoCmd(ts.client)

	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	assert.ErrorContains(t, err, "cannot read file")
}

func Test_InfoCmdDMSNotRunning(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	buf := new(bytes.Buffer)
	cmd := newInfoCmd(ts.client)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	assert.ErrorContains(t, err, "looks like DMS is not running...")
}
