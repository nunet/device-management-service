package cmd

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_SelfPeerCmd(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	buf := new(bytes.Buffer)
	cmd := newPeerSelfCmd(ts.client)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	assert.NoError(t, err)

	buf2 := new(bytes.Buffer)
	fmt.Fprintln(buf2, "Host ID: abcdef12345")
	fmt.Fprintln(buf2, "ip4/10000, ip6/20000")

	assert.Equal(t, buf.String(), buf2.String())
}
