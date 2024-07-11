package models

// BlockchainAddressPrivKey holds Ethereum wallet address and private key from which the
// address is derived.
type BlockchainAddressPrivKey struct {
	Address    string `json:"address,omitempty"`
	PrivateKey string `json:"private_key,omitempty"`
	Mnemonic   string `json:"mnemonic,omitempty"`
}

// CapacityForNunet is a struct required in request body for the onboarding
type CapacityForNunet struct {
	Memory            int64   `json:"memory,omitempty"`
	CPU               int64   `json:"cpu,omitempty"`
	NTXPricePerMinute float64 `json:"ntx_price,omitempty"`
	Channel           string  `json:"channel,omitempty"`
	PaymentAddress    string  `json:"payment_addr,omitempty"`
	Cardano           bool    `json:"cardano,omitempty"`
	ServerMode        bool    `json:"server_mode,omitempty,"`
	IsAvailable       bool    `json:"is_available"`
}

// Provisioned struct holds data about how much total resource
// host machine is equipped with
type Provisioned struct {
	CPU      float64 `json:"cpu,omitempty"`
	Memory   uint64  `json:"memory,omitempty"`
	NumCores uint64  `json:"total_cores,omitempty"`
}

// Metadata - machine metadata of onboarding parameters
type Metadata struct {
	Name            string `json:"name,omitempty"`
	UpdateTimestamp int64  `json:"update_timestamp,omitempty"`
	Resource        struct {
		MemoryMax int64 `json:"memory_max,omitempty"`
		TotalCore int64 `json:"total_core,omitempty"`
		CPUMax    int64 `json:"cpu_max,omitempty"`
	} `json:"resource,omitempty"`
	Available struct {
		CPU    int64 `json:"cpu,omitempty"`
		Memory int64 `json:"memory,omitempty"`
	} `json:"available,omitempty"`
	Reserved struct {
		CPU    int64 `json:"cpu,omitempty"`
		Memory int64 `json:"memory,omitempty"`
	} `json:"reserved,omitempty"`
	Network           string  `json:"network,omitempty"`
	PublicKey         string  `json:"public_key,omitempty"`
	NodeID            string  `json:"node_id,omitempty"`
	AllowCardano      bool    `json:"allow_cardano,omitempty"`
	GpuInfo           []Gpu   `json:"gpu_info,omitempty"`
	Dashboard         string  `json:"dashboard,omitempty"`
	NTXPricePerMinute float64 `json:"ntx_price,omitempty"`
}

type OnboardingStatus struct {
	Onboarded    bool   `json:"onboarded"`
	Error        error  `json:"error"`
	MachineUUID  string `json:"machine_uuid"`
	MetadataPath string `json:"metadata_path"`
	DatabasePath string `json:"database_path"`
}

type LogBinAuth struct {
    BaseDBModel
	PeerID      string `json:"peer_id"`
	MachineUUID string `json:"machine_uuid"`
	Token       string `json:"token"`
}
