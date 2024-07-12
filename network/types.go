// Package network provides a simple network interface for the rest of the DMS.
package network

type VPN interface {
	// Start takes in an initial routing table and starts the VPN.
	Start() error

	// AddPeer is for adding the peers after the VPN has been started.
	// This should also update the routing table with the new peer.
	AddPeer() error

	// RemovePeer is oppisite of AddPeer. It should also update the routing table.
	RemovePeer() error

	// Stop tears down the VPN.
	Stop() error
}
