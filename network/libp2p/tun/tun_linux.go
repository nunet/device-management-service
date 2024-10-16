//go:build linux
// +build linux

package tun

import (
	"errors"
	"net"
	"sync"

	"github.com/songgao/water"
	"github.com/vishvananda/netlink"
)

// used to synchronize access to underlying OS resources
var rw = &sync.RWMutex{}

// New creates and returns a new TUN interface for the application.
func New(name string, opts ...Option) (*TUN, error) {
	// Setup TUN Config
	cfg := water.Config{
		DeviceType: water.TUN,
	}
	cfg.Name = name

	defer rw.Unlock()
	rw.Lock()
	// Create Water Interface
	iface, err := water.New(cfg)
	if err != nil {
		return nil, err
	}

	// Create TUN result struct
	result := TUN{
		Iface: iface,
	}

	// Apply options to set TUN config values
	err = result.Apply(opts...)
	return &result, err
}

// setMTU sets the Maximum Tansmission Unit Size for a
// Packet on the interface.
//
// Important: do not lock rw here, it is locked in the calling function
// as this function is being called by New when applying options
func (t *TUN) setMTU(mtu int) error {
	link, err := netlink.LinkByName(t.Iface.Name())
	if err != nil {
		return err
	}
	return netlink.LinkSetMTU(link, mtu)
}

// setDestAddress sets the interface's destination address and subnet.
//
// Important: do not lock rw here, it is locked in the calling function
// as this function is being called by New when applying options
func (t *TUN) setAddress(address string) error {
	addr, err := netlink.ParseAddr(address)
	if err != nil {
		return err
	}
	link, err := netlink.LinkByName(t.Iface.Name())
	if err != nil {
		return err
	}
	return netlink.AddrAdd(link, addr)
}

// SetDestAddress isn't supported under Linux.
// You should instead use set address to set the interface to handle
// all addresses within a subnet.
//
// Important: do not lock rw here, it is locked in the calling function
// as this function is being called by New when applying options
func (t *TUN) setDestAddress(_ string) error { //nolint
	return errors.New("destination addresses are not supported under linux")
}

func (t *TUN) addRoute(network net.IPNet) error { //nolint
	defer rw.Unlock()
	rw.Lock()

	link, err := netlink.LinkByName(t.Iface.Name())
	if err != nil {
		return err
	}
	return netlink.RouteAdd(&netlink.Route{
		LinkIndex: link.Attrs().Index,
		Dst:       &network,
		Priority:  3000,
	})
}

func (t *TUN) delRoute(network net.IPNet) error { //nolint
	defer rw.Unlock()
	rw.Lock()

	link, err := netlink.LinkByName(t.Iface.Name())
	if err != nil {
		return err
	}
	return netlink.RouteDel(&netlink.Route{
		LinkIndex: link.Attrs().Index,
		Dst:       &network,
		Priority:  3000,
	})
}

// Up brings up an interface to allow it to start accepting connections.
func (t *TUN) Up() error {
	defer rw.Unlock()
	rw.Lock()

	link, err := netlink.LinkByName(t.Iface.Name())
	if err != nil {
		return err
	}
	return netlink.LinkSetUp(link)
}

// Down brings down an interface stopping active connections.
func (t *TUN) Down() error {
	defer rw.Unlock()
	rw.Lock()

	link, err := netlink.LinkByName(t.Iface.Name())
	if err != nil {
		return err
	}
	return netlink.LinkSetDown(link)
}

// Delete removes a TUN device from the host.
func Delete(name string) error {
	defer rw.Unlock()
	rw.Lock()

	link, err := netlink.LinkByName(name)
	if err != nil {
		return err
	}
	return netlink.LinkDel(link)
}
