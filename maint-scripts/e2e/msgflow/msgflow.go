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
	"gitlab.com/nunet/device-management-service/maint-scripts/e2e/msgflow/presets"
	"gitlab.com/nunet/device-management-service/maint-scripts/e2e/msgflow/types"
)

func init() {
	types.Presets = slices.Concat(
		slices.Collect(maps.Keys(presets.Presets)),
		slices.Collect(maps.Keys(presets.PresetsArgs)),
	)
}

var args types.Args

func main() {
	p := arg.MustParse(&args)

	// intro screen
	if args.SourceName == "" {
		p.WriteHelp(os.Stdout)
		os.Exit(0)
	}

	// validate presets
	for _, preset := range args.Preset {
		if _, ok := presets.Presets[preset]; ok {
			continue
		}
		if _, ok := presets.PresetsArgs[preset]; ok {
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

	// collect log files
	logs := shared.CollectLogFiles(args.ArgsBasic, args.NodeName)
	var results []*shared.LogLine

	// disable level filters
	args.LvlInfo = false

	// process log files
	DIDs := make(map[string]string)
	for i, logFile := range logs {
		// collect
		lines, did, err := shared.CollectLines(logFile, args.ArgsAdjacent, args.ArgsFilters, args.Flightrec)
		if err != nil {
			fmt.Printf("warn: collecting lines for '%s': %v\n", logFile.NodeName, err)
			continue
		}
		logs[i].DID = did
		DIDs[did] = logFile.NodeName
		fmt.Printf("Processing %s (%s)\n", logFile.NodeName, did)
		results = append(results, shared.ParseLines(lines)...)
	}

	if len(results) == 0 {
		p.Fail("no results")
	}

	// handle presets
	for _, preset := range args.Preset {
		if _, ok := presets.Presets[preset]; ok {
			logs, results = presets.Presets[preset](args, logs, results)
		}
	}

	// filter all msg-passing entries
	filtered := make([]*shared.LogLine, 0, len(results))
	for _, l := range results {

		// filters TODO ignore errs caused by node not being rendered (eg no capabilities)
		if l.Error != "" {
			filtered = append(filtered, l)
			continue
		}
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

	// sort, save, and render
	shared.SortByTimestamp(filtered)
	output1 := ""
	if args.Headers && args.HeadersNetwork {
		output1 = shared.RenderLogHeader(logs)
	}
	if output2, err := shared.RenderSlice("msgflow", filtered, args.ArgsBasic); err != nil {
		p.Fail(err.Error())

		// save HTML
	} else if wd, err := os.Getwd(); err == nil && args.OutputHTML != "" {
		err := shared.SaveHTML(filepath.Join(wd, args.OutputHTML), output1+output2, true)
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
func GenDiagram(args types.Args, lines []*shared.LogLine, name string, groupMsgs bool, dids map[string]string) error {
	// init
	path, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	lastMsg := ""
	group := GroupInfo{}

	// declare nodes
	didsSorted := slices.Collect(maps.Values(dids))
	slices.Sort(didsSorted)
	names := ""
	for _, name := range didsSorted {
		// reverse DID
		did := ""
		for k, v := range dids {
			if v == name {
				did = k
				break
			}
		}
		names += fmt.Sprintf("\t- **%s**\n\t\t- *%s*\n", name, did)
	}
	ret := []string{shared.Sprintf(`
		# https://d2lang.com/tour/sequence-diagrams
		shape: sequence_diagram
		vars: { d2-config: { theme-id: 201 } }
	
		explanation: |md
			#Nodes
			
		%s
	
		| {
		  near: top-center
		}
	
	`, names)}

	// parse spans TODO move to main
	proc := shared.SpansProc
	if !groupMsgs && args.DiagramSpan != "" {
		// TODO build the processor
		for _, l := range lines {
			proc.ProcessLine(l)
		}
	}

	// process lines
	spanOpen := false
	for _, l := range lines {
		// errors
		if l.Error != "" {
			// TODO class, break long lines
			ret = append(ret, shared.Sprintf(`
				%s -> %s: ERROR: %s {
					style.stroke: red
					style.stroke-dash: 5
					style.stroke-width: 5
				}`,
				l.Node, l.Node, brakeLine(l.Error, 50)))

			continue
		}

		// messages
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
			spanLeft = "." + s.Name
			spanRight = "." + s.Name

			// render note
			if !spanOpen {
				label := strings.ReplaceAll(s.Name, "_", " ")
				ret = append(ret, fmt.Sprintf(`%s."%s"`, l.Node, label))
				spanOpen = true
			}
			if s.EndLine == l {
				spanOpen = false
			}
		}

		msgTime := l.Timestamp.Format(time.RFC3339)
		if l.FlightTime != "" {
			msgTime = "flight " + l.FlightTime
		}
		ret = append(ret, fmt.Sprintf("%s%s -> %s%s: %s\\nline %s:%d\\n%s",
			from, spanLeft, l.Node, spanRight, brakeLine(l.Behavior, 50),
			l.Node, l.Line, msgTime))

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
	ret = append(ret, fmt.Sprintf("%s -> %s: GROUP OF %d\\nlines %s:%d to %s:%d\\n%s\\n%s",
		from, to, group.Count+1, to, group.LineStart, to, group.LineEnd, group.TimeStart, group.TimeEnd))

	return ret
}

// brakeLine breaks a line at a given limit.
func brakeLine(line string, limit int) string {
	if len(line) <= limit {
		return line
	}
	return line[:limit] + "\\n" + brakeLine(line[limit:], limit)
}
