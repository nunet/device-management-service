package presets

import (
	"gitlab.com/nunet/device-management-service/maint-scripts/e2e/logs/types"
)

func init() {
	register("map", nil, MapArgs)
}

// MapArgs hides all the fields.
func MapArgs(args types.Args) types.Args {
	args.Max = -1

	return args
}
