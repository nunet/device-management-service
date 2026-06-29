// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package config

// Config aggregates all sub-sections that can be loaded from the config file.
type Config struct {
	Profiler      `mapstructure:"profiler"      json:"profiler"`
	General       `mapstructure:"general"       json:"general"`
	Rest          `mapstructure:"rest"          json:"rest"`
	P2P           `mapstructure:"p2p"           json:"p2p"`
	Job           `mapstructure:"job"           json:"job"`
	Observability `mapstructure:"observability" json:"observability"`
	APM           `mapstructure:"apm"           json:"apm"`
	CoinMarketCap CoinMarketCapConfig `mapstructure:"coinmarketcap" json:"coinmarketcap"`
}

type General struct {
	Env                      string `mapstructure:"env" json:"env"`
	UserDir                  string `mapstructure:"user_dir"                   json:"user_dir"`
	WorkDir                  string `mapstructure:"work_dir"                   json:"work_dir"`
	DataDir                  string `mapstructure:"data_dir"                   json:"data_dir"`
	Debug                    bool   `mapstructure:"debug"                      json:"debug"`
	HostCity                 string `mapstructure:"host_city"                  json:"host_city"`
	HostCountry              string `mapstructure:"host_country"               json:"host_country"`
	HostContinent            string `mapstructure:"host_continent"             json:"host_continent"`
	PortAvailableRangeFrom   int    `mapstructure:"port_available_range_from"  json:"port_available_range_from"`
	PortAvailableRangeTo     int    `mapstructure:"port_available_range_to"    json:"port_available_range_to"`
	StorageMode              bool   `mapstructure:"storage_mode"               json:"storage_mode"`
	StorageCADirectory       string `mapstructure:"storage_ca_directory"       json:"storage_ca_directory"`
	StorageBricksDir         string `mapstructure:"storage_bricks_dir"         json:"storage_bricks_dir"`
	StorageGlusterfsHostname string `mapstructure:"storage_glusterfs_hostname" json:"storage_glusterfs_hostname"`
	ComputeGateway           bool   `mapstructure:"compute_gateway" json:"compute_gateway"`

	// server provider configurations
	Providers []ProviderConfig `mapstructure:"providers" json:"providers"`

	PaymentProvider PaymentProvider `mapstructure:"payment_provider" json:"payment_provider"`
}

type ProviderConfig struct {
	Name   string                 `mapstructure:"name" json:"name"`
	Type   string                 `mapstructure:"type" json:"type"` // e.g "local-incus"
	Config map[string]interface{} `mapstructure:"config" json:"config"`
}

type PaymentProvider struct {
	Mode               bool   `mapstructure:"mode" json:"mode"`
	EthereumRPCURL     string `mapstructure:"ethereum_rpc_url" json:"ethereum_rpc_url"`
	EthereumRPCToken   string `mapstructure:"ethereum_rpc_token" json:"ethereum_rpc_token"`
	NtxContractAddress string `mapstructure:"ntx_contract_address" json:"ntx_contract_address"`

	BlockFrostAPIURL     string `mapstructure:"block_frost_api_url" json:"block_frost_api_url"`
	BlockFrostAPIKey     string `mapstructure:"block_frost_api_key" json:"block_frost_api_key"`
	CardanoAssetName     string `mapstructure:"cardano_asset_name" json:"cardano_asset_name"`
	CardanoAssetPolicyID string `mapstructure:"cardano_asset_policy_id" json:"cardano_asset_policy_id"`
}

type Rest struct {
	Addr string `mapstructure:"addr" json:"addr"`
	Port uint32 `mapstructure:"port" json:"port"`
}

type Profiler struct {
	Enabled bool   `mapstructure:"enabled" json:"enabled"`
	Addr    string `mapstructure:"addr"    json:"addr"`
	Port    uint32 `mapstructure:"port"    json:"port"`
}

type P2P struct {
	ListenAddress   []string `mapstructure:"listen_address" json:"listen_address"`
	BootstrapPeers  []string `mapstructure:"bootstrap_peers" json:"bootstrap_peers"`
	Memory          int      `mapstructure:"memory" json:"memory"`
	FileDescriptors int      `mapstructure:"fd" json:"fd"`
}

type Job struct {
	AllowPrivilegedDocker         bool       `mapstructure:"allow_privileged_docker" json:"allow_privileged_docker"`
	RequireContractsForDeployment bool       `mapstructure:"require_contracts_for_deployment" json:"require_contracts_for_deployment"`
	Containerd                    Containerd `mapstructure:"containerd" json:"containerd"`
}

