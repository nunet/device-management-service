package presets

import (
	"gitlab.com/nunet/device-management-service/maint-scripts/e2e/logs/types"
)

func init() {
	register("expanded", nil, ExpandedArgs)
}

// ExpandedArgs shows all the fields.
func ExpandedArgs(args types.Args) types.Args {
	args.AllFields = true

	return args
}
