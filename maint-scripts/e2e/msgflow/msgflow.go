package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/alexflint/go-arg"

	shared "gitlab.com/nunet/device-management-service/maint-scripts/e2e"
)

type Args struct {
	shared.ArgsBasic
	shared.ArgsFilters
	shared.ArgsAdjacent
	IncludeExternal bool `help:"Include msgs from unknown nodes" default:"false" arg:"--include-external"`
	Diagram         bool `help:"Generate a sequence diagram (D2 and SVG)" default:"true"`
	DiagramImage    bool `help:"Generate a PNG image (slow)" default:"false" arg:"--diagram-image"`
	DiagramGroup    bool `help:"Group consecutive messages between nodes" default:"false" arg:"--diagram-group"`
	// TODO limit by repeated DIDs / dir names / acceptance names
	//  --only dms0 --only dms1
	//  --only did:123 --only alice
	// TODO include expired false
	//  --include-expired-behaviors
	// TODO include unknown behaviors false
	//  --include-unknown-behaviors
	// TODO add names to DIDs
	//  --did 123 --name Foo --did 456 --name Bar
}

func (Args) Description() string {
	return shared.Sprintf(`
	List all messages from a test run as a single log, and plot a sequence diagram.
	
	Install github.com/brocode/fblog
	$> cargo install fblog
	
	Install d2 for diagrams, or use https://play.d2lang.com
	$> curl -fsSL https://d2lang.com/install.sh | sh -s --
	
	Examples:
	
	Grouped diagram for the latest acceptance test run
	$> logs.sh --test-name acceptance --diagram-group
	
	Log for all nodes in the E2E deployment_updates test run
	$> logs.sh --test-name deployment_updates
	
	Log 1 second around line 50 for the node "dms1"
	$> logs.sh --line 50 --node-name dms1 --adjacent-duration 1s
	
	Msgs within lines 50 to 60 for the node "dms1"
	$> logs.sh --line 50:60 --node-name dms1
	
	Msgs within lines 50 to 60 for the node "dms1", with flight times
	$> logs.sh --line 50:60 --node-name dms1 --fligtrec

	Msgs from 10:09:50 to 10:09:56 for the node "dms1"
	$> logs.sh \
		--timestamp-start 2025-09-24T10:09:50 \
		--timestamp-end 2025-09-24T10:09:56 \
		--node-name dms1
	
	Msgs from 10:09:56 plus 10 adjacent lines, for the node "dms1"
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
	var results []string

	// process log files
	for _, logFile := range logs {
		// collect
		lines, err := shared.CollectLines(logFile, args.ArgsAdjacent, args.ArgsFilters, args.Flightrec)
		if err != nil {
			p.Fail(fmt.Sprintf("collecting lines for %s: %s", logFile.Name, err.Error()))
		}
		results = append(results, lines...)
	}

	if len(results) == 0 {
		p.Fail("no results")
	}

	// filter all msg-passing entries
	filtered := make([]*shared.LogLine, 0, len(results))
	filteredStr := make([]string, 0, len(results))
	DIDs := make(map[string]string)
	for _, line := range results {
		// parse
		l := &shared.LogLine{}
		if err := json.Unmarshal([]byte(line), l); err != nil {
			continue
		}

		// build a map
		if l.DID != "" {
			DIDs[l.DID] = l.Node
		}

		// filter
		if l.MsgFrom == nil {
			continue
		}
		fromDID := l.MsgFrom.DID.String()
		if _, ok := DIDs[fromDID]; !ok && !args.IncludeExternal {
			continue
		}

		// filter ok
		filtered = append(filtered, l)
		filteredStr = append(filteredStr, line)
	}

	// sort, save, and render
	shared.SortByTimestamp(results)
	if err := shared.RenderSlice("msgflow", filteredStr); err != nil {
		p.Fail("rendering tmp file: " + err.Error())
	}

	// gen diagram
	if args.Diagram {
		if err := GenDiagram(args, filtered, DIDs); err != nil {
			p.Fail("generating diagram: " + err.Error())
		}
		return
	}
}

type GroupInfo struct {
	Count     int
	LineStart int
	LineEnd   int
	TimeStart string
	TimeEnd   string
}

// GenDiagram generates a sequence diagram from the given logs.
//
// TODO color errors in red
// TODO add regions
//
//	region: {
//	  ...
//	}
func GenDiagram(args Args, lines []*shared.LogLine, didMap map[string]string) error {
	path := shared.LogRoot(args.TestName)
	ret := []string{
		"shape: sequence_diagram",
		"vars: { d2-config: { theme-id: 201 } }",
	}
	lastMsg := ""
	group := GroupInfo{}

	// process lines
	for _, l := range lines {
		fromDID := l.MsgFrom.DID.String()
		from := "external"
		if _, ok := didMap[fromDID]; ok {
			from = didMap[fromDID]
		}

		// group
		if args.DiagramGroup {
			if lastMsg == from+"|"+l.Node {
				group.LineEnd = l.Line
				if l.FlightTime != "" {
					group.TimeEnd = "flight to " + l.FlightTime
				} else {
					group.TimeEnd = l.Timestamp.Format(time.RFC3339)
				}
				group.Count++

				// group at least 2msgs
			} else if group.Count > 0 {
				ret = groupResults(lastMsg, ret, group)
				group.Count = 0
			}
		}

		// TODO break long lines of l.Behavior
		msgTime := l.Timestamp.Format(time.RFC3339)
		if l.FlightTime != "" {
			msgTime = "flight " + l.FlightTime
		}
		ret = append(ret, fmt.Sprintf("%s -> %s: line %s:%d\\n%s\\n%s",
			from, l.Node, l.Node, l.Line, msgTime, l.Behavior))

		// grouping state
		lastMsg = from + "|" + l.Node
		if group.Count == 0 {
			group.LineStart = l.Line
			if l.FlightTime != "" {
				group.TimeStart = "flight from " + l.FlightTime
			} else {
				group.TimeStart = l.Timestamp.Format(time.RFC3339)
			}
		}
	}

	// group tail
	if group.Count > 0 {
		ret = groupResults(lastMsg, ret, group)
	}

	fmt.Println()

	// text
	fmt.Println("Generating msgflow.d2...")
	if err := os.WriteFile(path+"/msgflow.d2", []byte(strings.Join(ret, "\n")), 0o644); err != nil {
		return fmt.Errorf("writing d2 file: %w", err)
	}

	// svg
	fmt.Println("Generating msgflow.svg...")
	cmd := exec.Command("d2", path+"/msgflow.d2", path+"/msgflow.svg")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("generating svg: %w", err)
	}

	// png TODO progress bar
	if args.DiagramImage {
		fmt.Print("Generating msgflow.png (1-3m)...")
		cmd := exec.Command("d2", path+"/msgflow.d2", path+"/msgflow.png")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("generating png: %w", err)
		}
	}

	return nil
}

func groupResults(
	lastMsg string, ret []string, group GroupInfo,
) []string {
	msg := strings.Split(lastMsg, "|")
	from := msg[0]
	to := msg[1]
	ret = ret[0 : len(ret)-group.Count-1]
	ret = append(ret, fmt.Sprintf("%s -> %s: lines %s:%d to %s:%d\\n%s\\n%s\\ngroup of %d",
		from, to, to, group.LineStart, to, group.LineEnd, group.TimeStart, group.TimeEnd, group.Count+1))

	return ret
}
