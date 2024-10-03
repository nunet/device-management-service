package hardware

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetRAM(t *testing.T) {
	t.Parallel()

	ram, err := GetRAM()
	require.NoError(t, err)
	require.Greater(t, ram.Size, uint64(0))
}
