package config

import (
	"fmt"
	"os"

	"gitlab.com/nunet/device-management-service/utils"
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
	IncusHosts      []IncusHost `yaml:"incus_hosts"`
	VMsPrefix       string      `yaml:"vms_prefix"`
	GlusterfsVMName string      `yaml:"glusterfs_vm_name"`
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

func saveConfig(filename string, config *Config) error {
	if err := os.Rename(filename, filename+".bk"); err != nil {
		return fmt.Errorf("unable to move file %s: %w", filename, err)
	}

	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("unable to create file %s for writing: %w", filename, err)
	}
	defer f.Close()

	d, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("unable to marshal yaml: %w", err)
	}

	if _, err := f.Write(d); err != nil {
		return fmt.Errorf("unable to write to file %s: %w", filename, err)
	}

	return nil
}

// Get get configuration from environment variables or default to local
func Get() (*Config, error) {
	confFile := os.Getenv("DMS_ACC_TEST_CONFIG_FILE")

	var config *Config
	if confFile == "" {
		config = &Config{
			VMsPrefix:       "acc-test",
			GlusterfsVMName: "glusterfs-test-node1",
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

	if config.VMsPrefix == "" && confFile != "" {
		fmt.Print("[WARN] VM Prefix empty. Generating one and adding to config file...\n")
		randString, err := utils.RandomString(8)
		if err != nil {
			return nil, err
		}
		config.VMsPrefix = "acc-test-" + randString

		if err := saveConfig(confFile, config); err != nil {
			return nil, err
		}
		fmt.Printf("[INFO] Added vm prefix %s to config file %s\n", config.VMsPrefix, confFile)
	}

	return config, nil
}
