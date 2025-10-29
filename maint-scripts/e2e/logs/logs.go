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
	"os"
	"path/filepath"

	"github.com/alexflint/go-arg"
	shared "gitlab.com/nunet/device-management-service/maint-scripts/e2e"
)

type Args struct {
	shared.ArgsBasic
	shared.ArgsFilters
	shared.ArgsAdjacent
	// TODO support showing ungrouped log lines (with node name prefix for each)
}

func (Args) Description() string {
	return shared.Sprintf(`
		List log lines from a test run.
		
		Install github.com/brocode/fblog
		$> cargo install fblog
		
		Examples:
		
		Logs for all nodes in the latest acceptance test run
		$> logs.sh --test-name acceptance
		
		Logs for all nodes in the E2E deployment_updates test run
		$> logs.sh --test-name deployment_updates
		
		Logs 1 second around line 50 for the node "dms1"
		$> logs.sh --line 50 --node-name dms1 --adjacent-duration 1s
		
		Log lines 50 to 60 for the node "dms1"
		$> logs.sh --line 50:60 --node-name dms1
		
		Log lines 50 to 60 for the node "dms1", with flight times
		$> logs.sh --line 50:60 --node-name dms1 --fligtrec
		
		Logs from 10:09:50 to 10:09:56 for the node "dms1"
		$> logs.sh \
			--timestamp-start 2025-09-24T10:09:50 \
			--timestamp-end 2025-09-24T10:09:56 \
			--node-name dms1
		
		Logs from 10:09:56 for the node "dms1", with 10 adjacent lines
		$> logs.sh --timestamp 2025-09-24T10:09:56 \
			--node-name dms1 \
			--adjacent-lines 10
	`)
}

var args Args

func main() {
	p := arg.MustParse(&args)

	// collect log files
	logs := shared.CollectLogFiles(args.TestName, args.NodeName)

	// process log files
	var output string
	shown := 0
	for _, logFile := range logs {
		// collect
		lines, err := shared.CollectLines(logFile, args.ArgsAdjacent, args.ArgsFilters, args.Flightrec)
		if err != nil {
			p.Fail(fmt.Sprintf("collecting lines for %s: %s", logFile.Name, err.Error()))
		}
		if len(lines) == 0 {
			continue
		}

		// render
		if args.Headers {
			output += shared.RenderHeader(logFile, shown > 0)
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
		err := shared.SaveHTML(filepath.Join(wd, args.OutputHTML), output, args.Headers)
		if err != nil {
			p.Fail(err.Error())
		}
	}
}
