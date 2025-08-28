// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package actor

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/client"
	"gitlab.com/nunet/device-management-service/cmd/cli"
	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/crypto/keystore"
)

func TestMain(m *testing.M) {
	// Use fast scrypt params for tests
	keystore.SetTestScryptParams(1<<10, 1, 1) // N=1024, R=1, P=1 (significantly faster)

	code := m.Run()

	// Reset scrypt params to defaults after tests
	keystore.ResetScryptParamsToDefaults()

	os.Exit(code)
}

func setupTest(t *testing.T, c client.DmsClient) *cli.DmsCLI {
	t.Helper()

	dmsCli := utils.NewTestCli(cli.WithClientFn(
		func(_ *config.Config, _ actor.SecurityContext) (client.DmsClient, error) {
			return c, nil
		},
	))

	_, _, err := utils.NewCapabilityContext(dmsCli, utils.DefaultUserContextName)
	require.NoError(t, err)

	return dmsCli
}

func checkMessageOptions(t *testing.T, expected client.MessageOptions, opts ...client.Option) {
	t.Helper()
	var msgOpts client.MessageOptions
	for _, opt := range opts {
		opt(&msgOpts)
	}
	assert.Equal(t, expected, msgOpts)
}

func TestActorCmd(t *testing.T) {
	t.Parallel()

	dmsCli := utils.NewTestCli()
	cmd := NewActorCmd(dmsCli)
	_, _, err := utils.ExecuteCommand(cmd)
	assert.NoError(t, err)
}
