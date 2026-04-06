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

func TestNewRunCmd(t *testing.T) {
	dmsCli := utils.NewTestCli()
	cmd := newRunCmd(dmsCli)

	assert.Equal(t, "run", cmd.Use)
	assert.Equal(t, "Start the Device Management Service", cmd.Short)
	assert.Contains(t, cmd.Long, "Start the Device Management Service")
	assert.Contains(t, cmd.Long, "nunet config --help")
	assert.NotNil(t, cmd.RunE)
	// Check flags
	contextFlag := cmd.Flags().Lookup("context")
	assert.NotNil(t, contextFlag)
	assert.Equal(t, "c", contextFlag.Shorthand)
	prismFlag := cmd.Flags().Lookup("prism-url")
	assert.NotNil(t, prismFlag)
}
