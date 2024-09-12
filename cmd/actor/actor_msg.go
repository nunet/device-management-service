package actor

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/cmd/utils"
	dmsUtils "gitlab.com/nunet/device-management-service/utils"
)

// NewActorMsgCmd is a constructor for `actor msg` subcommand
func newActorMsgCmd(client *dmsUtils.HTTPClient, afs afero.Afero) *cobra.Command {
	fnDest := "dest"
	fnBroadcast := "broadcast"
	fnTimeout := "timeout"
	fnExpiry := "expiry"
	fnInvoke := "invoke"
	fnContextName := "context"

	cmd := &cobra.Command{
		Use:   "msg <behavior> <payload>",
		Short: "Construct message for actor",
		RunE: func(cmd *cobra.Command, args []string) error {
			destStr, _ := cmd.Flags().GetString(fnDest)
			topic, _ := cmd.Flags().GetString(fnBroadcast)
			timeout, _ := cmd.Flags().GetDuration(fnTimeout)
			expiry, _ := utils.GetTime(cmd.Flags(), fnExpiry)
			invocation, _ := cmd.Flags().GetBool(fnInvoke)
			contextName, _ := cmd.Flags().GetString(fnContextName)

			behavior := args[0]
			payload := args[1]

			dmsHandle, err := getDMSHandle(client)
			if err != nil {
				return fmt.Errorf("could not get source handle: %w", err)
			}

			msg, err := newActorMessage(afs, dmsHandle, destStr, topic, behavior, payload, timeout, expiry, invocation, contextName)
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
		Args: cobra.ExactArgs(2),
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
