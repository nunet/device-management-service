// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package logs

import (
	"slices"
)

type SpanFactory interface {
	Factory() SpanFactory
	Matches() []Span
	ProcessLine(line *LogLine)
	VerifySpan(span *Span, line *LogLine) bool
}

type Span struct {
	Name      string
	StartLine *LogLine
	EndLine   *LogLine
	Nodes     []string
}

type SpanTree struct {
	Trace    SpanFactory
	Children []*SpanTree
}

// PROCESSOR

type SpanProcessor struct {
	factories []SpanFactory
}

func NewTraceProcessor(factories []SpanFactory) *SpanProcessor {
	return &SpanProcessor{
		factories: factories,
	}
}

func (p *SpanProcessor) MatchesForLine(line *LogLine) []Span {
	var ret []Span

	// find all the matches from registered factories
	for _, t := range p.factories {
		for _, m := range t.Matches() {
			// skip unrelated nodes and open traces
			if !slices.Contains(m.Nodes, line.Node) && !slices.Contains(m.Nodes, line.FromNode) || m.EndLine == nil {
				continue
			}

			// check with the factory
			if !t.VerifySpan(&m, line) {
				continue
			}

			timestamp := line.Timestamp
			lineNum := line.Line
			sl := m.StartLine
			el := m.EndLine

			// match by line nums (only for lines from the same log)
			if sl.Node == el.Node && el.Node == line.Node && sl.Line <= line.Line && el.Line >= lineNum {
				ret = append(ret, m)

				// match by timestamp
			} else if (sl.Timestamp.Before(timestamp) || sl.Timestamp.Equal(timestamp)) &&
				(el.Timestamp.After(timestamp) || el.Timestamp.Equal(timestamp)) {

				ret = append(ret, m)
			}
		}
	}

	return ret
}

func (p *SpanProcessor) ProcessLine(line *LogLine) {
	for _, f := range p.factories {
		f.ProcessLine(line)
	}
}

// REGISTRATION

// TODO build from Args
var (
	deploymentTraces = &SpanDeployments{}
	SpansProc        = NewTraceProcessor([]SpanFactory{
		deploymentTraces.Factory(),
	})
)

// DEPLOYMENTS

type SpanDeployments struct {
	matches []Span
}

var _ SpanFactory = &SpanDeployments{}

func (d *SpanDeployments) Matches() []Span {
	return d.matches
}

func (d *SpanDeployments) Factory() SpanFactory {
	return d
}

func (d *SpanDeployments) ProcessLine(line *LogLine) {
	// match start to open
	if line.Behavior == "/dms/deployment/request" {
		// skip self TODO why self deploy?
		if line.FromNode == line.Node {
			return
		}
		t := Span{
			Name:      "Successful deployment",
			StartLine: line,
			Nodes:     []string{line.Node, line.FromNode},
		}
		d.matches = append(d.matches, t)
		return
	}

	// match end to close
	for i, m := range d.matches {
		// skip closed spans
		if m.EndLine != nil {
			continue
		}
		// varify nodes
		if m.Nodes[0] != line.Node || m.Nodes[1] != line.FromNode {
			continue
		}
		if line.Behavior == "/dms/allocation/start" {
			d.matches[i].EndLine = line
			return
		}
	}
}

func (d *SpanDeployments) VerifySpan(span *Span, line *LogLine) bool {
	if span.Nodes[0] == line.Node && span.Nodes[1] == line.FromNode {
		return true
	}
	if span.Nodes[1] == line.Node && span.Nodes[0] == line.FromNode {
		return true
	}
	return false
}
