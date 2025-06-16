// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package actor

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/cmd/cli"
)

// newActorSendCmd is a constructor for `actor send` subcommand
func newActorSendCmd(dmsCli *cli.DmsCLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send <msg>",
		Short: "Send a message",
		Long: `Send a message to an actor

Actors only communicate via messages. For more information on constructing a message, see:

  nunet actor msg --help

The message is encoded into an actor envelope, which then is sent across the network through the API.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runActorSendCmd(
				cmd.Context(),
				dmsCli,
				args[0],
				cli.CmdStreams(cmd),
			)
		},
	}
	return cmd
}

// runActorSendCmd is the testable core logic for the send command
func runActorSendCmd(
	ctx context.Context,
	dmsCli *cli.DmsCLI,
	msgArg string,
	streams cli.Streams,
) error {
	var msg actor.Envelope

	if err := json.Unmarshal([]byte(msgArg), &msg); err != nil {
		return fmt.Errorf("could not unmarshal message: %w", err)
	}

	client, err := dmsCli.NewClient(nil)
	if err != nil {
		return fmt.Errorf("could not create client: %w", err)
	}

	res, err := client.SendMessageRaw(ctx, msg)
	if err != nil {
		return fmt.Errorf("could not send message: %w", err)
	}

	return displayResponse(streams.Out, json.RawMessage(res.Message))
}
