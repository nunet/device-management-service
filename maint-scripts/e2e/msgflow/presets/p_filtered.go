package presets

import (
	"gitlab.com/nunet/device-management-service/maint-scripts/e2e/msgflow/types"
)

func init() {
	register("filtered", nil, FilteredArgs)
}

// FilteredArgs filters out self, reply and hello messages.
func FilteredArgs(args types.Args) types.Args {
	args.SelfMsgs = false
	args.ReplyToMsgs = false
	args.HelloMsgs = false

	return args
}
