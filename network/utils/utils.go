// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package utils

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"golang.org/x/exp/rand"
)

// GetNextIP returns the next available IP in the CIDR range
func GetNextIP(cidr string, usedIPs map[string]bool) (net.IP, error) {
	cidrParts := strings.Split(cidr, "/")
	if len(cidrParts) != 2 {
		return nil, fmt.Errorf("invalid CIDR %s", cidr)
	}

	mask, err := strconv.Atoi(cidrParts[1])
	if err != nil {
		return nil, err
	}
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	networkMask := ipnet.IP.Mask(ipnet.Mask)
	firstHostIP := net.IP{networkMask[0], networkMask[1], networkMask[2], networkMask[3] + byte(1)}

	for ip := firstHostIP; ipnet.Contains(ip); ip = nextIP(ip, mask) {
		if ip == nil {
			break
		}
		if !usedIPs[ip.String()] {
			return ip, nil
		}
	}

	return nil, fmt.Errorf("no available IPs in CIDR %s", cidr)
}

// nextIP returns the next available IP in the network
func nextIP(ip net.IP, netmask int) net.IP {
	if ip4 := ip.To4(); ip4 != nil {
		if netmask == 0 && ip4[0] == 255 && ip4[1] == 255 && ip4[2] == 255 && ip4[3] == 254 {
			return nil // no more IPs for this network
		}

		if netmask == 0 && ip4[1] == 255 && ip4[2] == 255 && ip4[3] == 254 {
			ip4[0]++
			ip4[1] = 0
			ip4[2] = 0
			ip4[3] = 0
		}

		if netmask == 8 && ip[1] == 255 && ip4[2] == 255 && ip4[3] == 254 {
			return nil // no more IPs for this network
		}

		if netmask == 8 && ip[1] < 255 && ip4[2] == 255 && ip4[3] == 254 {
			ip4[1]++
			ip4[2] = 0
			ip4[3] = 0
		}

		if netmask == 16 && ip4[2] == 255 && ip4[3] == 254 {
			return nil // no more IPs for this network
		}

		if (netmask == 16 || netmask == 8) && ip[2] < 255 && ip4[3] == 254 {
			ip4[2]++
			ip4[3] = 0
		}

		if netmask == 24 && ip4[3] == 254 {
			return nil // no more IPs for this network
		}

		ip4[3]++
		return ip4
	}
	return nil
}

// GetRandomCIDR returns a random CIDR with the given mask
// and not in the blacklist.
// If the blacklist is empty, it will return a random CIDR with the given mask.
// This function supports mask 0, 8, 16, 24.
// If you need more elaborate masks to get more subnets (i.e: 0<mask<32)
// refactor this to use bitwise operations on the IP.
func GetRandomCIDR(mask int, blacklist []string) (string, error) {
	var cidr string
	var breakCounter int
	for {
		if mask > 0 && breakCounter > 2^mask || mask == 0 && breakCounter > 255 {
			return fmt.Sprintf("%s/%d", "0.0.0.0", mask), fmt.Errorf("could not find a CIDR after %d attempts", breakCounter)
		}

		ip := net.IP{byte(rand.Intn(256)), byte(rand.Intn(256)), byte(rand.Intn(256)), byte(rand.Intn(256))}

		candidate := fmt.Sprintf("%s/%d", ip, mask)
		if !isOnBlacklist(candidate, blacklist) {
			cidr = candidate
			break
		}
		breakCounter++
	}

	_, ip, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", nil
	}

	return ip.String(), nil
}

// isOnBlacklist returns true if the given CIDR is in the blacklist.
func isOnBlacklist(cidr string, blacklist []string) bool {
	for _, blacklistedCIDR := range blacklist {
		_, blacklistedSubnet, err := net.ParseCIDR(blacklistedCIDR)
		if err != nil {
			return false // Ignore errors in blacklist for safety
		}
		_, subnet, err := net.ParseCIDR(cidr)
		if err != nil {
			return false // Ignore errors in generated CIDR for safety
		}
		if blacklistedSubnet.Contains(subnet.IP) {
			return true
		}
	}
	return false
}

func GetRandomCIDRInRange(mask int, start, end net.IP, blacklist []string) (string, error) {
	var cidr string
	var breakCounter int
	networkBitsIndex := mask / 8
	for {
		if mask > 0 && breakCounter > 2^mask || mask == 0 && breakCounter > 255 {
			return fmt.Sprintf("%s/%d", "0.0.0.0", mask), fmt.Errorf("could not find a CIDR after %d attempts", breakCounter)
		}

		ip := net.IP{byte(0), byte(0), byte(0), byte(0)}
		for i := 0; i < networkBitsIndex; i++ {
			ip[i] = byte(randRange(int(start.To4()[i]), int(end.To4()[i])))
		}

		candidate := fmt.Sprintf("%s/%d", ip, mask)
		if !isOnBlacklist(candidate, blacklist) {
			cidr = candidate
			break
		}
		breakCounter++
	}

	_, ip, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", nil
	}

	return ip.String(), nil
}

func randRange(min, max int) int {
	return rand.Intn(max-min+1) + min
}

// IsFreePort checks if a given port is free to use by trying to listen on it.
func IsFreePort(port int) bool {
	addr := net.JoinHostPort("", strconv.Itoa(port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		// error listening, probably in use
		return false
	}

	_ = ln.Close()
	return true
}
