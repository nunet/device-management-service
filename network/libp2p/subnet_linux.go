//go:build linux
// +build linux

package libp2p

import (
	"fmt"

	"gitlab.com/nunet/device-management-service/lib/sys"
)

func (l *Libp2p) MapPort(subnetID, protocol, sourceIP, sourcePort, destIP, destPort string) error {
	s, ok := l.subnets[subnetID]
	if !ok {
		return fmt.Errorf("subnet with ID %s does not exist", subnetID)
	}

	if _, ok := s.portMapping[sourcePort]; ok {
		return fmt.Errorf("port %s is already mapped", sourcePort)
	}

	// TODO track the port so that we can unmap it when we tear down the subnet
	err := sys.AddDNATRule(protocol, sourceIP, sourcePort, destIP, destPort)
	if err != nil {
		return err
	}

	err = sys.AddForwardRule("tcp", destIP, destPort)
	if err != nil {
		return err
	}

	err = sys.AddMasqueradeRule()
	if err != nil {
		return err
	}

	s.portMapping[sourcePort] = &struct {
		destPort string
		destIP   string
		srcIP    string
	}{
		destPort: destPort,
		destIP:   destIP,
		srcIP:    sourceIP,
	}

	return nil
}

func (l *Libp2p) UnmapPort(subnetID, protocol, sourceIP, sourcePort, destIP, destPort string) error {
	s, ok := l.subnets[subnetID]
	if !ok {
		return fmt.Errorf("subnet with ID %s does not exist", subnetID)
	}

	mapping, ok := s.portMapping[sourcePort]
	if !ok {
		return fmt.Errorf("port %s is not mapped", sourcePort)
	}

	if mapping.destIP != destIP || mapping.destPort != destPort || mapping.srcIP != sourceIP {
		return fmt.Errorf("port %s is not mapped to %s:%s", sourcePort, destIP, destPort)
	}

	err := sys.DelDNATRule(protocol, sourceIP, sourcePort, destIP, destPort)
	if err != nil {
		return err
	}
	err = sys.DelForwardRule("tcp", destIP, destPort)
	if err != nil {
		return err
	}

	err = sys.DelMasqueradeRule()
	if err != nil {
		return err
	}

	delete(s.portMapping, sourcePort)

	return nil
}
