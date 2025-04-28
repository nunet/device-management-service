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
		fmt.Println(name, string(script))
		scriptData, err := fs.ReadFile(string(script))
		if err != nil {
			return nil, fmt.Errorf("failed to read script file: %w", err)
		}
		cfg.Ensemble.V1.Scripts[name] = scriptData
	}

	for i, alloc := range cfg.Ensemble.V1.Allocations {
		if alloc.Volume == nil {
			continue
		}

		if alloc.Volume.ClientPrivateKey != "" {
			pvkeyData, err := fs.ReadFile(alloc.Volume.ClientPrivateKey)
			if err != nil {
				return nil, fmt.Errorf("failed to read pvkeyData data: %w", err)
			}
			cfg.Ensemble.V1.Allocations[i].Volume.ClientPrivateKey = string(pvkeyData)
		}

		if alloc.Volume.ClientPEM != "" {
			pemData, err := fs.ReadFile(alloc.Volume.ClientPEM)
			if err != nil {
				return nil, fmt.Errorf("failed to read pem data: %w", err)
			}
			cfg.Ensemble.V1.Allocations[i].Volume.ClientPEM = string(pemData)
		}

		if alloc.Volume.ClientCA != "" {
			caData, err := fs.ReadFile(alloc.Volume.ClientCA)
			if err != nil {
				return nil, fmt.Errorf("failed to read ca data: %w", err)
			}
			cfg.Ensemble.V1.Allocations[i].Volume.ClientCA = string(caData)
		}
	}

	return cfg, nil
}
