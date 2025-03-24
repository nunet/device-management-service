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
	"time"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/client"
	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/internal/config"
)

func newActorMsgCmd(afs afero.Afero, cfg *config.Config) *cobra.Command {
	fnDest := "dest"
	fnBroadcast := "broadcast"
	fnTimeout := "timeout"
	fnExpiry := "expiry"
	fnInvoke := "invoke"
	fnContextName := "context"

	cmd := &cobra.Command{
		Use:   "msg <behavior> <payload>",
		Short: "Construct a message",
		Long: `Construct and sign a message that can be communicated to an actor.

The constructed message is returned as a JSON object that can be used stored or piped into another command, for instance the the send, invoke, or broadcast command.

Example:
  nunet actor msg --broadcast /nunet/hello /broadcast/hello 'Hello, World!'`,

		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			destStr, _ := cmd.Flags().GetString(fnDest)
			topic, _ := cmd.Flags().GetString(fnBroadcast)
			timeout, _ := cmd.Flags().GetDuration(fnTimeout)
			expiry, _ := utils.GetTime(cmd.Flags(), fnExpiry)
			invocation, _ := cmd.Flags().GetBool(fnInvoke)
			contextName, _ := cmd.Flags().GetString(fnContextName)

			behavior := args[0]
			payload := args[1]

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

			msg, err := cli.NewActorMessage(cmd.Context(), behavior, payload, client.MessageOptions{
				Destination:  destStr,
				Topic:        topic,
				Timeout:      timeout,
				Expiry:       expiry,
				IsInvocation: invocation,
			})
			if err != nil {
				return fmt.Errorf("could not create message: %w", err)
			}

			msgData, err := json.Marshal(msg)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(msgData))

			return nil
		},
	}
	cmd.Flags().StringP(fnDest, "d", "", "destination handle")
	cmd.Flags().StringP(fnBroadcast, "b", "", "broadcast topic")
	cmd.Flags().BoolP(fnInvoke, "i", false, "construct an invocation")
	cmd.Flags().StringP(fnContextName, "c", "", "capability context name")
	cmd.Flags().DurationP(fnTimeout, "t", 0, "timeout duration")
	cmd.Flags().VarP(utils.NewTimeValue(&time.Time{}), fnExpiry, "e", "expiration time")

	cmd.MarkFlagsMutuallyExclusive(fnDest, fnBroadcast)
	cmd.MarkFlagsMutuallyExclusive(fnInvoke, fnBroadcast)
	cmd.MarkFlagsMutuallyExclusive(fnTimeout, fnExpiry)
	return cmd
}
