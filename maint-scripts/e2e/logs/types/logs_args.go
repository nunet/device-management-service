package types

import (
	"os"
	"strings"

	shared "gitlab.com/nunet/device-management-service/maint-scripts/e2e"
)

type Args struct {
	shared.ArgsBasic
	shared.ArgsFilters
	shared.ArgsAdjacent
	shared.ArgsPresets
	// TODO support showing ungrouped log lines (with node name prefix for each)
}

var ArgsPresets []string

func (Args) Description() string {
	n := os.Args[0]
	if strings.Contains(n, "go-build") {
		n = "./maint-scripts/e2e/logs.sh"
	}
	return shared.Sprintf(`
		View logs produced by DMS.
		
		Install github.com/brocode/fblog
		$> cargo install fblog
		
		Examples:
		
		Logs for all nodes in the latest acceptance test run
		$> %s acceptance
		
		Logs for all nodes in the E2E deployment_updates test run
		$> %s deployment_updates
		
		Logs for the local DMS and logs from ./tmp, max 3 lines per node
		$> %s config \
			--dir tmp \
			--max 3
		
		Logs 1 second around line 50 for the node "dms1"
		$> %s deployment_updates \
			--line 50 \
			--node-name dms1 \
			--adjacent-duration 1s
		
		Log lines 50 to 60 for the node "dms1"
		$> %s deployment_updates --line 50:60 --node-name dms1
		
		Log lines 50 to 60 for the node "dms1", with flight times
		$> %s deployment_updates --line 50:60 --node-name dms1 --fligtrec
		
		Logs from 10:09:50 to 10:09:56 for the node "dms1"
		$> %s deployment_updates \
			--timestamp-start 2025-09-24T10:09:50 \
			--timestamp-end 2025-09-24T10:09:56 \
			--node-name dms1
		
		Logs from 10:09:56 for the node "dms1", with 10 adjacent lines
		$> %s deployment_updates \
			--timestamp 2025-09-24T10:09:56 \
			--node-name dms1 \
			--adjacent-lines 10
	
		Presets: %s	
	`, n, n, n, n, n, n, n, n, ArgsPresets)
}
