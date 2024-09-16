package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_LogDarwinCmdSuccess(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	t.Skipf("Not yet implemented")
}
