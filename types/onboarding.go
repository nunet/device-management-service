package types

// OnboardingConfig - parameters to configure onboarding
type OnboardingConfig struct {
	BaseDBModel
	Name string `json:"name,omitempty"`

	PublicKey string `json:"public_key,omitempty"`

	Dashboard         string  `json:"dashboard,omitempty"`
	NTXPricePerMinute float64 `json:"ntx_price,omitempty"`

	// These are not stored in the database, but are part of the onboarding config
	// during the get onboarding config call these are populated from the resource manager and hardware
	OnboardedResources Resources `json:"resources,omitempty" gorm:"-"`
	MachineResources   Resources `json:"machine_resources,omitempty" gorm:"-"`
}

type OnboardingStatus struct {
	Onboarded        bool             `json:"onboarded"`
	Error            error            `json:"error"`
	MachineUUID      string           `json:"machine_uuid"`
	WorkDir          string           `json:"work_dir"`
	DatabasePath     string           `json:"database_path"`
	OnboardingConfig OnboardingConfig `json:"onboarding_config"`
}
