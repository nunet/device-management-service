// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package config

type Config struct {
	Profiler      `mapstructure:"profiler" json:"profiler"`
	General       `mapstructure:"general" json:"general"`
	Rest          `mapstructure:"rest" json:"rest"`
	P2P           `mapstructure:"p2p" json:"p2p"`
	Job           `mapstructure:"job" json:"job"`
	Observability `mapstructure:"observability" json:"observability"`
	APM           `mapstructure:"apm" json:"apm"`
}

type General struct {
	UserDir                string `mapstructure:"user_dir" json:"user_dir"`
	WorkDir                string `mapstructure:"work_dir" json:"work_dir"`
	DataDir                string `mapstructure:"data_dir" json:"data_dir"`
	Debug                  bool   `mapstructure:"debug" json:"debug"`
	HostCountry            string `mapstructure:"host_country" json:"host_country"`
	HostCity               string `mapstructure:"host_city" json:"host_city"`
	HostContinent          string `mapstructure:"host_continent" json:"host_continent"`
	PortAvailableRangeFrom int    `mapstructure:"port_available_range_from" json:"port_available_range_from"`
	PortAvailableRangeTo   int    `mapstructure:"port_available_range_to" json:"port_available_range_to"`
}

type Rest struct {
	Addr string `mapstructure:"addr" json:"addr"`
	Port uint32 `mapstructure:"port" json:"port"`
}

type Profiler struct {
	Enabled bool   `mapstructure:"enabled" json:"enabled"`
	Addr    string `mapstructure:"addr" json:"addr"`
	Port    uint32 `mapstructure:"port" json:"port"`
}

type P2P struct {
	ListenAddress   []string `mapstructure:"listen_address" json:"listen_address"`
	BootstrapPeers  []string `mapstructure:"bootstrap_peers" json:"bootstrap_peers"`
	Memory          int      `mapstructure:"memory" json:"memory"`
	FileDescriptors int      `mapstructure:"fd" json:"fd"`
}

type Job struct {
	LogUpdateInterval int    `mapstructure:"log_update_interval" json:"log_update_interval"` // in minutes
	TargetPeer        string `mapstructure:"target_peer" json:"target_peer"`                 // specific peer to send deployment requests to - XXX probably not a good idea. Remove after testing stage.
	CleanupInterval   int    `mapstructure:"cleanup_interval" json:"cleanup_interval"`       // docker container and images clean up interval in days
}

type Observability struct {
	LogLevel           string `mapstructure:"log_level" json:"log_level"`
	LogFile            string `mapstructure:"log_file" json:"log_file"`
	MaxSize            int    `mapstructure:"max_size" json:"max_size"` // in megabytes
	MaxBackups         int    `mapstructure:"max_backups" json:"max_backups"`
	MaxAge             int    `mapstructure:"max_age" json:"max_age"` // in days
	ElasticsearchURL   string `mapstructure:"elasticsearch_url" json:"elasticsearch_url"`
	ElasticsearchIndex string `mapstructure:"elasticsearch_index" json:"elasticsearch_index"`
	FlushInterval      int    `mapstructure:"flush_interval" json:"flush_interval"` // in seconds
}

type APM struct {
	ServerURL   string `mapstructure:"server_url" json:"server_url"`
	ServiceName string `mapstructure:"service_name" json:"service_name"`
	Environment string `mapstructure:"environment" json:"environment"`
	Certificate string `mapstructure:"certificate" json:"certificate"`
	Key         string `mapstructure:"key" json:"key"`
	CA          string `mapstructure:"ca" json:"ca"`
}
