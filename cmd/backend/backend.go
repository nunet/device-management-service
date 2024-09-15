package backend

import (
	gonet "github.com/shirou/gopsutil/net"
)

// NetworkManager abstracts connection on ports
type NetworkManager interface {
	GetConnections(kind string) ([]gonet.ConnectionStat, error)
}
