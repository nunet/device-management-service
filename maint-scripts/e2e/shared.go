package scriptse2e

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/bitfield/script"
	"github.com/itchyny/gojq"
	"github.com/lithammer/dedent"
	"golang.org/x/exp/trace"

	"gitlab.com/nunet/device-management-service/actor"
)

var DateFormats = []string{
	time.RFC3339Nano,
	time.RFC3339,
	time.RFC822,
	time.RFC822Z,
	time.DateTime,
	time.DateOnly,
}

type LogLine struct {
	Timestamp  time.Time     `json:"timestamp"`
	FlightTime string        `json:"flight_time"`
	Msg        string        `json:"msg"`
	Node       string        `json:"node"`
	Line       int           `json:"line"`
	MsgFrom    *actor.Handle `json:"msg_from"`
	Behavior   string        `json:"behavior"`
	Error      string        `json:"error"`
	DID        string        `json:"did"`
}

type LogFile struct {
	Path   string
	Name   string
	Number int
	// full path the flight recorder trace file.
	Flightrec string
}

type ArgsBasic struct {
	TestName string `help:"'acceptance' or an E2E test name" arg:"positional,required"`
	// TODO support repeated
	NodeName  string `help:"Show logs of specific node, eg dms0" arg:"--node-name"`
	Flightrec bool   `help:"Read a flight recorder file for relative timestamps" arg:"--flightrec"`
	// TODO ExtraField, repeated
	// TODO SkipField, repeated
}

type ArgsFilters struct {
	Line           string `help:"Reference line number, use '12:15' for a range"`
	Timestamp      string `help:"Reference timestamp"`
	TimestampStart string `help:"Earliest timetsamp" arg:"--timestamp-start"`
	TimestampEnd   string `help:"Latest timestamp" arg:"--timestamp-end"`
	Query          string `help:"Custom jq select() query to run on each line, eg '.error'"`
}

type ArgsAdjacent struct {
	AdjacentLines    int           `help:"Amount of surrounding lines to output, eg 3" arg:"--adjacent-lines"`
	AdjacentDuration time.Duration `help:"Amount of surrounding time to output, eg 3s" arg:"--adjacent-duration"`
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

// SaveSlice saves a named JSONL file to /tmp.
func SaveSlice(name string, data []string) (string, error) {
	tmpDir := os.TempDir() + "/dms-logs"
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return "", err
	}
	tmpPath := tmpDir + "/" + name + ".jsonl"
	buf := bytes.NewBufferString(strings.Join(data, "\n"))
	return tmpPath, os.WriteFile(tmpPath, buf.Bytes(), 0o644)
}

// RenderLog renders a JSONL file to the terminal.
func RenderLog(path string) error {
	// TODO ExtraField, SkipField
	fmt.Println()
	cmd := "fblog -a node -a line -a did -a error -a behavior -a msg_from.did -a stack_trace " +
		`-a "msg_from > did > uri" -a flight_time -a to ` +
		path
	fmt.Println("Rendering: " + path)
	fmt.Println("$> " + cmd)
	output, _ := script.Exec(cmd).String()
	fmt.Print(output)

	return nil
}

// SortByTimestamp sorts a list of JSONL log entries by the [Filters] field.
func SortByTimestamp(logs []string) {
	slices.SortFunc(logs, func(a, b string) int {
		var entryA, entryB LogLine
		if err := json.Unmarshal([]byte(a), &entryA); err != nil {
			return 0
		}
		if err := json.Unmarshal([]byte(b), &entryB); err != nil {
			return 0
		}
		if entryA.Timestamp.Before(entryB.Timestamp) {
			return -1
		}
		if entryA.Timestamp.After(entryB.Timestamp) {
			return 1
		}
		return 0
	})
}

