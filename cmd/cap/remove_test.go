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
	"gitlab.com/nunet/device-management-service/cmd/utils"
)

func TestRemoveCmd_InvalidArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "no context",
			args: []string{},
		},
		{
			name: "no anchor type",
			args: []string{"--context", userCtx},
		},
		{
			name: "invalid did",
			args: []string{"--context", userCtx, "--root", "not-a-did"},
		},
		{
			name: "invalid provide token format",
			args: []string{"--context", userCtx, "--provide", "not-valid-json"},
		},
		{
			name: "invalid require token format",
			args: []string{"--context", userCtx, "--require", "not-valid-json"},
		},
	}

	dmsCli := newTestCli()

	// create test cap context
	_, _, err := utils.NewCapabilityContext(dmsCli, userCtx)
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := utils.ExecuteCommand(newRemoveCmd(dmsCli), tt.args...)
			assert.Error(t, err)
		})
	}
}

func TestRemoveCmd_PassphraseError(t *testing.T) {
	t.Parallel()
	dmsCli := newTestCli()

	_, _, err := utils.ExecuteCommand(newRemoveCmd(dmsCli),
		"--context", "invalid",
		"--root", "did:key:z6MkwQpRz8b7vJY4N1k2",
	)
	assert.Error(t, err)
}

func TestRemoveCmd_Success(t *testing.T) {
	t.Parallel()
	dmsCli := newTestCli()

	// create test cap context
	_, _, err := utils.NewCapabilityContext(dmsCli, "removectx")
	assert.NoError(t, err)

	// anchor
	_, _, err = utils.ExecuteCommand(newAnchorCmd(dmsCli),
		"--context", "removectx",
		"--root", "did:key:z6MkwQpRz8b7vJY4N1k2",
	)
	assert.NoError(t, err)

	// remove
	_, _, err = utils.ExecuteCommand(newRemoveCmd(dmsCli),
		"--context", "removectx",
		"--root", "did:key:z6MkwQpRz8b7vJY4N1k2",
	)
	assert.NoError(t, err)
}
