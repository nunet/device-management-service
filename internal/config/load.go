// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

var (
	cfg        Config
	homeDir, _ = os.UserHomeDir()
)

func getViper() *viper.Viper {
	v := viper.New()
	v.SetConfigName("dms_config")
	v.SetConfigType("json")
	v.AddConfigPath(".")                               // config file reading order starts with current working directory
	v.AddConfigPath(fmt.Sprintf("%s/.nunet", homeDir)) // then home directory
	v.AddConfigPath("/etc/nunet/")                     // finally /etc/nunet
	return v
}

func setDefaultConfig() *viper.Viper {
	v := getViper()
	v.SetDefault("general.user_dir", fmt.Sprintf("%s/.nunet", homeDir))
	v.SetDefault("general.work_dir", fmt.Sprintf("%s/nunet", homeDir))
	v.SetDefault("general.data_dir", fmt.Sprintf("%s/nunet/data", homeDir))
	v.SetDefault("general.debug", false)
	v.SetDefault("general.port_available_range_from", 16384)
	v.SetDefault("general.port_available_range_to", 32768)

	v.SetDefault("rest.addr", "127.0.0.1")
	v.SetDefault("rest.port", 9999)
	v.SetDefault("profiler.enabled", true)
	v.SetDefault("profiler.addr", "127.0.0.1")
	v.SetDefault("profiler.port", 6060)
	v.SetDefault("p2p.listen_address", []string{
		"/ip4/0.0.0.0/tcp/9000",
		"/ip4/0.0.0.0/udp/9000/quic-v1",
	})
	v.SetDefault("p2p.bootstrap_peers", []string{
		"/dnsaddr/bootstrap.p2p.nunet.io/p2p/QmQ2irHa8aFTLRhkbkQCRrounE4MbttNp8ki7Nmys4F9NP",
		"/dnsaddr/bootstrap.p2p.nunet.io/p2p/Qmf16N2ecJVWufa29XKLNyiBxKWqVPNZXjbL3JisPcGqTw",
		"/dnsaddr/bootstrap.p2p.nunet.io/p2p/QmTkWP72uECwCsiiYDpCFeTrVeUM9huGTPsg3m6bHxYQFZ",
	})
	v.SetDefault("p2p.memory", 1024)
	v.SetDefault("p2p.fd", 512)
	v.SetDefault("job.log_update_interval", 2)
	v.SetDefault("job.target_peer", "")
	v.SetDefault("job.cleanup_interval", 3)

	// default observability settings
	v.SetDefault("observability.log_level", "INFO")
	v.SetDefault("observability.log_file", fmt.Sprintf("%s/nunet/logs/nunet-dms.log", homeDir))
	v.SetDefault("observability.max_size", 100) // megabytes
	v.SetDefault("observability.max_backups", 3)
	v.SetDefault("observability.max_age", 28) // days
	v.SetDefault("observability.elasticsearch_url", "http://localhost:9200")
	v.SetDefault("observability.elasticsearch_index", "nunet-dms")
	v.SetDefault("observability.flush_interval", 5) // Default flush interval is 5 seconds
	v.SetDefault("observability.elasticsearch_enabled", true)
	v.SetDefault("observability.elasticsearch_api_key", "")

	// default APM settings
	v.SetDefault("apm.server_url", "http://apm.telemetry.nunet.io")
	v.SetDefault("apm.service_name", "nunet-dms")
	v.SetDefault("apm.environment", "production")
	v.SetDefault("apm.api_key", v.GetString("observability.elasticsearch_api_key"))

	return v
}

func LoadConfig() error {
	v := setDefaultConfig()
	if err := v.ReadInConfig(); err != nil {
		if err := setDefaultConfig().UnmarshalExact(&cfg); err != nil {
			return fmt.Errorf("failed to unmarshal default config: %w", err)
		}
		return nil
	}
	if err := v.UnmarshalExact(&cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return nil
}

func GetConfig() *Config {
	if reflect.DeepEqual(cfg, Config{}) {
		if err := LoadConfig(); err != nil {
			return &cfg
		}
	}
	return &cfg
}

func Get(key string) (interface{}, error) {
	v := getViper()
	loadedConfig, err := json.Marshal(GetConfig())
	if err != nil {
		return nil, fmt.Errorf("could not marshal config: %w", err)
	}
	if err := v.ReadConfig(bytes.NewReader(loadedConfig)); err != nil {
		return nil, fmt.Errorf("could not read config: %w", err)
	}
	if !v.IsSet(key) {
		return nil, fmt.Errorf("key '%s' not found in configuration", key)
	}
	return v.Get(key), nil
}

func Set(fs afero.Fs, key string, value interface{}) error {
	v := getViper()
	v.SetFs(fs)

	v.Set(key, value)
	if err := v.UnmarshalExact(&cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	loadedConfig, err := json.Marshal(GetConfig())
	if err != nil {
		return fmt.Errorf("could not marshal config: %w", err)
	}
	if err := v.MergeConfig(bytes.NewReader(loadedConfig)); err != nil {
		return fmt.Errorf("failed to merge config: %w", err)
	}

	if err := v.WriteConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file does not exist, create it.
			return v.SafeWriteConfig()
		}
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func FileExists(fs afero.Fs) (bool, error) {
	v := getViper()
	v.SetFs(fs)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return false, nil
		}
		return false, fmt.Errorf("could not read config file: %w", err)
	}
	return true, nil
}

func GetPath() string {
	v := getViper()
	if err := v.ReadInConfig(); err != nil {
		return setDefaultConfig().ConfigFileUsed()
	}
	return v.ConfigFileUsed()
}

func CreateConfigFileIfNotExists(fs afero.Fs) error {
	exists, err := FileExists(fs)
	if err != nil {
		return fmt.Errorf("failed to check if config file exists: %w", err)
	}
	if !exists {
		v := setDefaultConfig()
		if err := v.SafeWriteConfig(); err != nil {
			return fmt.Errorf("failed to create config file: %w", err)
		}
	}
	return nil
}
