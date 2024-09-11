package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/types"
	dmsUtils "gitlab.com/nunet/device-management-service/utils"
)

// NewOnboardCmd is a constructor for `onboard` command
func newOnboardCmd(client *dmsUtils.HTTPClient) *cobra.Command {
	fnCPU := "cpu"
	fnMemory := "memory"
	fnChannel := "nunet-channel"
	fnAddr := "address"
	fnPlugin := "plugin"
	fnNTXPrice := "ntx-price"
	fnLocal := "local-enable"
	fnCardano := "cardano"
	fnUnavailable := "unavailable"

	cmd := &cobra.Command{
		Use:   "onboard",
		Short: "Onboard current machine to NuNet",
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "Checking onboard status...")

			onboarded, err := checkOnboarded(client)
			if err != nil {
				return fmt.Errorf("could not check onboard status: %w", err)
			}

			if onboarded {
				err := utils.PromptReonboard(cmd.InOrStdin(), cmd.OutOrStdout())
				if err != nil {
					return err
				}
			}

			memory, err := cmd.Flags().GetUint64(fnMemory)
			if err != nil {
				fmt.Println("couldn't get 'memory' flag: %w", err)
			}
			cpu, err := cmd.Flags().GetInt64(fnCPU)
			if err != nil {
				fmt.Println("couldn't get 'cpu' flag: %w", err)
			}
			channel, err := cmd.Flags().GetString(fnChannel)
			if err != nil {
				fmt.Println("couldn't get 'channel' flag: %w", err)
			}
			addr, err := cmd.Flags().GetString(fnAddr)
			if err != nil {
				fmt.Println("couldn't get 'addr' flag: %w", err)
			}
			ntx, err := cmd.Flags().GetFloat64(fnNTXPrice)
			if err != nil {
				fmt.Println("couldn't get 'ntx' flag: %w", err)
			}
			local, err := cmd.Flags().GetBool(fnLocal)
			if err != nil {
				fmt.Println("couldn't get 'local' flag: %w", err)
			}
			unavailable, err := cmd.Flags().GetBool(fnUnavailable)
			if err != nil {
				fmt.Println("couldn't get 'unavailable' flag: %w", err)
			}
			cardano, err := cmd.Flags().GetBool(fnCardano)
			if err != nil {
				fmt.Println("couldn't get 'cardano' flag: %w", err)
			}

			reserved := types.CapacityForNunet{
				Memory:            memory,
				CPU:               cpu,
				Channel:           channel,
				PaymentAddress:    addr,
				NTXPricePerMinute: ntx,
				Cardano:           cardano,
				ServerMode:        local,
				IsAvailable:       unavailable, // TODO: Update this
			}

			onboardJSON, err := json.Marshal(reserved)
			if err != nil {
				return fmt.Errorf("unable to marshal JSON data: %w", err)
			}

			resBody, resCode, err := client.MakeRequest("POST", "/onboarding/onboard", onboardJSON)
			if err != nil {
				return fmt.Errorf("could not make request: %w", err)
			}

			if resCode != 201 {
				return fmt.Errorf("request failed with status code: %d", resCode)
			}

			// TODO: what to do with the response body?
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n", resBody)

			fmt.Fprintln(cmd.OutOrStdout(), "Successfully onboarded!")
			return nil
		},
	}

	cmd.Flags().Uint64P(fnMemory, "m", 0, "set value for memory usage")
	cmd.Flags().Int64P(fnCPU, "c", 0, "set value for CPU usage")
	cmd.Flags().StringP(fnChannel, "n", "", "set channel")
	cmd.Flags().StringP(fnAddr, "a", "", "set wallet address")
	cmd.Flags().Float64P(fnNTXPrice, "x", 0, "price in NTX per minute for onboarded compute resource")
	cmd.Flags().StringP(fnPlugin, "p", "", "set plugin")
	cmd.Flags().BoolP(fnUnavailable, "u", false, "unavailable for job deployment (default: false)")
	cmd.Flags().BoolP(fnLocal, "l", true, "set server mode (enable for local)")
	cmd.Flags().BoolP(fnCardano, "C", false, "set Cardano wallet")
	cmd.MarkFlagsRequiredTogether("memory", "cpu", "nunet-channel", "address")
	return cmd
}