type Containerd struct {
	SocketPath     string `mapstructure:"socket_path" json:"socket_path"`
	Namespace      string `mapstructure:"namespace" json:"namespace"`
	ConfigPath     string `mapstructure:"config_path" json:"config_path"`
	CNINetConfDir  string `mapstructure:"cni_net_conf_dir" json:"cni_net_conf_dir"`
	CNIPluginDir   string `mapstructure:"cni_plugin_dir" json:"cni_plugin_dir"`
	CNINetworkName string `mapstructure:"cni_network_name" json:"cni_network_name"`
	CNIBridgeIface string `mapstructure:"cni_bridge_iface" json:"cni_bridge_iface"`
	NetNSBaseDir   string `mapstructure:"netns_base_dir" json:"netns_base_dir"`
}

// Observability

type Observability struct {
	// Preferred structured layout TODO bind in observability
	Logging Logging `mapstructure:"logging" json:"logging"`
	Elastic Elastic `mapstructure:"elastic" json:"elastic"`
	OTel    OTel    `mapstructure:"otel"    json:"otel"`

	// Deprecated: use Logging.Level instead
	LogLevel string `mapstructure:"log_level"             json:"-"`
	// Deprecated: use Logging.File instead
	LogFile string `mapstructure:"log_file"              json:"-"`
	// Deprecated: use Logging.Rotation instead
	MaxSize int `mapstructure:"max_size"              json:"-"` // MB
	// Deprecated: use Logging.Rotation.MaxBackups instead
	MaxBackups int `mapstructure:"max_backups"           json:"-"`
	// Deprecated: use Logging.Rotation.MaxAgeDays instead
	MaxAge int `mapstructure:"max_age"               json:"-"` // days

	// Deprecated: use Elastic.URL instead
	ElasticsearchURL string `mapstructure:"elasticsearch_url"     json:"-"`
	// Deprecated: use Elastic.Index instead
	ElasticsearchIndex string `mapstructure:"elasticsearch_index"   json:"-"`
	// Deprecated: use Elastic.FlushInterval instead
	FlushInterval int `mapstructure:"flush_interval"        json:"-"` // seconds
	// Deprecated: use Elastic.Enabled instead
	ElasticsearchEnabled bool `mapstructure:"elasticsearch_enabled" json:"-"`
	// Deprecated: use Elastic.APIKey instead
	ElasticsearchAPIKey string `mapstructure:"elasticsearch_api_key" json:"-"`
	// Deprecated: use Elastic.InsecureSkipVerify instead
	InsecureSkipVerify bool `mapstructure:"insecure_skip_verify"  json:"-"`
}

type Logging struct {
	Level    string   `mapstructure:"level"    json:"level"`
	File     string   `mapstructure:"file"     json:"file"`
	Rotation Rotation `mapstructure:"rotation" json:"rotation"`
}

type Rotation struct {
	MaxSizeMB  int `mapstructure:"max_size_mb"  json:"max_size_mb"`
	MaxBackups int `mapstructure:"max_backups"  json:"max_backups"`
	MaxAgeDays int `mapstructure:"max_age_days" json:"max_age_days"`
}

type Elastic struct {
	URL                string `mapstructure:"url"                 json:"url"`
	Index              string `mapstructure:"index"               json:"index"`
	FlushInterval      int    `mapstructure:"flush_interval"      json:"flush_interval"` // seconds
	Enabled            bool   `mapstructure:"enabled"             json:"enabled"`
	APIKey             string `mapstructure:"api_key"             json:"api_key"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify" json:"insecure_skip_verify"`
}

// APM

type APM struct {
	ServerURL   string `mapstructure:"server_url"   json:"server_url"`
	ServiceName string `mapstructure:"service_name" json:"service_name"`
	Environment string `mapstructure:"environment"  json:"environment"`
	APIKey      string `mapstructure:"api_key"      json:"api_key"`
	// SecretToken is a legacy API key used for local ELK deployments.
	SecretToken string `mapstructure:"secret_token"      json:"secret_token"`
}

type OTel struct {
	// Enabled controls whether OTel metrics are exported
	Enabled bool `mapstructure:"enabled" json:"enabled"`
	// Endpoint is the OTel Collector gRPC endpoint (e.g., "localhost:4317")
	Endpoint string `mapstructure:"endpoint" json:"endpoint"`
	// Insecure disables TLS for the gRPC connection
	Insecure bool `mapstructure:"insecure" json:"insecure"`
}

// CoinMarketCap configuration for price oracle
type CoinMarketCapConfig struct {
	APIKey       string `mapstructure:"api_key" json:"api_key"`
	BaseURL      string `mapstructure:"base_url" json:"base_url"`
	EndpointPath string `mapstructure:"endpoint_path" json:"endpoint_path"`
	CacheTTL     string `mapstructure:"cache_ttl" json:"cache_ttl"`
	QuoteTTL     string `mapstructure:"quote_ttl" json:"quote_ttl"` // Time-to-live for payment quotes
}
