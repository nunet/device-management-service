// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package e2e

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"

	"gitlab.com/nunet/device-management-service/lib/hardware"
	"gitlab.com/nunet/device-management-service/types"
)

// envE2ERAMGB overrides RAM required by E2E tests.
const envE2ERAMGB = "DMS_E2E_RAM_GB"

func BasicTests(suite *TestSuite) {
	for _, node := range suite.nodes {
		// Test 1 - setup onboarded 10GiB of Disk - test if the unit is correctly parsed
		// setup onboarded with 40% of existing RAM as GB and 10GB of Disk as GiB
		// test if the units are correctly parsed
		require := suite.Require()

		resp := node.client.getOnboarded(suite.T(), node.userContext, node.password)
		onboardedResp := struct {
			OK        bool
			Resources types.Resources
		}{}
		err := json.Unmarshal([]byte(resp), &onboardedResp)
		require.NoError(err)

		expectedDisk := uint64(10 * types.GB)
		require.Equal(expectedDisk, onboardedResp.Resources.Disk.Size, "disk size mismatch")

		require.EqualValues(expectedRAMGB(suite.T()), onboardedResp.Resources.RAM.SizeInGB(), "RAM size mismatch")

		// Test 2 - every node in the network must be able to broadcast and receive DID from all other nodes
		// every node in the network must be able to broadcast and receive DID from all other nodes
		result := node.client.broadcast(suite.T(), node.userContext, node.password)
		suite.Equal(suite.numNodes, countDIDOccurrences(result))
	}
}

func expectedRAMGB(t *testing.T) int {
	mr, err := hardware.NewHardwareManager().GetMachineResources()
	if err != nil {
		t.Fatal(err)
	}

	expected := float64(mr.RAM.Size) * 0.4 / types.GB
	if ram := os.Getenv(envE2ERAMGB); ram != "" {
		if val, err := strconv.Atoi(ram); err == nil {
			expected = float64(val)
		}
	}

	return int(expected)
}
