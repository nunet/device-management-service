// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.
//revive:disable

package logs

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/bitfield/script"
	t2h "github.com/buildkite/terminal-to-html"
	"github.com/itchyny/gojq"
	"github.com/lithammer/dedent"
	"gitlab.com/nunet/device-management-service/internal/config"
	"golang.org/x/exp/trace"

	"gitlab.com/nunet/device-management-service/actor"
)

//go:embed terminal.css
var CSS string

var HTML = Sprintf(`
	<!DOCTYPE html>
	<html>
		<head>
			<title>TITLE</title>
			<style>STYLESHEET</style>
		</head>
		<body>
			<div class="term-container">CONTENT</div>
		</body>
	</html>
`)

// ///// ///// /////

// ///// TYPES

// ///// ///// /////

var DateFormats = []string{
	time.RFC3339Nano,
	time.RFC3339,
	time.RFC822,
	time.RFC822Z,
	time.DateTime,
	time.DateOnly,
}

type LogLine struct {
	Timestamp  time.Time `json:"timestamp"`
	FlightTime string    `json:"flight_time"`
	Msg        string    `json:"msg"`
	// Node is set by [lineCollector].
	Node     string        `json:"node"`
	Line     int           `json:"line"`
	MsgFrom  *actor.Handle `json:"msg_from"`
	FromNode string        `json:"from_node"`
	Behavior string        `json:"behavior"`
	Error    string        `json:"error"`
	DID      string        `json:"did"`
	Labels   []string      `json:"labels"`
	Label    string        `json:"label"`
	Level    string        `json:"level"`

	RawJSON string `json:"-"`
}

type LogFile struct {
	Path   string
	Name   string
	Number int
	DID    string
	// full path the flight recorder trace file.
	Flightrec string
	// Node's role "cp" or "sp". TODO >1 role?
	Role string
}

type ArgsBasic struct {
	SourceName string   `help:"'acceptance' or 'config' or an E2E test name (required)" arg:"positional" default:"config"`
	NodeName   []string `help:"Show logs of specific nodes, eg dms0 (repeatable)" arg:"--node-name,separate"`
	Config     string   `help:"Fixed path to the config for SOURCENAME config"`
	Dir        []string `help:"Additional source of *.jsonl files (repeatable)" arg:"--dir,separate"`
	Flightrec  bool     `help:"Read flight recorder files for relative timestamps" arg:"--flightrec"`

	ExtraField     []string `help:"Show a specific field from the log line, eg 'msg_from.did' (repeatable)" arg:"-f,--extra-field,separate"`
	SkipField      []string `help:"Skip a specific field from the log line, eg 'msg_from.did' (repeatable)" arg:"-s,--skip-field,separate"`
	NoFields       bool     `help:"Don't show any fields, except extra fields" arg:"-n,--no-fields"`
	NoCommonFields bool     `help:"Don't show common fields (node, line)" arg:"-c,--no-common-fields"`
	AllFields      bool     `help:"Show all fields" arg:"-a,--all-fields"`

	Max            int    `help:"Maximum lines to show per node" default:"1000"`
	Last           int    `help:"Show the last N lines (ignores --max)"`
	Headers        bool   `help:"Show node headers" default:"true"`
	HeadersNetwork bool   `help:"Show the network header" default:"true" arg:"--headers-network"`
	OutputHTML     string `help:"Render HTML to a file" arg:"-o,--output-html"`
}

type ArgsFilters struct {
	LastRun        bool   `help:"Show only the last run of each node (from docker_client_init_started)" arg:"--last-run" default:"true"`
	Line           string `help:"Reference line number, use '12:15' for a range" arg:"-l,--line"`
	Timestamp      string `help:"Reference timestamp"`
	TimestampStart string `help:"Earliest timestamp" arg:"--timestamp-start"`
	TimestampEnd   string `help:"Latest timestamp" arg:"--timestamp-end"`
	Query          string `help:"Custom jq select() query to run on each line, eg '.error'" arg:"-q,--query"`
	LvlInfo        bool   `help:"Show lines up to log level INFO" arg:"--lvl-info"`
}

type ArgsAdjacent struct {
	AdjacentLines    int           `help:"Amount of surrounding lines to output, eg 3" arg:"--adjacent-lines"`
	AdjacentDuration time.Duration `help:"Amount of surrounding time to output, eg 3s" arg:"--adjacent-duration"`
}

