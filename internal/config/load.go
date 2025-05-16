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

	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	cfg            Config
	homeDir, _     = os.UserHomeDir()
	defaultcfgName = "dms_config"
	defaultCfgExt  = "json"
	defaultCfgPath = "."
)

var log = logging.Logger("config")

func SetupConfig() {
	v := viper.GetViper()
	fs := pflag.NewFlagSet("", pflag.ContinueOnError)
	var cfgFile string

	fs.StringVar(&cfgFile, "config", "", "config file")
	setupFlags(fs)

	pflag.CommandLine.AddFlagSet(fs)
	fs.Usage = func() {}

	err := fs.Parse(os.Args)
	if err != nil && err != pflag.ErrHelp {
		// we can ignore flags intended for other commands
		log.Warnf("error parsing command line flags: %s", err)
	}

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	}
}

func init() {
	v := viper.GetViper()
	v.SetConfigName(defaultcfgName)
	v.SetConfigType(defaultCfgExt)
	v.AddConfigPath(defaultCfgPath)                    // config file reading order starts with current working directory
	v.AddConfigPath(fmt.Sprintf("%s/.nunet", homeDir)) // then home directory
	v.AddConfigPath("/etc/nunet/")                     // finally system directory

	setDefaultConfig()

	v.AutomaticEnv()
}

// setupFlags sets up the command line flags and binds them to viper
func setupFlags(flagSet *pflag.FlagSet) {
	v := viper.GetViper()

	flags := []struct {
		name      string
		viperKey  string
		shorthand string
		usage     string
		valueType string
		hidden    bool
	}{
		{"rest-addr", "rest.addr", "", "REST API host", "string", false},
		{"rest-port", "rest.port", "", "REST API port", "int", false},
		{"user-dir", "general.user_dir", "", "user directory", "string", false},
		{"work-dir", "general.work_dir", "", "work directory", "string", false},
		{"data-dir", "general.data_dir", "", "data directory", "string", false},
		{"debug", "general.debug", "", "debug mode", "bool", false},
		{"profiler-port", "profiler.port", "", "profiler port", "int", false},
	}

	for _, flag := range flags {
		switch flag.valueType {
		case "string":
			flagSet.StringP(flag.name, flag.shorthand, v.GetString(flag.viperKey), flag.usage)
		case "int":
			flagSet.IntP(flag.name, flag.shorthand, v.GetInt(flag.viperKey), flag.usage)
		case "bool":
			flagSet.BoolP(flag.name, flag.shorthand, v.GetBool(flag.viperKey), flag.usage)
		}
		if flag.hidden {
			if err := flagSet.MarkHidden(flag.name); err != nil {
				log.Debugf("failed to mark flag %s as hidden: %s", flag.name, err)
			}
		}

		// bind flags to viper
		if err := v.BindPFlag(flag.viperKey, flagSet.Lookup(flag.name)); err != nil {
			log.Errorf("failed to bind flag %s to viper: %v", flag.name, err)
		}
	}
}

// setDefaultConfig sets default values for configuration
func setDefaultConfig() {
	v := viper.GetViper()
	v.SetDefault("general.user_dir", fmt.Sprintf("%s/.nunet", homeDir))
	v.SetDefault("general.work_dir", fmt.Sprintf("%s/nunet", homeDir))
	v.SetDefault("general.data_dir", fmt.Sprintf("%s/nunet/data", homeDir))
	v.SetDefault("general.debug", false)
	v.SetDefault("general.port_available_range_from", 16384)
	v.SetDefault("general.port_available_range_to", 65536)

	v.SetDefault("general.storage_ca_directory", fmt.Sprintf("%s/.nunet/storage_ca_directory", homeDir))
	v.SetDefault("general.storage_bricks_dir", fmt.Sprintf("%s/.nunet/storage_bricks_dir", homeDir))

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

	// default observability settings
	v.SetDefault("observability.log_level", "INFO")
	v.SetDefault("observability.log_file", fmt.Sprintf("%s/nunet/logs/nunet-dms.log", homeDir))
	v.SetDefault("observability.max_size", 100) // megabytes
	v.SetDefault("observability.max_backups", 3)
	v.SetDefault("observability.max_age", 28) // days
	v.SetDefault("observability.elasticsearch_url", "http://localhost:9200")
	v.SetDefault("observability.elasticsearch_index", "nunet-dms")
	v.SetDefault("observability.flush_interval", 5) // Default flush interval is 5 seconds
	v.SetDefault("observability.elasticsearch_enabled", false)
	v.SetDefault("observability.elasticsearch_api_key", "")
	v.SetDefault("observability.insecure_skip_verify", true) // Default to insecure TLS connections for now

	// default APM settings
	v.SetDefault("apm.server_url", "http://apm.telemetry.nunet.io")
	v.SetDefault("apm.service_name", "nunet-dms")
	v.SetDefault("apm.environment", "production")
	v.SetDefault("apm.api_key", v.GetString("observability.elasticsearch_api_key"))

	// jobs
	v.SetDefault("job.allow_privileged_docker", false)
}

func LoadConfig() error {
	v := viper.GetViper()
	if err := v.ReadInConfig(); err != nil {
		if err := v.UnmarshalExact(&cfg); err != nil {
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
	v := viper.GetViper()
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
	v := viper.GetViper()
	v.SetFs(fs)

	// get the current config for backup
	currentConfig, err := json.Marshal(GetConfig())
	if err != nil {
		return fmt.Errorf("could not read the current config: %w", err)
	}

	// update the config
	err = v.MergeConfigMap(map[string]any{key: value})
	if err != nil {
		return fmt.Errorf("failed to set %s: %w", key, err)
	}

	// update config struct
	if err := v.UnmarshalExact(&cfg); err != nil {
		_ = v.ReadConfig(bytes.NewReader(currentConfig))
		return fmt.Errorf("invalid config key: %s", key)
	}

	// Write updated config to file
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
	v := viper.GetViper()

	v.SetFs(fs)

	if cfgFile := v.ConfigFileUsed(); cfgFile != "" {
		return afero.Exists(fs, cfgFile)
	}
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return false, nil
		}
		return false, fmt.Errorf("could not read config file: %w", err)
	}
	return true, nil
}

func GetPath() string {
	v := viper.GetViper()
	if err := v.ReadInConfig(); err != nil {
		return v.ConfigFileUsed()
	}
	return v.ConfigFileUsed()
}

func CreateConfigFileIfNotExists(fs afero.Fs) error {
	exists, err := FileExists(fs)
	if err != nil {
		return fmt.Errorf("failed to check if config file exists: %w", err)
	}
	if !exists {
		v := viper.GetViper()
		v.SetFs(fs)
		if err := v.SafeWriteConfig(); err != nil {
			return fmt.Errorf("failed to create config file: %w", err)
		}
	}
	return nil
}
