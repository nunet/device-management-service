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
)

func TestNewGetCmd(t *testing.T) {
	dmsCli := utils.NewTestCli()
	cmd := newGetCmd(dmsCli)

	assert.Equal(t, "get", cmd.Use)
	assert.Equal(t, "Get deployments, allocations etc.", cmd.Short)
	assert.Len(t, cmd.Commands(), 2)
}

func TestNewGetDeployments(t *testing.T) {
	dmsCli := utils.NewTestCli()
	cmd := newGetDeployments(dmsCli)

	assert.Equal(t, "deployments", cmd.Use)
	assert.Equal(t, "Get all deployments", cmd.Short)
	assert.Contains(t, cmd.Long, "Get all deployments")
	assert.Contains(t, cmd.Long, "ensemble ID")
}

func TestNewGetAllocations(t *testing.T) {
	dmsCli := utils.NewTestCli()
	cmd := newGetAllocations(dmsCli)

	assert.Equal(t, "allocations", cmd.Use)
	assert.Contains(t, cmd.Long, "Get all allocations")
	assert.Contains(t, cmd.Long, "Compute Provider")
}