type ArgsPresets struct {
	Preset     []string `help:"Run a named preset, eg errors (repeatable)" arg:"-p,--preset,separate"`
	PresetHelp bool     `help:"Show help message for a preset (requires --preset)" default:"false" arg:"--preset-help"`
	PresetArgs string   `help:"Arguments to pass to presets" arg:"--preset-args"`
}

// ///// ///// /////

// ///// FUNCTIONS

// ///// ///// /////

// SourceConfig returns a JSONL filename pointed at by a config file.
func SourceConfig(path string) []LogFile {
	// DMS-style config resolve (internal/config/load.go)
	paths := []string{path}
	if path == "" {
		paths = []string{"./dms_config.json"}
		if hd, err := os.UserHomeDir(); err == nil {
			paths = append(paths, hd+"/.nunet/dms_config.json", hd+"/nunet/dms_config.json")
		}
		if runtime.GOOS != "windows" {
			paths = append(paths, "/etc/nunet/dms_config.json")
		}
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		js, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var cfg config.Config
		if err := json.Unmarshal(js, &cfg); err != nil {
			continue
		}

		// check flightrec.trace
		flightrec := filepath.Join(path, "flightrec.trace")
		if _, err := os.Stat(flightrec); err != nil {
			flightrec = ""
		}

		return []LogFile{{
			Path:      cfg.Logging.File,
			Name:      "config",
			Number:    0,
			Flightrec: flightrec,
		}}
	}

	return nil
}

// SourceAcceptanceLatest returns a list of JSONL filenames from the latest acceptance test run.
func SourceAcceptanceLatest(path string, nodeNames []string) []LogFile {
	ret := make([]LogFile, 0, len(nodeNames))
	var latest time.Time
	var latestStr string
	entries, err := os.ReadDir(path)
	if err != nil {
		return ret
	}
	// format1: alice_sp_logs_1762251399.txt
	// format1: alice_cp_logs_1762251399.txt
	re := regexp.MustCompile(`(.+)_(cp|sp)_logs_(\d+)\.txt`)

	// find the latest timestamp
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// format2: flightrec-1762251399.alice.trace
		matches := re.FindStringSubmatch(entry.Name())
		if len(matches) != 4 {
			continue
		}
		timestamp, err := strconv.Atoi(matches[3])
		if err != nil {
			continue
		}
		t := time.Unix(int64(timestamp), 0)
		if t.Before(latest) {
			continue
		}

		latest = t
		latestStr = matches[3]
	}

	// collect results
	nodeNum := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".txt") {
			continue
		}

		// format1: flightrec_1-1759482433.alice.trace
		// format2: dms_logs_node_3-1759482433.jsonl
		matches := re.FindStringSubmatch(entry.Name())
		if len(matches) != 4 || matches[3] != latestStr {
			continue
		}
		name := matches[1]
		role := matches[2]
		ts := matches[3]

		// filter
		if len(nodeNames) > 0 && !slices.Contains(nodeNames, name) {
			continue
		}

		// check flightrec-1762251399.alice.trace
		flightrec := filepath.Join(path, "flightrec-"+ts+"."+name+".trace")
		if _, err := os.Stat(flightrec); err != nil {
			flightrec = ""
		}
		ret = append(ret, LogFile{
			Path:      filepath.Join(path, entry.Name()),
			Name:      name,
			Number:    nodeNum,
			Role:      role,
			Flightrec: flightrec,
		})
		nodeNum++
	}

	return ret
}

// SourceE2E collects log paths for the given E2E test.
func SourceE2E(logRoot string, nodeNames []string) []LogFile {
	logs := make([]LogFile, 0, len(nodeNames))
	nodeDirs, err := e2eListNodeDirs(logRoot)
	if err != nil {
		return logs
	}

	for i, nodeDir := range nodeDirs {
		if len(nodeNames) > 0 && !slices.Contains(nodeNames, nodeDir) {
			continue
		}

		// check flightrec.trace
		flightrec := filepath.Join(logRoot, nodeDir, "work_dir", "logs", "flightrec.trace")
		if _, err := os.Stat(flightrec); err != nil {
			flightrec = ""
		}
		logs = append(logs, LogFile{
			Path:      filepath.Join(logRoot, nodeDir, "logs.jsonl"),
			Name:      nodeDir,
			Number:    i,
			Flightrec: flightrec,
		})
	}
	return logs
}

