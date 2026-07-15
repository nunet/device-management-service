// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package containerd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/containerd/containerd/v2/client"
	"gitlab.com/nunet/device-management-service/types"
)

type executionState struct {
	executionID string
	container   client.Container
	task        client.Task
	network     *networkSetup
	image       string

	resultsDir          string
	persistLogsDuration time.Duration
	stdoutFile          *os.File
	stderrFile          *os.File
	stdout              syncLogBuffer
	stderr              syncLogBuffer

	resultMu sync.RWMutex
	result   *types.ExecutionResult

	running *atomic.Bool
	doneCh  chan struct{}
}

const (
	stdoutLogFile = "stdout.log"
	stderrLogFile = "stderr.log"

	logStreamPollInterval = 100 * time.Millisecond
)

// syncLogBuffer is a mutex-protected buffer used for concurrent container log writes and log stream reads
type syncLogBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncLogBuffer) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncLogBuffer) WriteString(s string) (int, error) {
	return b.Write([]byte(s))
}

func (b *syncLogBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func (s *executionState) setResult(result *types.ExecutionResult) {
	s.resultMu.Lock()
	defer s.resultMu.Unlock()
	s.result = result
}

func (s *executionState) getResult() *types.ExecutionResult {
	s.resultMu.RLock()
	defer s.resultMu.RUnlock()
	return s.result
}

func (s *executionState) openLogFiles(resultsDir string) error {
	if err := os.MkdirAll(resultsDir, 0o755); err != nil {
		return fmt.Errorf("create results directory: %w", err)
	}

	var err error
	s.stdoutFile, err = os.OpenFile(
		filepath.Join(resultsDir, stdoutLogFile),
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0o644,
	)
	if err != nil {
		return fmt.Errorf("open stdout log file: %w", err)
	}

	s.stderrFile, err = os.OpenFile(
		filepath.Join(resultsDir, stderrLogFile),
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0o644,
	)
	if err != nil {
		_ = s.stdoutFile.Close()
		s.stdoutFile = nil
		return fmt.Errorf("open stderr log file: %w", err)
	}

	s.resultsDir = resultsDir
	log.Debugw("execution log files opened",
		"executionID", s.executionID,
		"stdout", filepath.Join(resultsDir, stdoutLogFile),
		"stderr", filepath.Join(resultsDir, stderrLogFile),
	)
	return nil
}

func (s *executionState) logWriters() (stdout, stderr io.Writer) {
	if s.stdoutFile != nil {
		stdout = io.MultiWriter(s.stdoutFile, &s.stdout)
	} else {
		stdout = &s.stdout
	}
	if s.stderrFile != nil {
		stderr = io.MultiWriter(s.stderrFile, &s.stderr)
	} else {
		stderr = &s.stderr
	}
	return stdout, stderr
}

func (s *executionState) closeLogFiles() {
	if s.stdoutFile != nil {
		if err := s.stdoutFile.Close(); err != nil {
			log.Warnw("failed to close stdout log file",
				"executionID", s.executionID,
				"error", err,
			)
		}
		s.stdoutFile = nil
	}
	if s.stderrFile != nil {
		if err := s.stderrFile.Close(); err != nil {
			log.Warnw("failed to close stderr log file",
				"executionID", s.executionID,
				"error", err,
			)
		}
		s.stderrFile = nil
	}
}

func (s *executionState) readLogs() (string, string) {
	stdout := s.stdout.String()
	stderr := s.stderr.String()
	if stdout != "" || stderr != "" || s.resultsDir == "" {
		return stdout, stderr
	}

	// Fall back to on-disk logs when in-memory buffers are empty
	if b, err := os.ReadFile(filepath.Join(s.resultsDir, stdoutLogFile)); err == nil {
		stdout = string(b)
	} else {
		log.Warnw("failed to read stdout log file",
			"executionID", s.executionID,
			"error", err,
		)
	}
	if b, err := os.ReadFile(filepath.Join(s.resultsDir, stderrLogFile)); err == nil {
		stderr = string(b)
	} else {
		log.Warnw("failed to read stderr log file",
			"executionID", s.executionID,
			"error", err,
		)
	}
	return stdout, stderr
}

func (s *executionState) combinedLogs() string {
	stdout, stderr := s.readLogs()
	// TODO weak combination
	switch {
	case stdout == "":
		return stderr
	case stderr == "":
		return stdout
	default:
		return stdout + "\n" + stderr
	}
}

func (s *executionState) scheduleLogDeletion(after time.Duration) {
	if s.resultsDir == "" || after <= 0 {
		return
	}

	resultsDir := s.resultsDir
	executionID := s.executionID
	go func() {
		log.Debugw("scheduling execution log deletion",
			"executionID", executionID,
			"resultsDir", resultsDir,
			"after", after,
		)
		time.Sleep(after)
		if err := os.RemoveAll(resultsDir); err != nil {
			log.Errorw("failed to remove execution log files",
				"executionID", executionID,
				"resultsDir", resultsDir,
				"error", err,
			)
			return
		}
		log.Debugw("execution log files removed",
			"executionID", executionID,
			"resultsDir", resultsDir,
		)
	}()
}

type logStreamReader struct {
	ctx    context.Context
	state  *executionState
	follow bool
	offset int
}

func newLogStreamReader(ctx context.Context, state *executionState, follow bool) *logStreamReader {
	if ctx == nil {
		ctx = context.Background()
	}
	return &logStreamReader{
		ctx:    ctx,
		state:  state,
		follow: follow,
	}
}

func (r *logStreamReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	for {
		combined := r.state.combinedLogs()
		if r.offset < len(combined) {
			n := copy(p, combined[r.offset:])
			r.offset += n
			return n, nil
		}

		if !r.follow {
			return 0, io.EOF
		}

		if !r.state.running.Load() {
			return 0, io.EOF
		}

		select {
		case <-r.ctx.Done():
			return 0, r.ctx.Err()
		case <-time.After(logStreamPollInterval):
		}
	}
}

func (r *logStreamReader) Close() error {
	return nil
}
