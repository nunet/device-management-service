package cmd

import (
	"errors"
	"fmt"

	"github.com/buger/jsonparser"
	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/cmd/backend"
)

var resourceConfigCmd = NewResourceConfigCmd(networkService, utilsService)

func NewResourceConfigCmd(net backend.NetworkManager, utilsService backend.Utility) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "resource-config",
		Short:   "Update configuration of onboarded device",
		PreRunE: isDMSRunning(net),
		RunE: func(cmd *cobra.Command, _ []string) error {
			memory, _ := cmd.Flags().GetInt64("memory")
			cpu, _ := cmd.Flags().GetInt64("cpu")
			ntxPrice, _ := cmd.Flags().GetFloat64("ntx-price")

			// check for both flags values
			if memory == 0 || cpu == 0 || ntxPrice < 0 {
				_ = cmd.Help()
				return fmt.Errorf("all flag values must be specified")
			}

			err := checkOnboarded(utilsService)
			if err != nil {
				return err
			}

			// set data for body request
			resourceBody, err := setOnboardData(memory, cpu, ntxPrice, "", "", false, false, true)
			if err != nil {
				return fmt.Errorf("failed to set onboard data: %w", err)
			}

			resp, err := utilsService.ResponseBody(nil, "POST", "/api/v1/onboarding/resource-config", "", resourceBody)
			if err != nil {
				return fmt.Errorf("could not get response body: %w", err)
			}

			msg, err := jsonparser.GetString(resp, "error")
			if err == nil {
				return errors.New(msg)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Resources updated successfully!")
			fmt.Fprintln(cmd.OutOrStdout(), string(resp))
			return nil
		},
	}

	cmd.Flags().Int64VarP(&flagMemory, "memory", "m", 0, "set amount of memory")
	cmd.Flags().Int64VarP(&flagCPU, "cpu", "c", 0, "set amount of CPU")
	cmd.Flags().Float64VarP(&flagNtxPrice, "ntx-price", "x", 0, "Set NTX Price per minute for compute resources you are updating")
	return cmd
}