func SourceDir(dir string, nodeNames []string, nextNum int) []LogFile {
	// list JSONL files
	files, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if err != nil {
		// TODO log err
		return nil
	}
	ret := make([]LogFile, len(files))
	for i, path := range files {
		// node name
		file := filepath.Base(path)
		node := file[0 : len(file)-len(".jsonl")]
		if len(nodeNames) > 0 && !slices.Contains(nodeNames, node) {
			continue
		}

		// check NODE.flightrec.trace
		flightrec := filepath.Join(dir, node+".flightrec.trace")
		if _, err := os.Stat(flightrec); err != nil {
			flightrec = ""
		}

		// add new log source
		ret[i] = LogFile{
			Path:      path,
			Name:      node,
			Number:    nextNum,
			Flightrec: flightrec,
		}
		nextNum++
	}

	return ret
}

// ParseTimestamp parses a timestamp string using know formats.
func ParseTimestamp(timestamp string) (time.Time, error) {
	for _, format := range DateFormats {
		if t, err := time.Parse(format, timestamp); err == nil {
			return t, nil
		}
		if t, err := time.Parse(format, timestamp+"Z"); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse timestamp '%s'", timestamp)
}

// SortByTimestamp sorts a list of JSONL log entries by the [Filters] field.
func SortByTimestamp(logs []*LogLine) {
	slices.SortFunc(logs, func(a, b *LogLine) int {
		if a.Timestamp.Before(b.Timestamp) {
			return -1
		}
		if a.Timestamp.After(b.Timestamp) {
			return 1
		}
		return 0
	})
}

// RenderLog renders a JSONL file to the terminal, and returns the output.
func RenderLog(path string, remaining int, cfg ArgsBasic) (string, error) {
	common := []string{"node", "line", "did"}

	// compose the list of fields to show
	fields := []string{
		// processed fields

		// TODO _node and _line?
		"node",
		"line",
		// TODO arrays not supported
		// "labels",
		"label",
		"flight_time",

		// DMS produced fields

		"error",
		"behavior",
		"msg_from.did",
		"stack_trace",
		"to",
		"msg_from > did > uri",
		"config",
		"status",
		"allocation",
		"allocationID",
		"deploymentID",
		"contractID",
		"request",
		"path",
		"name",
		"amount",
		"bids",
		"config",
		"addr",
		"hostID",
		"resources",
		"manifest",
		"manifestID",
		"orchestrator",
		"orchestratorID",
		"nodeID",
		"peerID",
		"peer",
		"handle",
		"subnet",
		"ip",
	}
	fields = slices.DeleteFunc(fields, func(f string) bool {
		return slices.Contains(cfg.SkipField, f)
	})
	if cfg.NoFields {
		fields = []string{}
	} else if cfg.NoCommonFields {
		fields = slices.DeleteFunc(fields, func(f string) bool {
			return slices.Contains(common, f)
		})
	}
	fields = append(fields, cfg.ExtraField...)

	// run fblog, add header and footer
	cmd := `fblog -a "` + strings.Join(fields, `" -a "`) + `" ` + path
	if cfg.AllFields {
		cmd = `fblog -d ` + path
	}
	output, _ := script.Exec(cmd).String()
	if cfg.Headers {
		output = "\nRendering: " + path + "\n$> " + cmd + "\n\n" + output
	}
	if remaining > 0 {
		output += fmt.Sprintf("... and %d more lines\n", remaining)
	}

	// output stdout
	fmt.Print(output)

	return output, nil
}

// RenderSlice saves a JSONL file, renders it to the terminal, and returns the output.
func RenderSlice(name string, data []*LogLine, cfg ArgsBasic) (string, error) {
	show := data
	if cfg.Last > 0 {
		start := len(data) - cfg.Last
		start = max(0, start)
		show = data[start:]
	} else if cfg.Max > 0 && len(data) > cfg.Max {
		show = data[:cfg.Max]
	}

	if path, err := SaveSlice(name, show); err != nil {
		return "", err
	} else if ret, err := RenderLog(path, len(data)-len(show), cfg); err != nil {
		return "", err
	} else {
		return ret, nil
	}
}

func RenderSliceHeader(logFile LogFile, prefix bool) string {
	mark := strings.Repeat("##### ", 3)
	ret := fmt.Sprintf("%s\n%s: %s\n%s\n", mark, logFile.Name, logFile.Path, logFile.DID)
	if logFile.Role != "" {
		ret += "role: " + logFile.Role + "\n"
	}
	ret += mark + "\n"
	if prefix {
		ret = "\n" + ret
	}
	fmt.Print(ret)

	return ret
}

func RenderLogHeader(logFiles []LogFile) string {
	mark := strings.Repeat("##### ", 3)
	ret := fmt.Sprintf("%s\nNetwork:\n", mark)
	for _, lf := range logFiles {
		ret += fmt.Sprintf("%-2d \"%s\"\n   %s", lf.Number, lf.Name, lf.DID)
		if lf.Role != "" {
			ret += fmt.Sprintf("\n   role: %s", lf.Role)
		}
		ret += "\n"
	}
	ret += fmt.Sprintf("%s\n\n", mark)
	fmt.Print(ret)

	return ret
}

// CollectLogFiles collects logs based on the E2E test name, or "acceptance" for the latest acceptance test.
func CollectLogFiles(args ArgsBasic, nodeNames []string) []LogFile {
	var ret []LogFile
	var logRoot string
	switch args.SourceName {
	case "config":
		ret = SourceConfig(args.Config)
	case "acceptance":
		logRoot = "tests/acceptance/testdata/logs"
		ret = SourceAcceptanceLatest(logRoot, nodeNames)
	default:
		logRoot = "tests/e2e/testdata/" + args.SourceName
		ret = SourceE2E(logRoot, nodeNames)
	}

	// collect dirs
	for _, dir := range args.Dir {
		ret = append(ret, SourceDir(dir, nodeNames, len(ret))...)
	}

	return ret
}

// CollectLines reads a log file, applies filters and adjacent limits, optionally reads a flight recording, and returns
// matched lines with extra metadata (eg node's name).
// Implementation is backed by an unexported helper type with methods handling each section.
func CollectLines(
	logFile LogFile, adjacent ArgsAdjacent, filters ArgsFilters, flightrecTimes bool,
) (lines []string, did string, err error) {
	cl, err := newLineCollector(logFile, adjacent, filters, flightrecTimes)
	if err != nil {
		return nil, "", err
	}
	return cl.run()
}

func Sprintf(txt string, args ...any) string {
	return fmt.Sprintf(dedent.Dedent(strings.Trim(txt, "\n")), args...)
}

// FlightrecDuration returns the duration of a flight recording.
func FlightrecDuration(path string) (time.Duration, error) {
	// open trace
	f, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("failed to open trace file: %w", err)
	}
	defer f.Close()
	reader, err := trace.NewReader(f)
	if err != nil {
		return 0, fmt.Errorf("failed to create trace reader: %w", err)
	}

	// read and update time
	var endTime trace.Time
	var startTime trace.Time
	for {
		ev, err := reader.ReadEvent()
		if err == io.EOF {
			break // End of file, we're done.
		}
		if err != nil {
			return 0, err
		}

		if startTime == 0 {
			startTime = ev.Time()
		}

		// The timestamp of the last event represents the total duration.
		endTime = ev.Time()
	}

	return endTime.Sub(startTime).Abs(), nil
}

