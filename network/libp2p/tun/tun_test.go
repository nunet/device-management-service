package tun

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJoinedNetworks(t *testing.T) {
	addrs, err := JoinedNetworks()
	require.NoError(t, err)

	for _, addr := range addrs {
		fmt.Println(addr)
		assert.True(t, strings.Contains(addr, "/"))
		assert.True(t, strings.Contains(addr, "."))
		assert.True(t, !strings.Contains(addr, ":"))
	}
}
