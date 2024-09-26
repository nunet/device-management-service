package types

// BlockchainAddressPrivKey holds Ethereum wallet address and private key from which the
// address is derived.
type BlockchainAddressPrivKey struct {
	Address    string `json:"address,omitempty"`
	PrivateKey string `json:"private_key,omitempty"`
	Mnemonic   string `json:"mnemonic,omitempty"`
}

// CapacityForNunet is a struct required in request body for the onboarding
type CapacityForNunet struct {
	Memory            uint64  `json:"memory,omitempty"`
	CPU               int64   `json:"cpu,omitempty"`
	NTXPricePerMinute float64 `json:"ntx_price,omitempty"`
	PaymentAddress    string  `json:"payment_addr,omitempty"`
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

// OnboardingConfig - parameters to configure onboarding
type OnboardingConfig struct {
	BaseDBModel
	Name               string             `json:"name,omitempty"`
	UpdateTimestamp    int64              `json:"update_timestamp,omitempty"`
	TotalResources     MachineResources   `json:"total_resources,omitempty" gorm:"foreignKey:ID"`
	OnboardedResources OnboardedResources `json:"onboarded_resources,omitempty" gorm:"foreignKey:ID"`
	PublicKey          string             `json:"public_key,omitempty"`
	Dashboard          string             `json:"dashboard,omitempty"`
	NTXPricePerMinute  float64            `json:"ntx_price,omitempty"`
}

type OnboardingStatus struct {
	Onboarded        bool             `json:"onboarded"`
	Error            error            `json:"error"`
	MachineUUID      string           `json:"machine_uuid"`
	WorkDir          string           `json:"work_dir"`
	DatabasePath     string           `json:"database_path"`
	OnboardingConfig OnboardingConfig `json:"onboarding_config"`
}
