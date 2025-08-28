// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/cmd/cli"
	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/lib/env"
)

func TestNewCmd_MissingArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name:          "no args",
			args:          []string{},
			expectedError: "accepts 1 arg(s), received 0",
		},
	}

	cli := newTestCli()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := utils.ExecuteCommand(newNewCmd(cli), tt.args...)
			assert.ErrorContains(t, err, tt.expectedError)
		})
	}
}

func TestNewCmd_PassphraseError(t *testing.T) {
	t.Parallel()
	cli := newTestCli(cli.WithEnv(env.NewMockEnvironment()))

	_, _, err := utils.ExecuteCommand(newNewCmd(cli), "invalid")
	assert.Error(t, err)
}

func TestNewCmd_Success(t *testing.T) {
	t.Parallel()
	cli := newTestCli()

	_, _, err := utils.ExecuteCommand(newNewCmd(cli), "newctx")
	assert.NoError(t, err)
}
