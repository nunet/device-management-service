package types

const (
	NetP2P = "p2p"
)

// NetworkSpec is a stub. Please expand based on requirements.
type NetworkSpec struct {
}

// NetConfig is a stub. Please expand it or completely change it based on requirements.
type NetConfig struct {
	NetworkSpec SpecConfig `json:"network_spec"` // Network specification
}

func (nc *NetConfig) GetNetworkConfig() *SpecConfig {
	return &nc.NetworkSpec
}

// NetworkStats should contain all network info the user is interested in.
// for now there's only peerID and listening address but reachability, local and remote addr etc...
// can be added when necessary.
type NetworkStats struct {
	ID         string `json:"id"`
	ListenAddr string `json:"listen_addr"`
}

// MessageInfo is a stub. Please expand it or completely change it based on requirements.
type MessageInfo struct {
	Info string `json:"info"` // Message information
}
