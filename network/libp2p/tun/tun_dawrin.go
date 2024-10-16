//go:build darwin
// +build darwin

package tun

import (
	"fmt"
	"os/exec"
	"sync"

	"github.com/songgao/water"
)

// used to synchronize access to underlying OS resources
var rw = &sync.RWMutex{}

// New creates and returns a new TUN interface for the application.
func New(name string, opts ...Option) (*TUN, error) {
	// Setup TUN Config
	cfg := water.Config{
		DeviceType: water.TUN,
	}

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

// SetMTU sets the Maximum Tansmission Unit Size for a
// Packet on the interface.
func (t *TUN) setMTU(mtu int) error {
	return ifconfig(t.Iface.Name(), "mtu", fmt.Sprintf("%d", mtu))
}

// SetDestAddress sets the interface's address.
func (t *TUN) setAddress(address string) error {
	t.Src = address
	return nil
}

// SetDestAddress sets the interface's address.
func (t *TUN) setDestAddress(address string) error {
	t.Dst = address
	return nil
}

// Up brings up an interface to allow it to start accepting connections.
func (t *TUN) Up() error {
	return ifconfig(t.Iface.Name(), "inet", t.Src, t.Dst, "up")
}

// Down brings down an interface stopping active connections.
func (t *TUN) Down() error {
	return ifconfig(t.Iface.Name(), "down")
}

// Delete removes a TUN device from the host.
func Delete(name string) error {
	// return fmt.Errorf("removing an interface is unsupported under mac")
	return nil
}

func ifconfig(args ...string) error {
	defer rw.Unlock()
	rw.Lock()
	cmd := exec.Command("ifconfig", args...)
	fmt.Println(cmd.String())
	return cmd.Run()
}
