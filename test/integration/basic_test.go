package itest

import (
	"encoding/json"

	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/lib/hardware"
	"gitlab.com/nunet/device-management-service/types"
)

func BasicTests(suite *TestSuite) {
	for _, node := range suite.nodes {
		// Test 1 - setup onboarded 10GiB of Disk - test if the unit is correctly parsed
		// setup onboarded with 40% of existing RAM as GB and 10GB of Disk as GiB
		// test if the units are correctly parsed
		hw := hardware.NewHardwareManager()
		mr, err := hw.GetMachineResources()
		require.NoError(suite.T(), err)
		assert := suite.Assert()

		resp := node.client.getOnboarded(suite.T(), node.userContext, node.password)
		onboardedResp := struct {
			OK        bool
			Resources types.Resources
		}{}
		err = json.Unmarshal([]byte(resp), &onboardedResp)
		require.NoError(suite.T(), err)

		expectedDisk := uint64(10 * 1073741824) // GiB
		assert.Equal(expectedDisk, onboardedResp.Resources.Disk.Size)

		expectedRAMBytes := float64(mr.RAM.Size) * 0.4 // 40% of available
		expectedRAMGB := uint64(expectedRAMBytes / 1e9)
		assert.EqualValues(expectedRAMGB, onboardedResp.Resources.RAM.SizeInGB())

		// Test 2 - every node in the network must be able to broadcast and receive DID from all other nodes
		// every node in the network must be able to broadcast and receive DID from all other nodes
		result := node.client.broadcast(suite.T(), node.userContext, node.password)
		suite.Equal(suite.numNodes, countDIDOccurrences(result))
	}
}
