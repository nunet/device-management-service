//go:build linux
// +build linux

package backend

import "github.com/coreos/go-systemd/sdjournal"

// Logger abstracts systemd journal entries
type Logger interface {
	AddMatch(match string) error
	Close() error
	GetEntry() (*sdjournal.JournalEntry, error)
	Next() (uint64, error)
}

type Journal struct {
	journal *sdjournal.Journal
}

func SetRealJournal(j *sdjournal.Journal) *Journal {
	return &Journal{journal: j}
}

func (j *Journal) AddMatch(match string) error {
	return j.journal.AddMatch(match)
}

func (j *Journal) Close() error {
	return j.journal.Close()
}

func (j *Journal) GetEntry() (*sdjournal.JournalEntry, error) {
	return j.journal.GetEntry()
}

func (j *Journal) Next() (uint64, error) {
	return j.journal.Next()
}
