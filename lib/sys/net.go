package sys

import (
	"net"
	"strings"

	"github.com/songgao/water"
)

// types for tun tap
type TunTapMode int

const (
	NetTunMode TunTapMode = iota
	NetTapMode
)

// // TUN is a struct containing the fields necessary
// // to configure a system TUN device. Access the
// // internal TUN device through TUN.Iface
type NetInterface struct {
	Iface *water.Interface
	Src   string
	Dst   string
}

// GetNetInterfaces gets the list of network interfaces
func GetNetInterfaces() ([]net.Interface, error) {
	return net.Interfaces()
}

// GetNetInterfaceByName gets the network interface by name
func GetNetInterfaceByName(name string) (*net.Interface, error) {
	return net.InterfaceByName(name)
}

func GetUsedAddresses() ([]string, error) {
	ifaces, err := GetNetInterfaces()
	if err != nil {
		return nil, err
	}

	var networks []string
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			if !strings.Contains(addr.String(), ":") {
				networks = append(networks, addr.String())
			}
		}
	}

	return networks, nil
}
