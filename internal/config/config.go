package config

type Config struct {
	General   `mapstructure:"general" json:"general"`
	Rest      `mapstructure:"rest" json:"rest"`
	P2P       `mapstructure:"p2p" json:"p2p"`
	Job       `mapstructure:"job" json:"job"`
	Telemetry `mapstructure:"telemetry" json:"telemetry"`
}

type General struct {
	UserDir string `mapstructure:"user_dir" json:"user_dir"`
	WorkDir string `mapstructure:"work_dir" json:"work_dir"`
	DataDir string `mapstructure:"data_dir" json:"data_dir"`
	Debug   bool   `mapstructure:"debug" json:"debug"`
}

type Rest struct {
	Addr string `mapstructure:"addr" json:"addr"`
	Port uint32 `mapstructure:"port" json:"port"`
}

type P2P struct {
	ListenAddress  []string `mapstructure:"listen_address" json:"listen_address"`
	BootstrapPeers []string `mapstructure:"bootstrap_peers" json:"bootstrap_peers"`
}

type Job struct {
	LogUpdateInterval int    `mapstructure:"log_update_interval" json:"log_update_interval"` // in minutes
	TargetPeer        string `mapstructure:"target_peer" json:"target_peer"`                 // specific peer to send deployment requests to - XXX probably not a good idea. Remove after testing stage.
	CleanupInterval   int    `mapstructure:"cleanup_interval" json:"cleanup_interval"`       // docker container and images clean up interval in days
}

type Telemetry struct {
	ServiceName        string `mapstructure:"service_name" json:"service_name"`
	GlobalEndpoint     string `mapstructure:"global_endpoint" json:"global_endpoint"`
	ObservabilityLevel string `mapstructure:"observability_level" json:"observability_level"`
	TelemetryMode      string `mapstructure:"telemetry_mode" json:"telemetry_mode"`
}
