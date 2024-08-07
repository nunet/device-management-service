package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"

	"github.com/spf13/viper"
)

var cfg Config
var home = os.Getenv("HOME")

func getViper() *viper.Viper {
	v := viper.New()
	v.SetConfigName("dms_config")
	v.SetConfigType("json")
	v.AddConfigPath(".")            // config file reading order starts with current working directory
	v.AddConfigPath("$HOME/.nunet") // then home directory
	v.AddConfigPath("/etc/nunet/")  // finally /etc/nunet
	return v
}

func setDefaultConfig() *viper.Viper {
	v := getViper()
	v.SetDefault("general.work_dir", "/etc/nunet")
	v.SetDefault("general.data_dir", "/var/nunet")
	v.SetDefault("general.debug", false)
	v.SetDefault("rest.port", 9999)
	v.SetDefault("p2p.listen_address", []string{
		"/ip4/0.0.0.0/tcp/9000",
		"/ip4/0.0.0.0/udp/9000/quic",
	})
	v.SetDefault("p2p.bootstrap_peers", []string{
		"/dnsaddr/bootstrap.p2p.nunet.io/p2p/QmQ2irHa8aFTLRhkbkQCRrounE4MbttNp8ki7Nmys4F9NP",
		"/dnsaddr/bootstrap.p2p.nunet.io/p2p/Qmf16N2ecJVWufa29XKLNyiBxKWqVPNZXjbL3JisPcGqTw",
		"/dnsaddr/bootstrap.p2p.nunet.io/p2p/QmTkWP72uECwCsiiYDpCFeTrVeUM9huGTPsg3m6bHxYQFZ",
	})
	v.SetDefault("job.log_update_interval", 2)
	v.SetDefault("job.target_peer", "")
	v.SetDefault("job.cleanup_interval", 3)

	return v
}

func LoadConfig() {
	paths := []string{
		".",
		home + "/.nunet",
		"/etc/nunet",
	}
	configFile := "dms_config.json"
	v := setDefaultConfig()

	config, err := findConfig(paths, configFile)
	if err != nil {
		setDefaultConfig().Unmarshal(&cfg)
	}

	modifiedConfig := removeComments(config)
	if err = v.ReadConfig(bytes.NewBuffer(modifiedConfig)); err != nil { // Viper only reads buffer, keeping comments in original config
		setDefaultConfig().Unmarshal(&cfg)
	}

	if err = v.Unmarshal(&cfg); err != nil {
		setDefaultConfig().Unmarshal(&cfg)
	}
}

func SetConfig(key string, value interface{}) {
	v := getViper()
	v.Set(key, value)
	err := v.Unmarshal(&cfg)
	if err != nil {
		setDefaultConfig().Unmarshal(&cfg)
	}
}

func GetConfig() *Config {
	if reflect.DeepEqual(cfg, Config{}) {
		LoadConfig()
	}
	return &cfg
}

func findConfig(paths []string, filename string) ([]byte, error) {
	for _, path := range paths {
		fullPath := filepath.Join(path, filename)
		_, err := os.Stat(fullPath)
		if err == nil {
			config, err := os.ReadFile(fullPath)
			if err == nil {
				return config, nil
			} else {
				return nil, err
			}
		}
	}

	return nil, fmt.Errorf("file not found in any of the paths")
}

func removeComments(configBytes []byte) []byte {
	re := regexp.MustCompile("(?s)//.*?\n") // match all '//' until the end of the line
	result := re.ReplaceAll(configBytes, nil)
	return result
}
