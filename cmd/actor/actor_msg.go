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
	"fmt"

	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/client"
	"gitlab.com/nunet/device-management-service/cmd/cli"
	"gitlab.com/nunet/device-management-service/cmd/utils"
)

type actorMsgOptions struct {
	Context  string
	Behavior string
	Payload  string
	MsgOpts  client.MessageOptions
}

func newActorMsgCmd(dmsCli *cli.DmsCLI) *cobra.Command {
	var opts actorMsgOptions

	cmd := &cobra.Command{
		Use:   "msg <behavior> <payload>",
		Short: "Construct a message",
		Long: `Construct and sign a message that can be communicated to an actor.

The constructed message is returned as a JSON object that can be used stored or piped into another command, for instance the the send, invoke, or broadcast command.

Example:
  nunet actor msg --broadcast /nunet/hello /broadcast/hello 'Hello, World!'`,

		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Context, _ = cmd.Flags().GetString(fnContext)

			opts.Behavior = args[0]
			opts.Payload = args[1]

			for _, opt := range getNewMsgOpts(cmd) {
				opt(&opts.MsgOpts)
			}

			return runActorMsgCmd(cmd.Context(), dmsCli, opts, cli.CmdStreams(cmd))
		},
	}

	useNewMsgOptsFlags(cmd, false)

	return cmd
}

func runActorMsgCmd(ctx context.Context, dmsCli *cli.DmsCLI, opts actorMsgOptions, streams cli.Streams) error {
	sctx, err := utils.NewSecurityContext(dmsCli, opts.Context)
	if err != nil {
		return fmt.Errorf("could not create security context: %w", err)
	}

	cli, err := dmsCli.NewClient(sctx)
	if err != nil {
		return fmt.Errorf("could not create client: %w", err)
	}

	msg, err := cli.NewActorMessage(ctx, opts.Behavior, opts.Payload, opts.MsgOpts)
	if err != nil {
		return fmt.Errorf("could not create message: %w", err)
	}

	return displayResponse(streams.Out, msg)
}
