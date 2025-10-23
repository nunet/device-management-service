// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package actor

import (
	"fmt"

	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/client"
	"gitlab.com/nunet/device-management-service/cmd/cli"
)

func newActorCmdGroup(dmsCli *cli.DmsCLI) *cobra.Command {
	// Create a slice of valid arguments from behavior map keys
	validArgs := make([]string, 0, len(registeredBehaviors))
	for behavior := range registeredBehaviors {
		validArgs = append(validArgs, behavior)
	}

	cmd := &cobra.Command{
		Use:   "cmd",
		Short: "Invoke a predefined behavior on an actor",
		Long: `Invoke a predefined behavior on an actor

Example:
 nunet actor cmd --context user /broadcast/hello

Adding the --dest flag will cause the behavior to be invoked on the specified actor.

For more information on behaviors, refer to cmd/actor/README.md`,
		ValidArgs: validArgs,
	}

	for behavior, behaviorCfg := range registeredBehaviors {
		cmd.AddCommand(newActorCmdCmd(dmsCli, behavior, behaviorCfg))
	}

	useMessageOptsFlags(cmd, true)
	return cmd
}

type actorCmdOptions struct {
	Context string
	Payload any
	Args    []string
	MsgOpts []client.Option
	Streams cli.Streams
}

func newActorCmdCmd(dmsCli *cli.DmsCLI, behavior string, behaviorCfg *behaviorConfig) *cobra.Command {
	opts := actorCmdOptions{}
	if behaviorCfg.Payload != nil {
		opts.Payload = behaviorCfg.Payload()
	}

	cmd := &cobra.Command{
		Use:               fmt.Sprintf("%s [<param> ...]", behavior),
		Short:             behaviorCfg.Short,
		Long:              behaviorCfg.Long,
		ValidArgsFunction: behaviorCfg.ValidArgsFn,
		Args:              behaviorCfg.Args,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.Args = args
			opts.Context, _ = cmd.Flags().GetString(fnContext)
			opts.MsgOpts = getBehaviorMsgOpts(cmd)
			opts.Streams = cli.CmdStreams(cmd)
			if behaviorCfg.PreRunFn != nil {
				return behaviorCfg.PreRunFn(cmd, dmsCli, opts)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return behaviorCfg.Run(cmd.Context(), dmsCli, opts, cli.CmdStreams(cmd))
		},
	}

	if behaviorCfg.SetFlags != nil {
		behaviorCfg.SetFlags(cmd, opts.Payload)
	}

	return cmd
}

// NewActorCmdWrapper is a factory for creating actor command aliases.
func NewActorCmdWrapper(dmsCli *cli.DmsCLI, behavior string) (*cobra.Command, error) {
	behaviorCfg, ok := registeredBehaviors[behavior]
	if !ok {
		return nil, fmt.Errorf("unknown behavior: %s", behavior)
	}
	cmd := newActorCmdCmd(dmsCli, behavior, behaviorCfg)
	useMessageOptsFlags(cmd, true)
	return cmd, nil
}
