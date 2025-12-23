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
		View logs produced by Nunet DMS.
		
		Install github.com/brocode/fblog
		$> cargo install fblog
		
		Examples:
		
		Logs for the local DMS and logs from ./tmp, max 3 lines per node
		$> %s --max 3 --dir tmp
		
		Deployment bids from the local DMS via a JSON Query
		$> %s -q '.msg == "deployment_bid"' -f from
	
		Deployment lines by deployment ID
		$> %s -p deployment --preset-args=\
			' --deployment-id=XYZ123'
		
		Last 10 log lines in full
		$> %s --last 10 -a
		
		Logs 1 second around line 50 for the node "dms1"
		$> %s \
			--line 50 \
			--node-name dms1 \
			--adjacent-duration 1s
		
		Log lines 50 to 60 for the node "dms1"
		$> %s --line 50:60 --node-name dms1
		
		Log lines 50 to 60 for the node "dms1", with flight times
		$> %s --line 50:60 --node-name dms1 --fligtrec
		
		Logs from 10:09:50 to 10:09:56 for the node "dms1"
		$> %s \
			--timestamp-start 2025-09-24T10:09:50 \
			--timestamp-end 2025-09-24T10:09:56 \
			--node-name dms1
		
		Logs from 10:09:56 for the node "dms1", with 10 adjacent lines
		$> %s \
			--timestamp 2025-09-24T10:09:56 \
			--node-name dms1 \
			--adjacent-lines 10
		
		Run the "errors" preset with arguments
		$> %s -p errors \
			--preset-args=" --errors-inc-warn"
		
		See help for the "errors" preset
		$> %s -p errors --preset-help
		
		See all errors from all the runs
		$> %s -p errors-all --last-run=false
		
		Mix "bids" and "folded" presets
		$> %s -p bids -p folded
		
		Show the last bid in full ("expanded")
		$> %s -p bids -p expanded --last 1
		
		Logs for all nodes in the latest acceptance test run
		$> %s acceptance
		
		Logs for all nodes in the E2E deployment_updates test run
		$> %s deployment_updates
	
		Presets: %s	
	`, n, n, n, n, n, n, n, n, n, n, n, n, n, n, n, n, ArgsPresets)
}
