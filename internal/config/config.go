package config

type Config struct {
	General   `mapstructure:"general"`
	Rest      `mapstructure:"rest"`
	P2P       `mapstructure:"p2p"`
	Job       `mapstructure:"job"`
	Telemetry `mapstructure:"telemetry"`
}

type General struct {
	WorkDir string `mapstructure:"work_dir"`
	DataDir string `mapstructure:"data_dir"`
	Debug   bool   `mapstructure:"debug"`
}

type Rest struct {
	Addr string `mapstructure:"addr"`
	Port uint32 `mapstructure:"port"`
}

type P2P struct {
	ListenAddress  []string `mapstructure:"listen_address"`
	BootstrapPeers []string `mapstructure:"bootstrap_peers"`
}

type Job struct {
	LogUpdateInterval int    `mapstructure:"log_update_interval"` // in minutes
	TargetPeer        string `mapstructure:"target_peer"`         // specific peer to send deployment requests to - XXX probably not a good idea. Remove after testing stage.
	CleanupInterval   int    `mapstructure:"cleanup_interval"`    // docker container and images clean up interval in days
}

type Telemetry struct {
	ServiceName        string `mapstructure:"service_name"`
	GlobalEndpoint     string `mapstructure:"global_endpoint"`
	ObservabilityLevel string `mapstructure:"observability_level"`
	TelemetryMode      string `mapstructure:"telemetry_mode"`
}
