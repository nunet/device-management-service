package presets

import (
	"slices"

	"gitlab.com/nunet/device-management-service/maint-scripts/e2e/logs/types"
)

func init() {
	register("errors-all", nil, ErrorsAll)
}

// ErrorsAll shows all errors via `--preset errors`.
func ErrorsAll(args types.Args) types.Args {
	args.PresetArgs += "--errors-inc-info --errors-inc-debug --errors-inc-warn"
	if !slices.Contains(args.Preset, "errors") {
		args.Preset = append(args.Preset, "errors")
	}

	return args
}
