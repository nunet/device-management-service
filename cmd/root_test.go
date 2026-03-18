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

func TestNewRootCMD(t *testing.T) {
	dmsCli := utils.NewTestCli()
	cmd := NewRootCMD(dmsCli)

	assert.Equal(t, "nunet", cmd.Use)
	assert.Equal(t, "NuNet Device Management Service", cmd.Short)
	assert.Equal(t, "The Device Management Service (DMS) Command Line Interface (CLI)", cmd.Long)
	assert.True(t, cmd.SilenceErrors)
	assert.True(t, cmd.SilenceUsage)
	assert.NotNil(t, cmd.Run)
	assert.NotNil(t, cmd.PersistentPreRun)
	// Check that subcommands are added
	assert.True(t, len(cmd.Commands()) > 0)
}
