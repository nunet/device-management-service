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
)

func TestNewGPUCommand(t *testing.T) {
	cmd := newGPUCommand()

	assert.Equal(t, "gpu <operation>", cmd.Use)
	assert.Equal(t, "Manage GPU resources", cmd.Short)
	assert.Contains(t, cmd.Long, "list: List all available GPUs")
	assert.Len(t, cmd.Commands(), 2)
}

func TestNewGPUListCommand(t *testing.T) {
	cmd := newGPUListCommand()

	assert.Equal(t, "list", cmd.Use)
	assert.Equal(t, "List all available GPUs", cmd.Short)
	assert.NotNil(t, cmd.RunE)
}

func TestNewGPUTestCommand(t *testing.T) {
	cmd := newGPUTestCommand()

	assert.Equal(t, "test", cmd.Use)
	assert.Equal(t, "Test GPU deployment by running a Docker container with GPU resources", cmd.Short)
	assert.NotNil(t, cmd.RunE)
}
