//go:build darwin
// +build darwin

package libp2p

import (
	"fmt"
)

func (l *Libp2p) MapPort(subnetID, protocol, sourceIP, sourcePort, destIP, destPort string) error {
	// TODO track the port so that we can unmap it when we tear down the subnet
	return fmt.Errorf("TODO MapPort darwin")
}

func (l *Libp2p) UnmapPort(subnetID, protocol, sourceIP, sourcePort, destIP, destPort string) error {
	// TODO
	return fmt.Errorf("TODO UnmapPort darwin")
}
