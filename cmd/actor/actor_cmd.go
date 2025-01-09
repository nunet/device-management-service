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
	"strings"
	"time"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/internal/config"
	dmsUtil "gitlab.com/nunet/device-management-service/utils"
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

func newActorCmdGroup(client *dmsUtil.HTTPClient, afs afero.Afero, cfg *config.Config) *cobra.Command {
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
			cmd.AddCommand(newActorCmdCmd(client, afs, behavior, behaviorCfg, cfg))
		}
	}

	cmd.PersistentFlags().StringP(fnContextName, "c", "", "capability context name")
	cmd.PersistentFlags().DurationP(fnTimeout, "t", 0, "timeout duration")
	cmd.PersistentFlags().VarP(utils.NewTimeValue(&time.Time{}), fnExpiry, "e", "expiration time")
	cmd.PersistentFlags().StringP(fnDest, "d", "", "destination DMS DID, peer ID or handle")
	cmd.MarkFlagsMutuallyExclusive(fnTimeout, fnExpiry)
	return cmd
}

func newActorCmdCmd(client *dmsUtil.HTTPClient, afs afero.Afero, behavior string, behaviorCfg behaviorConfig, cfg *config.Config) *cobra.Command {
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

			dmsHandle, err := getDMSHandle(client)
			if err != nil {
				return fmt.Errorf("could not get source DMS handle: %w", err)
			}

			topic := ""
			if behaviorCfg.Type == bBroadcast {
				topic = behaviorCfg.Topic
			}

			if behaviorCfg.PayloadEnc != nil {
				payload.val, err = behaviorCfg.PayloadEnc(payload.val)
				if err != nil {
					return fmt.Errorf("could not marshal payload: %w", err)
				}
			}

			invocation := behaviorCfg.Type == bInvoke

			msg, err := newActorMessage(afs, dmsHandle, dest, topic, behavior, payload.val, timeout, expiry, invocation, contextName, cfg)
			if err != nil {
				return fmt.Errorf("could not create message: %w", err)
			}

			msgData, err := json.Marshal(msg)
			if err != nil {
				return fmt.Errorf("could not marshal message: %w", err)
			}

			endpoint := fmt.Sprintf("/actor/%s", behaviorCfg.Type)
			resBody, resCode, err := client.MakeRequest(cmd.Context(), "POST", endpoint, msgData)
			if err != nil {
				return fmt.Errorf("unable to make internal request: %w", err)
			}
			if resCode != 200 {
				return fmt.Errorf("request failed with status code: %d", resCode)
			}

			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "    ")

			if behaviorCfg.Type == bBroadcast {
				var resMsgs []cmdResponse
				if err := json.Unmarshal(resBody, &resMsgs); err != nil {
					return fmt.Errorf("could not unmarshal response: %w", err)
				}
				return enc.Encode(resMsgs)
			}
			var resMsg cmdResponse
			if err := json.Unmarshal(resBody, &resMsg); err != nil {
				return nil
			}
			return enc.Encode(resMsg)
		},
	}

	if behaviorCfg.SetFlags != nil {
		behaviorCfg.SetFlags(cmd, payload.val)
	}

	return cmd
}
