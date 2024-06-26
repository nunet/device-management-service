package models

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

// NetStat is a stub. Please expand it or completely change it based on requirements.
type NetStat struct {
	Status string `json:"status"` // Network status
	Info   string `json:"info"`   // Network information
}

// MessageInfo is a stub. Please expand it or completely change it based on requirements.
type MessageInfo struct {
	Info string `json:"info"` // Message information
}
