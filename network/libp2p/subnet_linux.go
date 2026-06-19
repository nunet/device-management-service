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
	"strings"

	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils/sys"
)

func (l *Libp2p) MapPort(req types.MapPortRequest) error {
	s, ok := l.subnets[req.SubnetID]
	if !ok {
		return fmt.Errorf("subnet with ID %s does not exist", req.SubnetID)
	}

	protocol := strings.ToLower(req.Protocol)

	// Check if port already mapped (with lock)
	s.portMappingMx.Lock()
	if _, ok := s.portMapping[req.SubnetPort]; ok {
		s.portMappingMx.Unlock()
		return fmt.Errorf("port %s is already mapped", req.SubnetPort)
	}
	s.portMappingMx.Unlock()

	// TODO: check if any rules for the port already exists
	entry := &portMapEntry{
		subnetPort:   req.SubnetPort,
		execPort:     req.ExecutionPort,
		subnetIP:     req.SubnetIP,
		protocol:     protocol,
		executorType: req.ExecutorType,
		cniBridge:    req.CNIBridge,
	}

	var err error
	switch req.ExecutorType {
	case types.ExecutorTypeContainerd:
		tunName, tunErr := s.tunNameForIP(req.SubnetIP)
		if tunErr != nil {
			return tunErr
		}
		entry.tunIface = tunName

		bridge := req.CNIBridge
		if bridge == "" {
			bridge = "cni-nunet0"
		}
		entry.cniBridge = bridge

		// requests come from the vpn subnet
		sip, err := sys.IPToCIDR(req.SubnetIP, 24)
		if err != nil {
			return err
		}
		err = sys.AddForwardCNIRule(protocol, sip, "10.68.63.0/24", req.ExecutionPort, tunName, bridge)
		if err != nil {
			return err
		}
	default:
		err = sys.AddForwardRule(protocol, req.SubnetIP, req.SubnetPort)
		if err != nil {
			return err
		}
	}

	loIface, err := sys.GetNetInterfaceByFlags(net.FlagLoopback)
	if err != nil {
		log.Errorf("failed to get loopback interface: %v", err)
		log.Warnf("port %s will not be mapped to localhost:%s", req.ExecutionPort, req.SubnetIP, req.SubnetPort)
	} else {
		err = sys.AddOutputNatRule(protocol, req.SubnetIP, req.SubnetPort, loIface.Name)
		if err != nil {
			return err
		}
	}

	// Store mapping (with lock)
	s.portMappingMx.Lock()
	s.portMapping[req.SubnetPort] = entry
	s.portMappingMx.Unlock()

	return nil
}

func (l *Libp2p) UnmapPort(req types.MapPortRequest) error {
	s, ok := l.subnets[req.SubnetID]
	if !ok {
		return fmt.Errorf("subnet with ID %s does not exist", req.SubnetID)
	}

	protocol := strings.ToLower(req.Protocol)

	// Get and validate mapping (with lock)
	s.portMappingMx.Lock()
	mapping, ok := s.portMapping[req.SubnetPort]
	if !ok {
		s.portMappingMx.Unlock()
		return fmt.Errorf("port %s is not mapped", req.SubnetPort)
	}

	if mapping.subnetIP != req.SubnetIP || mapping.subnetPort != req.SubnetPort {
		s.portMappingMx.Unlock()
		return fmt.Errorf("port %s is not mapped to %s:%s", req.ExecutionPort, req.SubnetIP, req.SubnetPort)
	}
	s.portMappingMx.Unlock()

	if mapping.protocol != "" {
		protocol = mapping.protocol
	}

	err := sys.DelDNATRule(protocol, req.ExecutionPort, req.SubnetIP, req.SubnetPort)
	if err != nil {
		return err
	}

	switch mapping.executorType {
	case types.ExecutorTypeContainerd:
		bridge := mapping.cniBridge
		if bridge == "" {
			bridge = req.CNIBridge
		}
		tunName := mapping.tunIface
		if tunName == "" {
			var tunErr error
			tunName, tunErr = s.tunNameForIP(req.SubnetIP)
			if tunErr != nil {
				return tunErr
			}
		}
		err = sys.DelForwardCNIRule(protocol, "0.0.0.0", mapping.subnetIP, mapping.execPort, bridge, tunName)
	default:
		err = sys.DelForwardRule(protocol, req.SubnetIP, req.SubnetPort)
	}
	if err != nil {
		return err
	}

	loIface, err := sys.GetNetInterfaceByFlags(net.FlagLoopback)
	if err != nil {
		log.Errorf("failed to get loopback interface: %v", err)
		log.Warnf("Unable to delete localhost OutputNat rule for %s:%s", req.SubnetIP, req.SubnetPort)
	} else {
		err = sys.DelOutputNatRule(protocol, req.SubnetIP, req.SubnetPort, loIface.Name)
		if err != nil {
			return err
		}
	}

	err = sys.DelMasqueradeRule()
	if err != nil {
		return err
	}

	// Delete mapping (with lock)
	s.portMappingMx.Lock()
	delete(s.portMapping, req.SubnetPort)
	s.portMappingMx.Unlock()

	log.Infof("port %s unmapped successfully", req.SubnetPort)

	return nil
}

func (s *subnet) tunNameForIP(ip string) (string, error) {
	s.mx.Lock()
	defer s.mx.Unlock()

	iface, ok := s.ifaces[ip]
	if !ok {
		return "", fmt.Errorf("no tun interface for subnet IP %s", ip)
	}
	return iface.tun.Name(), nil
}
