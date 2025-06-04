// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cap

import (
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/cmd/cli"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/env"
)

const (
	fnContext    = "context"
	fnAudience   = "audience"
	fnAction     = "action"
	fnCap        = "cap"
	fnTopic      = "topic"
	fnExpiry     = "expiry"
	fnDuration   = "duration"
	fnAutoExpire = "auto-expire"
	fnSelfSign   = "self-sign"
	fnDepth      = "depth"
	fnProvide    = "provide"
	fnRevoke     = "revoke"
	fnRoot       = "root"
	fnRequire    = "require"
	fnForce      = "force"
	fnToken      = "token"
)

// NewCapCmd returns the cap command that adds other commands
func NewCapCmd(
	afs afero.Afero, env env.EnvironmentProvider,
	cfg *config.Config,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cap",
		Short: "Manage capabilities",
		Long:  `Manage capabilities for the Device Management Service`,
	}

	dmsCli := cli.New(cli.WithConfig(cfg), cli.WithEnv(env), cli.WithFS(afs))

	cmd.AddCommand(newGrantCmd(dmsCli))
	cmd.AddCommand(newAnchorCmd(dmsCli))
	cmd.AddCommand(newRevokeCmd(dmsCli))
	cmd.AddCommand(newNewCmd(dmsCli))
	cmd.AddCommand(newDelegateCmd(dmsCli))
	cmd.AddCommand(newListCmd(dmsCli))
	cmd.AddCommand(newRemoveCmd(dmsCli))
	cmd.AddCommand(newHelpCmd())

	return cmd
}
