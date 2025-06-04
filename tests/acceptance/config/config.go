package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type IncusHost struct {
	Host       string `yaml:"host"`
	ClientCert string `yaml:"client_cert"`
	ClientKey  string `yaml:"client_key"`
	ServerCert string `yaml:"server_cert"`
	Port       int    `yaml:"port"`
}

type Config struct {
	IncusHosts []IncusHost `yaml:"incus_hosts"`
}

// Parse parses a yaml file and returns host config
func Parse(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to read file: %w", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal yaml: %w", err)
	}

	return &config, nil
}

// Get get configuration from environment variables or default to local
func Get() (*Config, error) {
	confFile := os.Getenv("DMS_ACC_TEST_CONFIG_FILE")

	var config *Config
	if confFile == "" {
		config = &Config{
			IncusHosts: []IncusHost{
				{
					Host: "local",
				},
			},
		}
	} else {
		c, err := Parse(confFile)
		if err != nil {
			return nil, fmt.Errorf("config file parse failed: %v", err)
		}
		config = c
	}

	return config, nil
}
