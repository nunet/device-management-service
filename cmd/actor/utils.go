package actor

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/dms/jobs/parser"
	"gitlab.com/nunet/device-management-service/dms/node"
)

func displayResponse(cmd *cobra.Command, resp any) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(resp)
}

func ProcessEnsembleYaml(fs afero.Afero, path string) (
	*node.NewDeploymentRequest, error,
) {
	data, err := fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := &node.NewDeploymentRequest{}
	err = parser.Parse(parser.SpecTypeEnsembleV1, data, &cfg.Ensemble)
	if err != nil {
		return nil, err
	}

	for name, script := range cfg.Ensemble.V1.Scripts {
		scriptData, err := fs.ReadFile(string(script))
		if err != nil {
			return nil, fmt.Errorf("failed to read script file: %w", err)
		}
		cfg.Ensemble.V1.Scripts[name] = scriptData
	}

	for aIdx, alloc := range cfg.Ensemble.V1.Allocations {
		// handle reading public key files
		if alloc.Keys != nil {
			for kIdx, key := range alloc.Keys {
				keyData, err := fs.ReadFile(key.File)
				if err != nil {
					return nil, fmt.Errorf("failed to read key file: %w", err)
				}
				cfg.Ensemble.V1.Allocations[aIdx].Keys[kIdx].File = string(keyData)
			}
		}

		// handle reading client certificate files for volumes
		if alloc.Volume != nil {
			if alloc.Volume.ClientPrivateKey != "" {
				pvkeyData, err := fs.ReadFile(alloc.Volume.ClientPrivateKey)
				if err != nil {
					return nil, fmt.Errorf("failed to read pvkeyData data: %w", err)
				}
				cfg.Ensemble.V1.Allocations[aIdx].Volume.ClientPrivateKey = string(pvkeyData)
			}

			if alloc.Volume.ClientPEM != "" {
				pemData, err := fs.ReadFile(alloc.Volume.ClientPEM)
				if err != nil {
					return nil, fmt.Errorf("failed to read pem data: %w", err)
				}
				cfg.Ensemble.V1.Allocations[aIdx].Volume.ClientPEM = string(pemData)
			}

			if alloc.Volume.ClientCA != "" {
				caData, err := fs.ReadFile(alloc.Volume.ClientCA)
				if err != nil {
					return nil, fmt.Errorf("failed to read ca data: %w", err)
				}
				cfg.Ensemble.V1.Allocations[aIdx].Volume.ClientCA = string(caData)
			}
		}
	}

	return cfg, nil
}