// SaveSlice saves a named JSONL file to /tmp.
func SaveSlice(name string, lines []*LogLine) (string, error) {
	tmpDir := os.TempDir() + "/dms-logs"
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return "", err
	}
	tmpPath := tmpDir + "/" + name + ".jsonl"
	data := ""
	for _, l := range lines {
		data += l.RawJSON + "\n"
	}
	buf := bytes.NewBufferString(data)

	return tmpPath, os.WriteFile(tmpPath, buf.Bytes(), 0o644)
}

func SaveHTML(path, output string, showCmd bool) error {
	if showCmd {
		name := filepath.Base(os.Args[0])
		output = "$ " + name + " " + strings.Join(os.Args[1:], " ") + "\n\n" + output
	}

	html := strings.Replace(HTML, "TITLE", fmt.Sprintf("DMS logs: %s", time.Now().Format(time.RFC3339)), 1)
	html = strings.Replace(html, "STYLESHEET", CSS, 1)
	html = strings.Replace(html, "CONTENT", string(t2h.Render([]byte(output))), 1)

	// mkdir -p and save
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	err := os.WriteFile(path, []byte(html), 0o644)
	if err != nil {
		return fmt.Errorf("failed to write HTML file: %w", err)
	}

	return nil
}

// ParseLines parses JSONL and returns a map of DIDs to nodes.
func ParseLines(lines []string) []*LogLine {
	// parse JSON
	parsed := make([]*LogLine, len(lines))
	for i, line := range lines {
		l := &LogLine{}
		if err := json.Unmarshal([]byte(line), l); err != nil {
			// TODO shorten the slice
			continue
		}
		l.RawJSON = line
		parsed[i] = l
	}

	return parsed
}

