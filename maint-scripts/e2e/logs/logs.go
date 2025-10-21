package main

import (
	"fmt"

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
	var nodes []string
	if args.NodeName != "" {
		nodes = append(nodes, args.NodeName)
	}
	logs := shared.CollectLogFiles(args.TestName, nodes)

	// process log files
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
		shared.RenderHeader(logFile)
		if err := shared.RenderSlice("logs-"+logFile.Name, lines); err != nil {
			p.Fail("rendering tmp file: " + err.Error())
		}
	}
}
