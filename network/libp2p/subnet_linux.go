// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build linux
// +build linux

package libp2p

import (
	"fmt"
	"net"

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

	// TODO: check if any rules for the port already exists

	// TODO track the port so that we can unmap it when we tear down the subnet
	err := sys.AddDNATRule(protocol, sourcePort, destIP, destPort)
	if err != nil {
		return err
	}

	err = sys.AddForwardRule(protocol, destIP, destPort)
	if err != nil {
		return err
	}

	loIface, err := sys.GetNetInterfaceByFlags(net.FlagLoopback)
	if err != nil {
		log.Errorf("failed to get loopback interface: %v", err)
		log.Warnf("port %s will not be mapped to localhost:%s", sourcePort, destIP, destPort)
	} else {
		err = sys.AddOutputNatRule(protocol, destIP, destPort, loIface.Name)
		if err != nil {
			return err
		}
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

	err := sys.DelDNATRule(protocol, sourcePort, destIP, destPort)
	if err != nil {
		return err
	}
	err = sys.DelForwardRule(protocol, destIP, destPort)
	if err != nil {
		return err
	}

	loIface, err := sys.GetNetInterfaceByFlags(net.FlagLoopback)
	if err != nil {
		log.Errorf("failed to get loopback interface: %v", err)
		log.Warnf("Unable to delete localhost OutputNat rule for %s:%s", destIP, destPort)
	} else {
		err = sys.DelOutputNatRule(protocol, destIP, destPort, loIface.Name)
		if err != nil {
			return err
		}
	}

	err = sys.DelMasqueradeRule()
	if err != nil {
		return err
	}

	delete(s.portMapping, sourcePort)

	log.Infof("port %s unmapped successfully", sourcePort)

	return nil
}
