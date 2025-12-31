package presets

import (
	"gitlab.com/nunet/device-management-service/maint-scripts/e2e/logs/types"
)

func init() {
	register("folded", nil, FoldedArgs)
}

// FoldedArgs hides all the fields.
func FoldedArgs(args types.Args) types.Args {
	args.NoFields = true

	return args
}
