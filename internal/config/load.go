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
	v.SetDefault("general.data_dir", "/var/nunet")
	v.SetDefault("general.debug", false)
	v.SetDefault("rest.addr", "127.0.0.1")
	v.SetDefault("rest.port", 9999)
	v.SetDefault("p2p.listen_address", []string{
		"/ip4/0.0.0.0/tcp/9000",
		"/ip4/0.0.0.0/udp/9000/quic-v1",
	})
	v.SetDefault("p2p.bootstrap_peers", []string{
		"/dnsaddr/bootstrap.p2p.nunet.io/p2p/QmQ2irHa8aFTLRhkbkQCRrounE4MbttNp8ki7Nmys4F9NP",
		"/dnsaddr/bootstrap.p2p.nunet.io/p2p/Qmf16N2ecJVWufa29XKLNyiBxKWqVPNZXjbL3JisPcGqTw",
		"/dnsaddr/bootstrap.p2p.nunet.io/p2p/QmTkWP72uECwCsiiYDpCFeTrVeUM9huGTPsg3m6bHxYQFZ",
	})
	v.SetDefault("job.log_update_interval", 2)
	v.SetDefault("job.target_peer", "")
	v.SetDefault("job.cleanup_interval", 3)

	v.SetDefault("telemetry.service_name", "NunetDMS")
	v.SetDefault("telemetry.global_endpoint", "otel-collector.telemetry.nunet.io:4318")
	v.SetDefault("telemetry.observability_level", "INFO")
	v.SetDefault("telemetry.telemetry_mode", "production")

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
