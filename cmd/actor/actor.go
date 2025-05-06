// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package actor

import (
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/env"
)

// NewActorCmd is a constructor for `actor` parent command
func NewActorCmd(
	afs afero.Afero, env env.EnvironmentProvider,
	cfg *config.Config,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "actor",
		Short: "Interact with the actor system",
		Long: `Interact with the actor system

Actors are the entities which compose the NuActor system, a secure decentralized programming framework based on the Actor Model.
Actors are connected through the libp2p network substrate and communication is achieved via immutable messages.

For more information on the actor system, please refer to actor/README.md`,
	}
	cmd.AddCommand(newActorMsgCmd(afs, env, cfg))
	cmd.AddCommand(newActorSendCmd(afs, cfg))
	cmd.AddCommand(newActorInvokeCmd(afs, cfg))
	cmd.AddCommand(newActorBroadcastCmd(afs, cfg))
	cmd.AddCommand(newActorCmdGroup(afs, env, cfg))
	return cmd
}
