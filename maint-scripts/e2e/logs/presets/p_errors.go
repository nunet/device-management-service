package presets

import (
	"fmt"
	"os"
	"slices"

	shared "gitlab.com/nunet/device-management-service/maint-scripts/e2e"
	"gitlab.com/nunet/device-management-service/maint-scripts/e2e/logs/types"
)

var errorsArgs ErrorsArgs

func init() {
	register("errors", Errors, nil)
}

type ErrorsArgs struct {
	ErrorsIncInfo  bool `help:"Include INFO level msgs containing an 'error' prop" default:"false" arg:"--errors-inc-info"`
	ErrorsIncDebug bool `help:"Include DEBUG level msgs containing an 'error' prop" default:"false" arg:"--errors-inc-debug"`
	ErrorsIncWarn  bool `help:"Include WARN level msgs containing an 'error' prop" default:"false" arg:"--errors-inc-warn"`
	ErrorsIncError bool `help:"Include ERROR level msgs containing an 'error' prop" default:"true" arg:"--errors-inc-error"`
}

func Errors(args types.Args, files []shared.LogFile, logs [][]*shared.LogLine) (
	[]shared.LogFile, [][]*shared.LogLine,
) {
	// parse args
	parsePresetArgs(args, "errors", &errorsArgs)

	// preset logic
	var err error
	for i, lines := range logs {
		// filter out levels
		if !errorsArgs.ErrorsIncInfo || !errorsArgs.ErrorsIncDebug || !errorsArgs.ErrorsIncError ||
			!errorsArgs.ErrorsIncWarn {

			lines = slices.DeleteFunc(lines, func(line *shared.LogLine) bool {
				if !errorsArgs.ErrorsIncInfo && line.Level == "INFO" {
					return true
				}
				if !errorsArgs.ErrorsIncDebug && line.Level == "DEBUG" {
					return true
				}
				if !errorsArgs.ErrorsIncError && line.Level == "ERROR" {
					return true
				}
				if !errorsArgs.ErrorsIncWarn && line.Level == "WARN" {
					return true
				}

				return false
			})
		}

		logs[i], err = shared.JSONQuery(lines, ".error")
		if err != nil {
			fmt.Printf("error: %v\n", err)
			os.Exit(1)
		}
	}

	// return data
	return files, logs
}
