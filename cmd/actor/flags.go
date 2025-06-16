package actor

import (
	"time"

	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/client"
	"gitlab.com/nunet/device-management-service/cmd/utils"
)

const (
	fnContext   = "context"
	fnDest      = "dest"
	fnTimeout   = "timeout"
	fnExpiry    = "expiry"
	fnBroadcast = "broadcast"
	fnInvoke    = "invoke"
)

func useMessageOptsFlags(cmd *cobra.Command, persistent bool) {
	flagSet := cmd.Flags()
	if persistent {
		flagSet = cmd.PersistentFlags()
	}
	flagSet.StringP(fnContext, "c", "", "capability context name")
	flagSet.StringP(fnDest, "d", "", "destination DMS DID, peer ID or handle")
	flagSet.DurationP(fnTimeout, "t", 0, "timeout duration")
	flagSet.VarP(utils.NewTimeValue(&time.Time{}), fnExpiry, "e", "expiration time")
	cmd.MarkFlagsMutuallyExclusive(fnTimeout, fnExpiry)
}

func useNewMsgOptsFlags(cmd *cobra.Command, persistent bool) {
	useMessageOptsFlags(cmd, persistent)
	flagSet := cmd.Flags()
	if persistent {
		flagSet = cmd.PersistentFlags()
	}
	flagSet.StringP(fnBroadcast, "b", "", "broadcast topic")
	flagSet.BoolP(fnInvoke, "i", false, "construct an invocation")
	cmd.MarkFlagsMutuallyExclusive(fnDest, fnBroadcast)
	cmd.MarkFlagsMutuallyExclusive(fnInvoke, fnBroadcast)
}

func getBehaviorMsgOpts(cmd *cobra.Command) []client.Option {
	var opts []client.Option
	if dest, err := cmd.Flags().GetString(fnDest); err == nil {
		opts = append(opts, client.WithDestination(dest))
	}
	if timeout, err := cmd.Flags().GetDuration(fnTimeout); err == nil {
		opts = append(opts, client.WithTimeout(timeout))
	}
	if expiry, err := utils.GetTime(cmd.Flags(), fnExpiry); err == nil {
		opts = append(opts, client.WithExpiry(expiry))
	}
	return opts
}

func getNewMsgOpts(cmd *cobra.Command) []client.Option {
	opts := getBehaviorMsgOpts(cmd)
	if topic, err := cmd.Flags().GetString(fnBroadcast); err == nil {
		opts = append(opts, client.WithTopic(topic))
	}
	if invocation, err := cmd.Flags().GetBool(fnInvoke); err == nil {
		opts = append(opts, client.WithInvocation(invocation))
	}
	return opts
}
