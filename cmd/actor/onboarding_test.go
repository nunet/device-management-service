// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package actor

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/cmd/cli"
	"gitlab.com/nunet/device-management-service/types"
)

func TestCommandLineGPUOnboarding(t *testing.T) {
	machineResources := types.Resources{
		GPUs: []types.GPU{
			{Index: 0, Model: "GPU0"},
			{Index: 1, Model: "GPU1"},
		},
	}

	tests := []struct {
		name    string
		gpuArgs string
		wantErr bool
		wantLen int
	}{
		{
			name:    "valid single GPU",
			gpuArgs: "0:100",
			wantErr: false,
			wantLen: 1,
		},
		{
			name:    "valid multiple GPUs",
			gpuArgs: "0:50,1:75",
			wantErr: false,
			wantLen: 2,
		},
		{
			name:    "invalid format",
			gpuArgs: "0",
			wantErr: true,
			wantLen: 0,
		},
		{
			name:    "invalid index",
			gpuArgs: "a:100",
			wantErr: true,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gpus, err := commandLineGPUOnboarding(machineResources, tt.gpuArgs, cli.Streams{})
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, gpus, tt.wantLen)
			}
		})
	}
}
