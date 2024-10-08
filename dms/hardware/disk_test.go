package hardware

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetDisk(t *testing.T) {
	disk, err := GetDisk()
	require.NoError(t, err)
	require.Greater(t, disk.Size, uint64(0))
}

func TestGetDiskUsage(t *testing.T) {
	disk, err := GetDiskUsage()
	require.NoError(t, err)
	require.Greater(t, disk.Size, uint64(0))
}
