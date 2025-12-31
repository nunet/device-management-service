// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package main

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/alexflint/go-arg"

	shared "gitlab.com/nunet/device-management-service/maint-scripts/e2e"
	"gitlab.com/nunet/device-management-service/maint-scripts/e2e/logs/presets"
	"gitlab.com/nunet/device-management-service/maint-scripts/e2e/logs/types"
)

func init() {
	types.ArgsPresets = slices.Collect(maps.Keys(presets.Presets))
}

var args types.Args

func main() {
	p := arg.MustParse(&args)
	var output string

	// intro screen
	if args.SourceName == "" {
		p.WriteHelp(os.Stdout)
		os.Exit(0)
	}

	// validate presets
	for _, preset := range args.Preset {
		if _, ok := presets.Presets[preset]; !ok {
			continue
		}
		if _, ok := presets.PresetsArgs[preset]; !ok {
			continue
		}
		p.Fail(fmt.Sprintf("unknown preset: %s", preset))
	}

	// handle args presets
	for _, preset := range args.Preset {
		if _, ok := presets.PresetsArgs[preset]; ok {
			args = presets.PresetsArgs[preset](args)
		}
	}

	// collect and process log files
	logs := shared.CollectLogFiles(args.ArgsBasic, args.NodeName)
	processed := make([][]*shared.LogLine, len(logs))
	for i, logFile := range logs {
		// collect
		lines, did, err := shared.CollectLines(logFile, args.ArgsAdjacent, args.ArgsFilters, args.Flightrec)
		if err != nil {
			p.Fail(fmt.Sprintf("collecting lines for %s: %s", logFile.Name, err.Error()))
		}
		if len(lines) == 0 {
			continue
		}
		logs[i].DID = did
		processed[i] = shared.ParseLines(lines)
	}

	// handle presets
	for _, preset := range args.Preset {
		if _, ok := presets.Presets[preset]; ok {
			logs, processed = presets.Presets[preset](args, logs, processed)
		}
	}

	// render stdout
	if args.Headers && args.HeadersNetwork {
		output += shared.RenderLogHeader(logs)
	}
	shown := 0
	for i, lines := range processed {
		if len(lines) == 0 {
			continue
		}
		logFile := logs[i]

		// render
		if args.Headers {
			output += shared.RenderSliceHeader(logFile, shown > 0)
		}
		if sliceOut, err := shared.RenderSlice("logs-"+logFile.Name, lines, args.ArgsBasic); err != nil {
			p.Fail("rendering tmp file: " + err.Error())
		} else {
			output += sliceOut
			shown++
		}
	}

	// save HTML
	if wd, err := os.Getwd(); err == nil && args.OutputHTML != "" {
		err := shared.SaveHTML(filepath.Join(wd, args.OutputHTML), output, true)
		if err != nil {
			p.Fail(err.Error())
		}
	}
}