// JSONQuery applies a JSON query to the log lines.
// See https://jqlang.org/manual
func JSONQuery(lines []*LogLine, q string) ([]*LogLine, error) {
	if q == "" {
		return nil, fmt.Errorf("query param required")
	}
	parsedQuery, err := gojq.Parse("select( " + q + ")")
	if err != nil {
		return nil, err
	}
	code, err := gojq.Compile(parsedQuery)
	if err != nil {
		return nil, err
	}
	resQuery := make([]*LogLine, 0, len(lines))
	for _, line := range lines {
		var input any
		if err := json.Unmarshal([]byte(line.RawJSON), &input); err != nil {
			return nil, fmt.Errorf("parsing log line: %w", err)
		}
		_, ok := code.Run(input).Next()
		if !ok {
			continue
		}
		resQuery = append(resQuery, line)
	}

	return resQuery, nil
}

// unexported

func e2eListNodeDirs(logRoot string) ([]string, error) {
	entries, err := os.ReadDir(logRoot)
	nodeDirs := []string{}
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			nodeDirs = append(nodeDirs, entry.Name())
		}
	}

	return nodeDirs, nil
}

// ///// ///// /////

// ///// LINE PARSING

// ///// ///// /////

// lineCollector is a log collection pipeline, which collects lines from a log file, based on filters and adjacent
// limits. It also reads a flight recording if requested.
type lineCollector struct {
	logFile   LogFile
	adjacent  ArgsAdjacent
	filters   ArgsFilters
	flightrec bool
	finished  bool

	file    *os.File
	scanner *bufio.Scanner

	// computed filters
	timestampStart time.Time
	timestampEnd   time.Time
	lineStart      int
	lineEnd        int

	// flightrec info
	frDuration time.Duration
	frLogTime  time.Time

	// results state
	results        []string
	did            string
	firstMatchLine int
	firstMatchTime time.Time
	lastMatchTime  time.Time
}

// newLineCollector creates a new lineCollector instance, suitable for a single run.
func newLineCollector(
	logFile LogFile, adjacent ArgsAdjacent, filters ArgsFilters, flightrecTimes bool,
) (*lineCollector, error) {
	f, err := os.Open(logFile.Path)
	if err != nil {
		return nil, fmt.Errorf("reading log: %w", err)
	}
	cl := &lineCollector{
		logFile:        logFile,
		adjacent:       adjacent,
		filters:        filters,
		flightrec:      flightrecTimes,
		file:           f,
		scanner:        bufio.NewScanner(f),
		lineStart:      -1,
		lineEnd:        -1,
		firstMatchLine: -1,
		results:        []string{},
	}

	return cl, nil
}

func (c *lineCollector) close() {
	c.finished = true
	if c.file != nil {
		_ = c.file.Close()
	}
}

func (c *lineCollector) run() (lines []string, did string, err error) {
	if c.finished {
		return nil, "", fmt.Errorf("lineCollector already finished, create a new instance")
	}

	defer c.close()
	if err := c.readFlightrec(); err != nil {
		return nil, "", err
	}
	if err := c.init(); err != nil {
		return nil, "", err
	}
	if err := c.scanMatches(); err != nil {
		return nil, "", err
	}
	if err := c.addAdjacent(); err != nil {
		return nil, "", err
	}
	if err := c.applyQuery(); err != nil {
		return nil, "", err
	}
	if err := c.addFlightrecTimes(); err != nil {
		return nil, "", err
	}

	return c.results, c.did, nil
}

