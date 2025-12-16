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

	IncludeExternal bool `help:"Include msgs from unknown nodes" default:"false" arg:"--include-external"`

	Diagram      string `help:"Generate a sequence diagram (D2 and SVG) to a file in CWD" default:"msgflow"`
	DiagramImage bool   `help:"Generate a PNG image (slow)" default:"false" arg:"--diagram-image"`
	DiagramSpan  string `help:"Render a single kind of spans to visually mark events" default:"deployments" arg:"--diagram-span"`

	SelfMsgs    bool `help:"Include msgs to itself" default:"true" arg:"--self-msgs"`
	ReplyToMsgs bool `help:"Include '/dms/actor/replyto' msgs" default:"true" arg:"--replyto-msgs"`
	HelloMsgs   bool `help:"Include '/public/hello' msgs" default:"true" arg:"--hello-msgs"`

	BehaviorPrefix string `help:"Limit to behaviors starting with a prefix" arg:"--behavior-prefix"`
}

// Presets is a list of registered presets.
var Presets []string

func (Args) Description() string {
	n := os.Args[0]
	if strings.Contains(n, "go-build") {
		n = "./maint-scripts/e2e/msgflow.sh"
	}
	return shared.Sprintf(`
		List all messages from a test run as a single log, and plot a sequence diagram.
		
		Install github.com/brocode/fblog
		$> cargo install fblog
		
		Install d2 for diagrams, or use https://play.d2lang.com
		$> curl -fsSL https://d2lang.com/install.sh | sh -s --
		
		Examples:
		
		Grouped diagram for the latest acceptance test run
		$> %s acceptance \
			--diagram-group
		
		Diagram for all nodes in the E2E deployment_updates test run
		$> %s deployment_updates
		
		Diagram for the local DMS and logs from ./tmp, max 3 lines per node
		$> %s config \
			--dir tmp \
			--max 3
		
		Filtered version of above
		$> %s config \
			--dir tmp \
			--max 3 \
			--self-msgs=false \
			--replyto-msgs=false \
			--hello-msgs=false
		
		Msgs from 1 second around line 50 for the node "dms1"
		$> %s deployment_updates \
			--line 50 \
			--node-name dms1 \
			--adjacent-duration 1s
		
		Msgs within lines 50 to 60 for the node "dms1"
		$> %s deployment_updates --line 50:60 --node-name dms1
		
		Msgs within lines 50 to 60 for the node "dms1", with flight times
		$> %s deployment_updates --line 50:60 --node-name dms1 --fligtrec
	
		Msgs from 10:09:50 to 10:09:56 for the node "dms1"
		$> %s deployment_updates \
			--timestamp-start 2025-09-24T10:09:50 \
			--timestamp-end 2025-09-24T10:09:56 \
			--node-name dms1
		
		Msgs from 10:09:56 plus 10 adjacent lines, for the node "dms1"
		$> %s deployment_updates \
			--timestamp 2025-09-24T10:09:56 \
			--node-name dms1 \
			--adjacent-lines 10
	
		Presets: %s	
	`, n, n, n, n, n, n, n, n, n, Presets)
}
