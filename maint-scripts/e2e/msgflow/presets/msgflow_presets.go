package presets

import (
	"fmt"
	"os"
	"strings"

	"github.com/alexflint/go-arg"

	shared "gitlab.com/nunet/device-management-service/maint-scripts/e2e"
	"gitlab.com/nunet/device-management-service/maint-scripts/e2e/msgflow/types"
)

type PresetFunc func(
	args types.Args, files []shared.LogFile, logs []*shared.LogLine,
) ([]shared.LogFile, []*shared.LogLine)

type PresetArgsFunc func(args types.Args) types.Args

var Presets = make(map[string]PresetFunc)

var PresetsArgs = make(map[string]PresetArgsFunc)

// register registers max 2 and min 1 preset functions. After that, the preset is available as `-p name`.
func register(name string, fnPreset PresetFunc, fnArgs PresetArgsFunc) {
	if fnPreset != nil {
		Presets[name] = fnPreset
	}
	if fnArgs != nil {
		PresetsArgs[name] = fnArgs
	}
}

func parsePresetArgs(args types.Args, preset string, target any) {
	p, err := arg.NewParser(arg.Config{}, target)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
	flags := []string{}
	for _, f := range strings.Split(args.PresetArgs, " ") {
		if !strings.HasPrefix(f, "--"+preset) {
			continue
		}
		flags = append(flags, f)
	}
	err = p.Parse(flags)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
	if args.PresetHelp {
		p.WriteHelp(os.Stdout)
		os.Exit(1)
	}
}

// BUILT-IN PRESETS