func (c *lineCollector) readFlightrec() error {
	if !c.flightrec || c.logFile.Flightrec == "" {
		return nil
	}
	dur, err := FlightrecDuration(c.logFile.Flightrec)
	if err != nil {
		return fmt.Errorf("reading flightrec: %w", err)
	}
	c.frDuration = dur
	return nil
}

// init validates input data and prepares filter values.
func (c *lineCollector) init() error {
	// validate
	var err error
	if c.filters.Line != "" && c.filters.Timestamp != "" {
		return fmt.Errorf("field Timestamp and Line filters cannot be used together")
	}
	switch {
	case c.filters.Timestamp != "":
		c.timestampStart, err = ParseTimestamp(c.filters.Timestamp)
		if err != nil {
			return fmt.Errorf("field Timestamp: %w", err)
		}
		c.timestampEnd = c.timestampStart
	case c.filters.TimestampStart != "" && c.filters.TimestampEnd != "":
		c.timestampStart, err = ParseTimestamp(c.filters.TimestampStart)
		if err != nil {
			return fmt.Errorf("field TimestampStart: %w", err)
		}
		c.timestampEnd, err = ParseTimestamp(c.filters.TimestampEnd)
		if err != nil {
			return fmt.Errorf("field TimestampEnd: %w", err)
		}
	case c.filters.TimestampStart != "" || c.filters.TimestampEnd != "":
		return fmt.Errorf("field TimestampStart and TimestampEnd must be used together")
	}
	if c.adjacent.AdjacentDuration > 0 && c.timestampStart.IsZero() && c.filters.Line == "" {
		return fmt.Errorf("field AdjacentDuration requires a Timestamp or Line filter")
	}
	if c.adjacent.AdjacentLines > 0 && c.timestampStart.IsZero() && c.filters.Line == "" {
		return fmt.Errorf("field AdjacentLines requires a Timestamp or Line filter")
	}

	// parse line filters
	if c.filters.Line != "" {
		if strings.Contains(c.filters.Line, ":") {
			nums := strings.Split(c.filters.Line, ":")
			if len(nums) != 2 {
				return fmt.Errorf("invalid line range: %s", c.filters.Line)
			}
			if c.lineStart, err = strconv.Atoi(nums[0]); err != nil {
				return fmt.Errorf("invalid line range: %s", c.filters.Line)
			}
			if c.lineEnd, err = strconv.Atoi(nums[1]); err != nil {
				return fmt.Errorf("invalid line range: %s", c.filters.Line)
			}
			if c.lineEnd < c.lineStart {
				return fmt.Errorf("invalid line range: %s", c.filters.Line)
			}
		} else {
			if c.lineStart, err = strconv.Atoi(c.filters.Line); err != nil {
				return fmt.Errorf("invalid line range: %s", c.filters.Line)
			}
			c.lineEnd = c.lineStart
		}
	}

	return nil
}

func (c *lineCollector) scanMatches() error {
	scanner := c.scanner
	lineNum := 1
	for scanner.Scan() {
		var entry LogLine
		line := scanner.Bytes()
		if err := json.Unmarshal(line, &entry); err != nil {
			lineNum++
			continue
		}

		match := false
		filtered := false
		ts := entry.Timestamp
		tsEvenSec := ts.Add(time.Duration(-ts.Nanosecond()))

		if !c.timestampStart.IsZero() {
			filtered = true
			if (ts.After(c.timestampStart) || tsEvenSec.Equal(c.timestampStart) || ts.Equal(c.timestampStart)) &&
				(ts.Before(c.timestampEnd) || tsEvenSec.Equal(c.timestampEnd) || ts.Equal(c.timestampEnd)) {
				match = true
			}
		}

		if entry.Msg == "flightrec_captured" {
			c.frLogTime = entry.Timestamp
		} else if entry.Msg == "docker_client_init_started" && c.filters.LastRun {
			c.results = nil
		}

		if c.lineStart >= 0 {
			filtered = true
			if lineNum >= c.lineStart && lineNum <= c.lineEnd {
				match = true
			}
		}

		// skip level DEBUG
		if c.filters.LvlInfo {
			allow := []string{"INFO", "ERROR", "WARN"}
			if !slices.Contains(allow, entry.Level) {
				continue
			}
		}

		if filtered && !match {
			lineNum++
			continue
		}

		if len(c.results) == 0 {
			c.firstMatchLine = lineNum
			c.firstMatchTime = entry.Timestamp
		}
		c.lastMatchTime = entry.Timestamp

		// remap labels[0] to label
		label := ""
		if len(entry.Labels) > 0 && entry.Labels[0] != "default" {
			label = `"label": "` + entry.Labels[0] + `", `
		}

		c.results = append(c.results, fmt.Sprintf(`{"node": "%s", "line": %d, %s`,
			c.logFile.Name, lineNum, label)+string(line)[1:])
		if c.did == "" {
			c.did = entry.DID
		}
		lineNum++
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading log: %w", err)
	}

	return nil
}

