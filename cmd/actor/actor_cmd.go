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
	"strings"
	"time"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/client"
	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/internal/config"
)

const (
	fnTimeout     = "timeout"
	fnExpiry      = "expiry"
	fnContextName = "context"
	fnDest        = "dest"

	bBroadcast = "broadcast"
	bInvoke    = "invoke"
	bSend      = "send"
)

func newActorCmdGroup(afs afero.Afero, cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cmd",
		Short: "Invoke a predefined behavior on an actor",
		Long: `Invoke a predefined behavior on an actor

Example:
 nunet actor cmd --context user /broadcast/hello

Adding the --dest flag will cause the behavior to be invoked on the specified actor.

For more information on behaviors, refer to cmd/actor/README.md`,
		ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveDefault
			}
			var completions []string
			for k := range registeredBehaviors {
				completions = append(completions, strings.Split(k, "/")[2])
			}
			return completions, cobra.ShellCompDirectiveNoFileComp
		},
		Run: func(cmd *cobra.Command, _ []string) {
			err := cmd.Help()
			if err != nil {
				cmd.Println(err)
			}
		},
	}

	for behavior := range registeredBehaviors {
		if behaviorCfg, ok := registeredBehaviors[behavior]; ok {
			cmd.AddCommand(newActorCmdCmd(afs, behavior, behaviorCfg, cfg))
		}
	}

	cmd.PersistentFlags().StringP(fnContextName, "c", "", "capability context name")
	cmd.PersistentFlags().DurationP(fnTimeout, "t", 0, "timeout duration")
	cmd.PersistentFlags().VarP(utils.NewTimeValue(&time.Time{}), fnExpiry, "e", "expiration time")
	cmd.PersistentFlags().StringP(fnDest, "d", "", "destination DMS DID, peer ID or handle")
	cmd.MarkFlagsMutuallyExclusive(fnTimeout, fnExpiry)
	return cmd
}

func newActorCmdCmd(afs afero.Afero, behavior string, behaviorCfg behaviorConfig, cfg *config.Config) *cobra.Command {
	payload := &Payload{val: nil}
	if behaviorCfg.Payload != nil {
		payload.val = behaviorCfg.Payload()
	}

	cmd := &cobra.Command{
		Use:               fmt.Sprintf("%s [<param> ...]", behavior),
		Short:             behaviorCfg.Short,
		Long:              behaviorCfg.Long,
		ValidArgsFunction: behaviorCfg.ValidArgsFn,
		Args:              behaviorCfg.Args,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			if behaviorCfg.PreRunE != nil {
				return behaviorCfg.PreRunE(cmd, payload.val)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			timeout, _ := cmd.Flags().GetDuration(fnTimeout)
			expiry, _ := utils.GetTime(cmd.Flags(), fnExpiry)
			contextName, _ := cmd.Flags().GetString(fnContextName)
			dest, _ := cmd.Flags().GetString(fnDest)

			// Create security context first
			sctx, err := utils.NewSecurityContext(afs, contextName, cfg)
			if err != nil {
				return fmt.Errorf("could not create security context: %w", err)
			}

			// Now call newClient with the correct arguments
			cli, err := utils.NewClient(cfg, sctx)
			if err != nil {
				return fmt.Errorf("could not create client: %w", err)
			}

			res, err := behaviorCfg.Run(
				cmd,
				cli,
				payload.val,
				client.WithTimeout(timeout),
				client.WithExpiry(expiry),
				client.WithDestination(dest),
			)
			if err != nil {
				return err
			}

			return displayResponse(cmd, res)
		},
	}

	if behaviorCfg.SetFlags != nil {
		behaviorCfg.SetFlags(cmd, payload.val)
	}

	return cmd
}
