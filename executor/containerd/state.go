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
	stdout              bytes.Buffer
	stderr              bytes.Buffer

	resultMu sync.RWMutex
	result   *types.ExecutionResult

	running *atomic.Bool
	doneCh  chan struct{}
}

const (
	stdoutLogFile = "stdout.log"
	stderrLogFile = "stderr.log"
)

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
		"stdout", filepath.Join(resultsDir, stdoutLogFile),
		"stderr", filepath.Join(resultsDir, stderrLogFile),
	)
	return nil
}

func (s *executionState) logWriters() (stdout, stderr io.Writer) {
	if s.stdoutFile != nil {
		stdout = s.stdoutFile
	} else {
		stdout = &s.stdout
	}
	if s.stderrFile != nil {
		stderr = s.stderrFile
	} else {
		stderr = &s.stderr
	}
	return stdout, stderr
}

func (s *executionState) closeLogFiles() {
	if s.stdoutFile != nil {
		if err := s.stdoutFile.Close(); err != nil {
			log.Warnw("failed to close stdout log file", "error", err)
		}
		s.stdoutFile = nil
	}
	if s.stderrFile != nil {
		if err := s.stderrFile.Close(); err != nil {
			log.Warnw("failed to close stderr log file", "error", err)
		}
		s.stderrFile = nil
	}
}

func (s *executionState) readLogs() (string, string) {
	if s.resultsDir == "" {
		return s.stdout.String(), s.stderr.String()
	}

	var stdout, stderr string
	if b, err := os.ReadFile(filepath.Join(s.resultsDir, stdoutLogFile)); err == nil {
		stdout = string(b)
	} else {
		log.Warnw("failed to read stdout log file", "error", err)
	}
	if b, err := os.ReadFile(filepath.Join(s.resultsDir, stderrLogFile)); err == nil {
		stderr = string(b)
	} else {
		log.Warnw("failed to read stderr log file", "error", err)
	}
	return stdout, stderr
}

func (s *executionState) combinedLogs() string {
	stdout, stderr := s.readLogs()
	// TODO weak combination
	return stdout + stderr
}

func (s *executionState) scheduleLogDeletion(after time.Duration) {
	if s.resultsDir == "" || after <= 0 {
		return
	}

	resultsDir := s.resultsDir
	go func() {
		time.Sleep(after)
		if err := os.RemoveAll(resultsDir); err != nil {
			log.Errorw("failed to remove execution log files", "resultsDir", resultsDir, "error", err)
		}
	}()
}