func (c *lineCollector) addAdjacent() error {
	if c.adjacent.AdjacentLines <= 0 && c.adjacent.AdjacentDuration <= 0 {
		return nil
	}
	before := []string{}
	after := []string{}
	lastMatchedLine := c.firstMatchLine + len(c.results) - 1
	if _, err := c.file.Seek(0, 0); err != nil {
		return fmt.Errorf("rewinding log file: %w", err)
	}
	scanner := bufio.NewScanner(c.file)
	lineNum := 1

	for scanner.Scan() {
		if lineNum >= c.firstMatchLine && lineNum <= lastMatchedLine {
			lineNum++
			continue
		}
		match := false
		var entry LogLine
		line := scanner.Bytes()
		if err := json.Unmarshal(line, &entry); err != nil {
			lineNum++
			continue
		}
		if c.adjacent.AdjacentLines > 0 {
			if lineNum < c.firstMatchLine && lineNum >= c.firstMatchLine-c.adjacent.AdjacentLines {
				match = true
			} else if lineNum > lastMatchedLine && lineNum <= lastMatchedLine+c.adjacent.AdjacentLines {
				match = true
			}
		}
		if c.adjacent.AdjacentDuration > 0 {
			if entry.Timestamp.Add(c.adjacent.AdjacentDuration).After(c.firstMatchTime) &&
				entry.Timestamp.Add(-c.adjacent.AdjacentDuration).Before(c.lastMatchTime) {

				match = true
			}
		}
		if !match {
			lineNum++
			continue
		}
		resultLine := fmt.Sprintf(`{"node": "%s", "line": %d, `, c.logFile.Name, lineNum) + string(line)[1:]
		if lineNum < c.firstMatchLine {
			before = append(before, resultLine)
		} else {
			after = append(after, resultLine)
		}
		lineNum++
	}
	c.results = slices.Concat(before, c.results, after)

	return nil
}

func (c *lineCollector) applyQuery() error {
	if c.filters.Query == "" {
		return nil
	}
	parsedQuery, err := gojq.Parse("select( " + c.filters.Query + ")")
	if err != nil {
		return err
	}
	code, err := gojq.Compile(parsedQuery)
	if err != nil {
		return err
	}
	resQuery := make([]string, 0, len(c.results))
	for _, line := range c.results {
		var input map[string]any
		if err := json.Unmarshal([]byte(line), &input); err != nil {
			return fmt.Errorf("parsing log line: %w", err)
		}
		if _, ok := input["allocationID"]; ok {
			print()
		}
		_, ok := code.Run(input).Next()
		if !ok {
			continue
		}
		resQuery = append(resQuery, line)
	}
	c.results = resQuery

	return nil
}

func (c *lineCollector) addFlightrecTimes() error {
	//nolint:staticcheck
	if !(c.flightrec && c.frDuration > 0 && !c.frLogTime.IsZero()) {
		return nil
	}
	for i, line := range c.results {
		var entry LogLine
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return fmt.Errorf("parsing log line: %w", err)
		}
		flightTime := c.frDuration - c.frLogTime.Sub(entry.Timestamp)
		if flightTime < 0 {
			continue
		}
		c.results[i] = fmt.Sprintf(`{"flight_time": "%.3fs", %s`, flightTime.Seconds(), line[1:])
	}

	return nil
}