// LogsAcceptanceLatest returns a list of JSONL filenames from the latest acceptance test run.
func LogsAcceptanceLatest(path string, nodeNames []string) []LogFile {
	ret := make([]LogFile, 0, len(nodeNames))
	var latest time.Time
	var id string
	entries, err := os.ReadDir(path)
	if err != nil {
		return ret
	}

	// find the latest run
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// format1: flightrec_1-1759482433.alice.trace
		// format2: dms_logs_node_3-1759482433.jsonl
		re := regexp.MustCompile(`[^-]+_(\d+)-(\d+)`)
		matches := re.FindStringSubmatch(entry.Name())
		if len(matches) != 3 {
			continue
		}
		timestamp, err := strconv.Atoi(matches[2])
		if err != nil {
			continue
		}
		t := time.Unix(int64(timestamp), 0)
		if t.Before(latest) {
			continue
		}

		latest = t
		id = matches[2]
	}

	// collect results
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}

		// format1: flightrec_1-1759482433.alice.trace
		// format2: dms_logs_node_3-1759482433.jsonl
		re := regexp.MustCompile(`[^-]+_(\d+)-(\d+)`)
		matches := re.FindStringSubmatch(entry.Name())
		if len(matches) != 3 || matches[2] != id {
			continue
		}
		num, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}

		// result
		name := "dms" + matches[1]
		if len(nodeNames) > 0 && !slices.Contains(nodeNames, name) {
			continue
		}
		ret = append(ret, LogFile{
			Path: filepath.Join(path, entry.Name()),
			// TODO support name suffixes alice, charlie, etc.
			Name:   name,
			Number: num,
			// TODO get the name from the log file
			Flightrec: "flightrec_1-1759482433.alice.trace",
		})
	}

	return ret
}

// LogsE2E collects log paths for the given E2E test.
func LogsE2E(logRoot string, nodeNames []string) []LogFile {
	logs := make([]LogFile, 0, len(nodeNames))
	nodeDirs, err := e2eListNodeDirs(logRoot)
	if err != nil {
		return logs
	}

	for i, nodeDir := range nodeDirs {
		if len(nodeNames) > 0 && !slices.Contains(nodeNames, nodeDir) {
			continue
		}
		logs = append(logs, LogFile{
			Path:      filepath.Join(logRoot, nodeDir, "logs.jsonl"),
			Name:      nodeDir,
			Number:    i,
			Flightrec: filepath.Join(logRoot, nodeDir, "work_dir", "logs", "flightrec.trace"),
		})
	}
	return logs
}

// RenderSlice saves a JSONL file and renders it to the terminal.
func RenderSlice(name string, data []string) error {
	if path, err := SaveSlice(name, data); err != nil {
		return err
	} else if err := RenderLog(path); err != nil {
		return err
	}

	return nil
}

func RenderHeader(logFile LogFile) {
	mark := strings.Repeat("##### ", 3)
	fmt.Printf("\n%s\n%s: %s\n%s\n", mark, logFile.Name, logFile.Path, mark)
}

func LogRoot(testName string) string {
	if testName == "acceptance" {
		return "tests/acceptance/testdata/logs"
	}
	return "tests/e2e/testdata/" + testName
}

