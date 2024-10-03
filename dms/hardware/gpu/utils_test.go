package gpu

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/types"
)

func TestAssignIndexToGPUs(t *testing.T) {
	gpus := []types.GPU{
		{
			Vendor: types.GPUVendorNvidia,
		},
		{
			Vendor: types.GPUVendorAMDATI,
		},
		{
			Vendor: types.GPUVendorIntel,
		},
	}

	assignIndexToGPUs(gpus)
	require.True(t, gpus[0].Index == 0)
	require.True(t, gpus[0].Vendor == types.GPUVendorNvidia)

	require.True(t, gpus[1].Index == 1)
	require.True(t, gpus[1].Vendor == types.GPUVendorAMDATI)

	require.True(t, gpus[2].Index == 2)
	require.True(t, gpus[2].Vendor == types.GPUVendorIntel)
}
