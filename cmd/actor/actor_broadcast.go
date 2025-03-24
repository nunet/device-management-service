// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package actor

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/internal/config"
)

func newActorBroadcastCmd(_ afero.Afero, cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "broadcast <msg>",
		Short: "Broadcast a message",
		Long: `Broadcast a message to a topic

If a topic is specified in the message's payload, the message will be published to all subscribers of that topic.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var msg actor.Envelope

			if err := json.Unmarshal([]byte(args[0]), &msg); err != nil {
				return fmt.Errorf("could not unmarshal message: %w", err)
			}

			cli, err := utils.NewClient(cfg, nil)
			if err != nil {
				return fmt.Errorf("could not create client: %w", err)
			}

			res, err := cli.BroadcastMessageRaw(cmd.Context(), msg)
			if err != nil {
				return fmt.Errorf("could not broadcast message: %w", err)
			}

			for _, r := range res {
				if err := displayResponse(cmd, r); err != nil {
					return err
				}
			}
			return nil
		},
	}
	return cmd
}
