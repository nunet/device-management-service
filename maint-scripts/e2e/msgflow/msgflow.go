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
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/alexflint/go-arg"

	shared "gitlab.com/nunet/device-management-service/maint-scripts/e2e"
)

type Args struct {
	shared.ArgsBasic
	shared.ArgsFilters
	shared.ArgsAdjacent

	IncludeExternal bool   `help:"Include msgs from unknown nodes" default:"false" arg:"--include-external"`
	Diagram         string `help:"Generate a sequence diagram (D2 and SVG) to a file in CWD" default:"msgflow"`
	DiagramImage    bool   `help:"Generate a PNG image (slow)" default:"false" arg:"--diagram-image"`
	SelfMsgs        bool   `help:"Include msgs to itself" default:"true" arg:"--self-msgs"`
	ReplyToMsgs     bool   `help:"Include '/dms/actor/replyto' msgs" default:"true" arg:"--replyto-msgs"`
	HelloMsgs       bool   `help:"Include '/public/hello' msgs" default:"true" arg:"--hello-msgs"`
	BehaviorPrefix  string `help:"Limit to behaviors starting with a prefix" arg:"--behavior-prefix"`
	Span            string `help:"Render a single kind of spans to visually mark events" default:"deployments"`
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
	logs := shared.CollectLogFiles(args.TestName, args.NodeName)
	var results []string

	// disable level filters
	args.LvlInfo = false

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
	parsed, DIDs := shared.ParseLines(results)
	for _, l := range parsed {

		// filters
		if l.MsgFrom == nil {
			continue
		}
		fromDID := l.MsgFrom.DID.String()
		if name, ok := DIDs[fromDID]; !ok && !args.IncludeExternal {
			continue
		} else if !args.SelfMsgs && name == l.Node {
			continue
		}
		if !args.ReplyToMsgs && strings.HasPrefix(l.Behavior, "/dms/actor/replyto") {
			continue
		}
		if !args.HelloMsgs && l.Behavior == "/public/hello" {
			continue
		}
		if args.BehaviorPrefix != "" && strings.HasPrefix(l.Behavior, args.BehaviorPrefix) {
			continue
		}

		// add fields
		l.FromNode = DIDs[l.MsgFrom.DID.String()]
		l.RawJSON = `{"from_node": "` + l.FromNode + `", ` + l.RawJSON[1:]

		// filter ok
		filtered = append(filtered, l)
	}
	filteredStr := make([]string, len(filtered))
	for i, l := range filtered {
		filteredStr[i] = l.RawJSON
	}

	// sort, save, and render
	shared.SortByTimestamp(filtered)
	// TODO render DID map
	if output, err := shared.RenderSlice("msgflow", filteredStr, args.ArgsBasic); err != nil {
		p.Fail(err.Error())

		// save HTML
	} else if wd, err := os.Getwd(); err == nil && args.OutputHTML != "" {
		err := shared.SaveHTML(filepath.Join(wd, args.OutputHTML), output, args.Headers)
		if err != nil {
			p.Fail(err.Error())
		}
	}

	// gen diagram
	if args.Diagram != "" {
		if err := GenDiagram(args, filtered, args.Diagram, false, DIDs); err != nil {
			p.Fail(err.Error())
		}
		if err := GenDiagram(args, filtered, args.Diagram+"-grouped", true, DIDs); err != nil {
			p.Fail(err.Error())
		}
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
func GenDiagram(args Args, lines []*shared.LogLine, name string, groupMsgs bool, dids map[string]string) error {
	// init
	path, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	lastMsg := ""
	group := GroupInfo{}
	ret := []string{
		"# https://d2lang.com/tour/sequence-diagrams",
		"shape: sequence_diagram",
		"vars: { d2-config: { theme-id: 201 } }",
	}

	// declare nodes
	didsSorted := slices.Collect(maps.Values(dids))
	slices.Sort(didsSorted)
	ret = append(ret, didsSorted...)

	// parse spans
	proc := shared.SpansProc
	if !groupMsgs && args.Span != "" {
		// TODO build the processor
		for _, l := range lines {
			proc.ProcessLine(l)
		}
	}

	// process lines
	spanOpen := false
	for _, l := range lines {
		fromDID := l.MsgFrom.DID.String()
		from := "external"
		if v, ok := dids[fromDID]; ok {
			from = v
		}

		spanLeft := ""
		spanRight := ""

		// group
		if groupMsgs {
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

			// spans
		} else if spans := proc.MatchesForLine(l); spans != nil {
			// first span only TODO nesting spans
			s := spans[0]
			n := strings.ReplaceAll(s.Name, " ", "_")
			spanLeft = "." + n
			spanRight = "." + n

			// render note
			if !spanOpen {
				ret = append(ret, from+`."`+s.Name+`"`)
				spanOpen = true
			}
			if s.EndLine == l {
				spanOpen = false
			}
		}

		// TODO break long lines of l.Behavior
		msgTime := l.Timestamp.Format(time.RFC3339)
		if l.FlightTime != "" {
			msgTime = "flight " + l.FlightTime
		}
		ret = append(ret, fmt.Sprintf("%s%s -> %s%s: %s\\nline %s:%d\\n%s",
			from, spanLeft, l.Node, spanRight, l.Behavior, l.Node, l.Line, msgTime))

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

	// save D2
	fmt.Println("Generating " + name + ".d2...")
	d2Diag := []byte(strings.Join(ret, "\n"))
	target := filepath.Join(path, name)
	d2Path := target + ".d2"
	if err := os.WriteFile(d2Path, d2Diag, 0o644); err != nil {
		return fmt.Errorf("writing d2 file: %w", err)
	}

	// save SVG
	fmt.Println("Generating " + name + ".svg...")
	cmd := exec.Command("d2", d2Path, target+".svg")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("generating svg: %w", err)
	}

	// save PNG TODO progress bar
	if args.DiagramImage {
		fmt.Print("Generating " + name + ".png (1-3m)...")
		cmd := exec.Command("d2", d2Path, target+".png")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("generating png: %w", err)
		}
	}

	return nil
}

func groupResults(lastMsg string, ret []string, group GroupInfo) []string {
	msg := strings.Split(lastMsg, "|")
	from := msg[0]
	to := msg[1]
	ret = ret[0 : len(ret)-group.Count-1]
	ret = append(ret, fmt.Sprintf("%s -> %s: group of %d\\nlines %s:%d to %s:%d\\n%s\\n%s",
		from, to, group.Count+1, to, group.LineStart, to, group.LineEnd, group.TimeStart, group.TimeEnd))

	return ret
}
