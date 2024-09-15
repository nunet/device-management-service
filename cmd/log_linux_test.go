//go:build linux
// +build linux

package cmd

import (
	"bytes"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/coreos/go-systemd/sdjournal"
	"github.com/stretchr/testify/assert"
)

type MockLogger struct {
	errRead   uint64
	nextEntry uint64
	entries   []sdjournal.JournalEntry
}

func (ml *MockLogger) AddMatch(match string) error {
	expected := fmt.Sprintf("_SYSTEMD_UNIT=%s", dmsUnit)

	if match != expected {
		return fmt.Errorf("invalid match for systemd unit")
	}

	return nil
}

func (ml *MockLogger) Close() error {
	return nil
}

func (ml *MockLogger) GetEntry() (*sdjournal.JournalEntry, error) {
	if ml.errRead == (ml.nextEntry - 2) {
		return &sdjournal.JournalEntry{}, fmt.Errorf("entry corrupted: unable to read")
	}

	return &ml.entries[(ml.nextEntry - 2)], nil
}

func (ml *MockLogger) Next() (uint64, error) {
	// EOF
	if ml.nextEntry > uint64(len(ml.entries)) {
		return 0, nil
	}

	ml.nextEntry++
	return 1, nil
}

func Test_LogLinuxCmdSuccess(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	/*  ENTRY is a struct with 3 fields
	One of the fields of ENTRY is "Fields"
	which is a string pair map

	There's additional fields of ENTRY like:
	RealtimeTimestamp

	So, a journal has many entries and each entry
	has some fields. One of the fields, "Fields", is
	a map containing many key-value pairs.
	*/
	logs := []map[string]string{
		{
			"_BOOT_ID": "1",
			"MESSAGE":  "some info",
		},
		{
			"_BOOT_ID": "2",
			"MESSAGE":  "some other info",
		},
		{
			"_BOOT_ID": "3",
			"MESSAGE":  "some other other info",
		},
	}

	entries := make([]sdjournal.JournalEntry, len(logs))
	for _, entry := range entries {
		for _, log := range logs {
			entry.Fields = log
		}
	}

	mockJournal := &MockLogger{
		nextEntry: 1,
		entries:   entries,
	}

	cmd := newLogCmd(ts.afs, mockJournal)

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	assert.NoError(t, err)

	expected := filepath.Join(logDir, tarGzName)
	assert.Contains(t, buf.String(), expected)
}

func Test_LogLinuxCmdNoEntries(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	entries := make([]sdjournal.JournalEntry, 0)
	mockJournal := &MockLogger{
		nextEntry: 1,
		entries:   entries,
	}

	cmd := newLogCmd(ts.afs, mockJournal)

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	assert.Error(t, err)

	assert.Contains(t, buf.String(), "no log entries")
}

func Test_LogLinuxCmdReadError(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	logs := []map[string]string{
		{
			"_BOOT_ID": "1",
			"MESSAGE":  "some info",
		},
		{
			"_BOOT_ID": "2",
			"MESSAGE":  "some other info",
		},
		{
			"_BOOT_ID": "3",
			"MESSAGE":  "some other other info",
		},
	}

	entries := make([]sdjournal.JournalEntry, len(logs))
	for _, entry := range entries {
		for _, log := range logs {
			entry.Fields = log
		}
	}

	mockJournal := &MockLogger{
		errRead:   2,
		nextEntry: 1,
		entries:   entries,
	}

	cmd := newLogCmd(ts.afs, mockJournal)

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	assert.NoError(t, err)

	expected := "entry corrupted: unable to read"
	assert.Contains(t, buf.String(), expected)
}

func Test_LogLinuxCmdNoMessageField(t *testing.T) {
	ts := NewTestSuite()
	assert.NoError(t, ts.setup())
	defer ts.teardown()

	logs := []map[string]string{
		{
			"_BOOT_ID": "1",
			"MESSAGE":  "some info",
		},
		{
			"_BOOT_ID": "2",
		},
		{
			"_BOOT_ID": "3",
			"MESSAGE":  "some other other info",
		},
	}

	entries := make([]sdjournal.JournalEntry, len(logs))
	for _, entry := range entries {
		for _, log := range logs {
			entry.Fields = log
		}
	}

	mockJournal := &MockLogger{
		nextEntry: 1,
		entries:   entries,
	}

	cmd := newLogCmd(ts.afs, mockJournal)

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	assert.NoError(t, err)

	expected := "no message field in entry"
	assert.Contains(t, buf.String(), expected)
}
