package backend

import (
	"github.com/coreos/go-systemd/sdjournal"
	gonet "github.com/shirou/gopsutil/net"
)

// NetworkManager abstracts connection on ports
type NetworkManager interface {
	GetConnections(kind string) ([]gonet.ConnectionStat, error)
}

// Logger abstracts systemd journal entries
type Logger interface {
	AddMatch(match string) error
	Close() error
	GetEntry() (*sdjournal.JournalEntry, error)
	Next() (uint64, error)
}
