package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"
)

// NewResourceConfigCmd is a constructor for `resource-config` command
func newResourceConfigCmd(client *utils.HTTPClient) *cobra.Command {
	fnMemory := "memory"
	fnCPU := "cpu"
	fnNTXPrice := "ntx-price"

	cmd := &cobra.Command{
		Use:   "resource-config",
		Short: "Update configuration of onboarded device",
		RunE: func(cmd *cobra.Command, _ []string) error {
			onboarded, err := checkOnboarded(client)
			if err != nil {
				return fmt.Errorf("could not check onboard status: %w", err)
			}
			if !onboarded {
				return fmt.Errorf("machine is not onboarded")
			}

			memory, _ := cmd.Flags().GetUint64(fnMemory)
			cpu, _ := cmd.Flags().GetInt64(fnCPU)
			ntx, _ := cmd.Flags().GetFloat64(fnNTXPrice)

			updated := types.CapacityForNunet{
				Memory:            memory,
				CPU:               cpu,
				NTXPricePerMinute: ntx,
			}

			updatedConfig, err := json.Marshal(updated)
			if err != nil {
				return fmt.Errorf("unable to marshal JSON data: %w", err)
			}

			resBody, resCode, err := client.MakeRequest("POST", "/onboarding/resource-config", updatedConfig)
			if err != nil {
				return fmt.Errorf("unable to make request: %w", err)
			}

			if resCode != 200 {
				return fmt.Errorf("request failed with status code: %d", resCode)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Resources updated successfully!")
			fmt.Fprintln(cmd.OutOrStdout(), string(resBody))
			return nil
		},
	}
	cmd.Flags().Uint64P(fnMemory, "m", 0, "set amount of memory")
	cmd.Flags().Int64P(fnCPU, "c", 0, "set amount of CPU")
	cmd.Flags().Float64P(fnNTXPrice, "x", 0, "Set NTX Price per minute for compute resources you are updating")
	cmd.MarkFlagsRequiredTogether("cpu", "memory")
	return cmd
}