// CollectLogFiles collects logs based on the E2E test name, or "acceptance" for the latest acceptance test.
func CollectLogFiles(testName string, nodeNames []string) []LogFile {
	var ret []LogFile
	var logRoot string
	if testName == "acceptance" {
		logRoot = "tests/acceptance/testdata/logs"
		ret = LogsAcceptanceLatest(logRoot, nodeNames)
	} else {
		logRoot = "tests/e2e/testdata/" + testName
		ret = LogsE2E(logRoot, nodeNames)
	}

	return ret
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

// CollectLines reads a log file, applies filters and adjacent limits, optionally read a flight recording, and returns
// matched lines with extra metadata (eg node's name).
// TODO split to smaller funcs
func CollectLines(logFile LogFile, adjacent ArgsAdjacent, filters ArgsFilters, flightrecTimes bool) ([]string, error) {
	// log file
	file, err := os.Open(logFile.Path)
	if err != nil {
		return nil, fmt.Errorf("reading log: %w", err)
	}
	defer file.Close()
	timestampStart := time.Time{}
	timestampEnd := time.Time{}

	// read flightrec for relative timestamps
	var (
		frDuration time.Duration
		frLogTime  time.Time
	)
	if flightrecTimes {
		frDuration, err = FlightrecDuration(logFile.Flightrec)
		if err != nil {
			return nil, fmt.Errorf("reading flightrec: %w", err)
		}
	}

	// validate
	if filters.Line != "" && filters.Timestamp != "" {
		return nil, fmt.Errorf("field Timestamp and Line filters cannot be used together")
	}
	switch {
	case filters.Timestamp != "":
		timestampStart, err = ParseTimestamp(filters.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("field Timestamp: %w", err)
		}
		timestampEnd = timestampStart
	case filters.TimestampStart != "" && filters.TimestampEnd != "":
		timestampStart, err = ParseTimestamp(filters.TimestampStart)
		if err != nil {
			return nil, fmt.Errorf("field TimestampStart: %w", err)
		}
		timestampEnd, err = ParseTimestamp(filters.TimestampEnd)
		if err != nil {
			return nil, fmt.Errorf("field TimestampEnd: %w", err)
		}
	case filters.TimestampStart != "" || filters.TimestampEnd != "":
		return nil, fmt.Errorf("field TimestampStart and TimestampEnd must be used together")
	}
	if adjacent.AdjacentDuration > 0 && timestampStart.IsZero() && filters.Line == "" {
		return nil, fmt.Errorf("field AdjacentDuration requires a Timestamp or Line filter")
	}
	if adjacent.AdjacentLines > 0 && timestampStart.IsZero() && filters.Line == "" {
		return nil, fmt.Errorf("field AdjacentLines requires a Timestamp or Line filter")
	}

	// line filters
	lineStart := -1
	lineEnd := -1
	if filters.Line != "" {
		// range eg 12:15
		if strings.Contains(filters.Line, ":") {
			nums := strings.Split(filters.Line, ":")
			if len(nums) != 2 {
				return nil, fmt.Errorf("invalid line range: %s", filters.Line)
			}
			if lineStart, err = strconv.Atoi(nums[0]); err != nil {
				return nil, fmt.Errorf("invalid line range: %s", filters.Line)
			}
			if lineEnd, err = strconv.Atoi(nums[1]); err != nil {
				return nil, fmt.Errorf("invalid line range: %s", filters.Line)
			}

			if lineEnd < lineStart {
				return nil, fmt.Errorf("invalid line range: %s", filters.Line)
			}

			// single line
		} else {
			if lineStart, err = strconv.Atoi(filters.Line); err != nil {
				return nil, fmt.Errorf("invalid line range: %s", filters.Line)
			}
			lineEnd = lineStart
		}
	}

	// read the log, filter
	results := []string{}
	firstMatchLine := -1
	var firstMatchTime time.Time
	var lastMatchTime time.Time
	scanner := bufio.NewScanner(file)
	lineNum := 1
	for scanner.Scan() {

		// parse line
		var entry LogLine
		line := scanner.Bytes()
		if err := json.Unmarshal(line, &entry); err != nil {
			lineNum++
			continue
		}

		match := false
		filtered := false
		ts := entry.Timestamp
		// match also round secs TODO use Truncate
		tsEvenSec := ts.Add(time.Duration(-ts.Nanosecond()))

		// timestamp filter
		if !timestampStart.IsZero() {
			filtered = true
			// inclusive start
			if (ts.After(timestampStart) || tsEvenSec.Equal(timestampStart) || ts.Equal(timestampStart)) &&
				// inclusive end
				(ts.Before(timestampEnd) || tsEvenSec.Equal(timestampEnd) || ts.Equal(timestampEnd)) {

				match = true
			}
		}

		// catch flightrec
		if entry.Msg == "flightrec_captured" {
			frLogTime = entry.Timestamp
		}

		// line filter
		if lineStart >= 0 {
			filtered = true
			if lineNum >= lineStart && lineNum <= lineEnd {
				match = true
			}
		}

		if filtered && !match {
			lineNum++
			continue
		}

		// memorize positions
		if len(results) == 0 {
			firstMatchLine = lineNum
			firstMatchTime = entry.Timestamp
		}
		lastMatchTime = entry.Timestamp

		// add to result with metadata
		results = append(results, fmt.Sprintf(
			`{"node": "%s", "line": %d, `, logFile.Name, lineNum)+
			string(line)[1:])
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading log: %w", err)
	}

	// add adjacent lines
	if adjacent.AdjacentLines > 0 || adjacent.AdjacentDuration > 0 {

		// prepare
		var before []string
		var after []string
		lastMatchedLine := firstMatchLine + len(results) - 1
		lineNum = 1
		if _, err := file.Seek(0, 0); err != nil {
			return nil, fmt.Errorf("rewinding log file: %w", err)
		}
		scanner = bufio.NewScanner(file)

		// scan
		for scanner.Scan() {
			// skip matched
			if lineNum >= firstMatchLine && lineNum <= lastMatchedLine {
				lineNum++
				continue
			}

			// parse line
			match := false
			var entry LogLine
			line := scanner.Bytes()
			if err := json.Unmarshal(line, &entry); err != nil {
				lineNum++
				continue
			}

			// add by lines
			if adjacent.AdjacentLines > 0 {
				// preceding lines
				if lineNum < firstMatchLine && lineNum >= firstMatchLine-adjacent.AdjacentLines {
					match = true
				} else if lineNum > lastMatchedLine &&
					lineNum <= lastMatchedLine+adjacent.AdjacentLines {
					match = true
				}
			}

			// add by duration
			if adjacent.AdjacentDuration > 0 {
				// preceding lines
				if entry.Timestamp.Add(adjacent.AdjacentDuration).After(firstMatchTime) &&
					entry.Timestamp.Add(-adjacent.AdjacentDuration).Before(lastMatchTime) {

					match = true
				}
			}

			if !match {
				lineNum++
				continue
			}
			resultLine := fmt.Sprintf(
				`{"node": "%s", "line": %d, `, logFile.Name, lineNum) +
				string(line)[1:]

			if !match {
				continue
			}

			// prepend or append
			if lineNum < firstMatchLine {
				before = append(before, resultLine)
			} else {
				after = append(after, resultLine)
			}

			lineNum++
		}

		results = slices.Concat(before, results, after)
	}

	// parse query TODO extract
	if filters.Query != "" {
		parsedQuery, err := gojq.Parse("select( " + filters.Query + ")")
		if err != nil {
			return nil, err
		}
		code, err := gojq.Compile(parsedQuery)
		if err != nil {
			return nil, err
		}
		resQuery := make([]string, 0, len(results))
		for _, line := range results {
			var input any
			if err := json.Unmarshal([]byte(line), &input); err != nil {
				return nil, fmt.Errorf("parsing log line: %w", err)
			}
			_, ok := code.Run(input).Next()
			if !ok {
				continue
			}

			resQuery = append(resQuery, line)

		}
		results = resQuery
	}

	// add flightrec times TODO extract
	if flightrecTimes && frDuration > 0 && !frLogTime.IsZero() {
		for i, line := range results {
			var entry LogLine
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				return nil, fmt.Errorf("parsing log line: %w", err)
			}
			flightTime := frDuration - frLogTime.Sub(entry.Timestamp)
			if flightTime < 0 {
				continue
			}
			results[i] = fmt.Sprintf(`{"flight_time": "%.3fs", %s`, flightTime.Seconds(), line[1:])
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading log: %w", err)
	}

	return results, nil
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
