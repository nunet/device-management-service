// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
)

func TestNewContractCmd(t *testing.T) {
	dmsCli := utils.NewTestCli()
	cmd := newContractCmd(dmsCli)

	assert.Equal(t, "contracts", cmd.Use)
	assert.Equal(t, "Interact with contracts", cmd.Short)
	assert.NotNil(t, cmd.Commands())
	assert.Len(t, cmd.Commands(), 1)
}

func TestNewContractListCmd(t *testing.T) {
	dmsCli := utils.NewTestCli()
	cmd := newContractListCmd(dmsCli)

	assert.Equal(t, "list", cmd.Use)
	assert.Equal(t, "List contracts", cmd.Short)
	assert.Len(t, cmd.Commands(), 2)
}

func TestNewContractListAlias(t *testing.T) {
	dmsCli := utils.NewTestCli()

	// Test incoming
	cmd := newContractListAlias(dmsCli, "incoming", "List contracts where this node is the provider", contracts.ContractRoleProvider)
	assert.Equal(t, "incoming [flags]", cmd.Use)
	assert.Equal(t, "List contracts where this node is the provider", cmd.Short)
	assert.Contains(t, cmd.Long, "nunet contract list incoming")

	// Test outgoing
	cmd = newContractListAlias(dmsCli, "outgoing", "List contracts where this node is the requestor", contracts.ContractRoleRequestor)
	assert.Equal(t, "outgoing [flags]", cmd.Use)
	assert.Equal(t, "List contracts where this node is the requestor", cmd.Short)
	assert.Contains(t, cmd.Long, "nunet contract list outgoing")
}
