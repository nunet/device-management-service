package utils

import (
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDHCPGetRandomCIDR(t *testing.T) {
	blacklist := []string{"10.0.0.0/8", "172.16.0.0/12", "10.10.0.0/16"} // Example blacklist
	mask := 8
	cidr, err := GetRandomCIDR(mask, blacklist)
	require.NoError(t, err)

	require.NotContains(t, blacklist, cidr)

	cidrParts := strings.Split(cidr, "/")
	ipParts := strings.Split(cidrParts[0], ".")
	assert.Equal(t, strings.Join(ipParts[1:], "."), "0.0.0")

	mask = 16
	cidr, err = GetRandomCIDR(mask, blacklist)
	require.NoError(t, err)

	require.NotContains(t, blacklist, cidr)

	cidrParts = strings.Split(cidr, "/")
	ipParts = strings.Split(cidrParts[0], ".")
	assert.NotEqual(t, ipParts[1], "0")
	assert.Equal(t, strings.Join(ipParts[2:], "."), "0.0")

	// blacklist all possible /16 networks
	blacklist = []string{}
	mask = 16
	for i := 0; i < 256; i++ {
		for j := 0; j < 256; j++ {
			blacklist = append(blacklist, fmt.Sprintf("%d.%d.0.0/16", i, j))
		}
	}
	cidr, err = GetRandomCIDR(mask, blacklist)
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not find a CIDR after")
	require.Equal(t, "0.0.0.0/16", cidr)
}

func TestDHCPGetRandomCIDRInRange(t *testing.T) {
	blacklist := []string{"10.0.0.0/16", "10.20.0.0/16", "10.30.0.0/16", "10.200.0.0/16"} // Example blacklist
	mask := 16
	start, end := net.ParseIP("10.0.0.0"), net.ParseIP("10.255.255.255")
	cidr, err := GetRandomCIDRInRange(mask, start, end, blacklist)
	require.NoError(t, err)

	require.NotContains(t, blacklist, cidr)

	cidrParts := strings.Split(cidr, "/")
	ipParts := strings.Split(cidrParts[0], ".")
	assert.Equal(t, strings.Join(ipParts[2:], "."), "0.0")

	// blacklist all possible /16 networks
	mask = 16
	for i := 0; i < 256; i++ {
		for j := 0; j < 256; j++ {
			blacklist = append(blacklist, fmt.Sprintf("%d.%d.0.0/16", i, j))
		}
	}
	cidr, err = GetRandomCIDRInRange(mask, start, end, blacklist)
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not find a CIDR after")
	require.Equal(t, "0.0.0.0/16", cidr)

	// \24 mask
	blacklist = []string{"10.0.0.0/24", "10.20.0.0/24", "10.30.0.0/24", "10.200.0.0/24"} // Example blacklist
	mask = 24
	cidr, err = GetRandomCIDRInRange(mask, start, end, blacklist)
	require.NoError(t, err)

	require.NotContains(t, blacklist, cidr)

	cidrParts = strings.Split(cidr, "/")
	ipParts = strings.Split(cidrParts[0], ".")
	assert.NotEqual(t, ipParts[1], "0")
	assert.Equal(t, strings.Join(ipParts[3:], "."), "0")

	// blacklist all possible /24 networks
	mask = 24
	for i := 0; i < 256; i++ {
		for j := 0; j < 256; j++ {
			for z := 0; z < 256; z++ {
				blacklist = append(blacklist, fmt.Sprintf("%d.%d.%d.0/24", i, j, z))
			}
		}
	}
	cidr, err = GetRandomCIDRInRange(mask, start, end, blacklist)
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not find a CIDR after")
	require.Equal(t, "0.0.0.0/24", cidr)
}

func TestDHCPNextIP(t *testing.T) {
	ip := net.IP{10, 10, 2, 253}
	mask := 24
	next := nextIP(ip, mask)
	assert.Equal(t, "10.10.2.254", next.String())

	// no next ip on /24
	ip = net.IP{10, 10, 2, 254}
	mask = 24
	next = nextIP(ip, mask)
	assert.Nil(t, next)

	// next ip exists on /16
	ip = net.IP{10, 10, 2, 254}
	mask = 16 // now we can increase the 3rd octet
	next = nextIP(ip, mask)
	assert.Equal(t, "10.10.3.1", next.String())

	// no next ip on /16
	ip = net.IP{10, 10, 255, 254}
	mask = 16 // now we can increase the 3rd octet
	next = nextIP(ip, mask)
	assert.Nil(t, next)

	// next ip exists on /8
	ip = net.IP{10, 10, 255, 254}
	mask = 8 // now we can increase the 2nd octet
	next = nextIP(ip, mask)
	assert.Equal(t, "10.11.0.1", next.String())

	// no next ip on /8
	ip = net.IP{10, 255, 255, 254}
	mask = 8 // now we can increase the 2nd octet
	next = nextIP(ip, mask)
	assert.Nil(t, next)

	// next ip exists on /0
	ip = net.IP{10, 255, 255, 254}
	mask = 0 // now we can increase the 2nd octet
	next = nextIP(ip, mask)
	assert.Equal(t, "11.0.0.1", next.String())

	// no next ip on /0
	ip = net.IP{255, 255, 255, 254}
	mask = 0 // now we can increase the 2nd octet
	next = nextIP(ip, mask)
	assert.Nil(t, next)
}

func TestDHCPGetNextIp(t *testing.T) {
	cidr := "10.10.2.0/24"
	usedIPs := map[string]bool{
		"10.10.2.1": true,
	}
	ip, err := GetNextIP(cidr, usedIPs)
	require.NoError(t, err)
	assert.Equal(t, "10.10.2.2", ip.String())

	usedIPs["10.10.2.2"] = true
	ip, err = GetNextIP(cidr, usedIPs)
	require.NoError(t, err)
	assert.Equal(t, "10.10.2.3", ip.String())

	// list of all ips on 10.10.10.0/24
	usedIPs = make(map[string]bool)
	for i := 1; i < 255; i++ {
		usedIPs[net.IP{10, 10, 10, byte(i)}.String()] = true
	}
	ip, err = GetNextIP("10.10.10.0/24", usedIPs)
	require.Nil(t, ip)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no available IPs in CIDR 10.10.10.0/24")

	// list of all ips on 10.10.0.0/16
	usedIPs = make(map[string]bool)
	for i := 0; i <= 255; i++ {
		for j := 1; j < 255; j++ {
			usedIPs[net.IP{10, 10, byte(i), byte(j)}.String()] = true
		}
	}
	ip, err = GetNextIP("10.10.0.0/16", usedIPs)
	require.Nil(t, ip)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no available IPs in CIDR 10.10.0.0/16")

	// list of all ips on 10.0.10.0/8
	usedIPs = make(map[string]bool)
	for x := 0; x <= 255; x++ {
		for i := 0; i <= 255; i++ {
			for j := 1; j < 255; j++ {
				usedIPs[net.IP{10, byte(x), byte(i), byte(j)}.String()] = true
			}
		}
	}
	ip, err = GetNextIP("10.0.0.0/8", usedIPs)
	require.Nil(t, ip)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no available IPs in CIDR 10.0.0.0/8")
}
