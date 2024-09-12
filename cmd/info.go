package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/buger/jsonparser"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"
)

// NewInfoCmd is a constructor for `info` command
func newInfoCmd(client *utils.HTTPClient) *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Display information about onboarded device",
		Long:  "Display onboarding config of onboarded device on Nunet Device Management Service",
		RunE: func(cmd *cobra.Command, _ []string) error {
			onboarded, err := checkOnboarded(client)
			if err != nil {
				return fmt.Errorf("could not check onboard status: %w", err)
			}
			if !onboarded {
				return fmt.Errorf("machine is not onboarded")
			}

			resBody, resCode, err := client.MakeRequest("GET", "/onboarding/info", nil)
			if err != nil {
				return fmt.Errorf("could not make request: %w", err)
			}

			if resCode != 200 {
				return fmt.Errorf("request failed with status code: %d", resCode)
			}

			info, _, _, err := jsonparser.Get(resBody, "info")
			if err != nil {
				return fmt.Errorf("unable to parse JSON from key: %w", err)
			}

			var config types.OnboardingConfig
			if err := json.Unmarshal(info, &config); err != nil {
				return fmt.Errorf("could not unmarshal response body: %w", err)
			}

			displayInTable(cmd.OutOrStdout(), &config)
			return nil
		},
	}
}

func displayInTable(w io.Writer, config *types.OnboardingConfig) {
	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"Info", "Value"})

	table.Append([]string{"Name", config.Name})
	table.Append([]string{"Update Timestamp", fmt.Sprintf("%d", config.UpdateTimestamp)})
	table.Append([]string{"Memory Max", fmt.Sprintf("%d", config.TotalResources.RAM.Size)})
	table.Append([]string{"Total Core", fmt.Sprintf("%.2f", config.TotalResources.CPU.Cores)})
	table.Append([]string{"CPU Max", fmt.Sprintf("%.2f", config.TotalResources.CPU.Compute)})
	table.Append([]string{"Reserved CPU", fmt.Sprintf("%.2f", config.OnboardedResources.CPU.Compute)})
	table.Append([]string{"Reserved Memory", fmt.Sprintf("%d", config.OnboardedResources.RAM.Size)})
	table.Append([]string{"Network", config.Network})
	table.Append([]string{"Public Key", config.PublicKey})
	table.Append([]string{"Node ID", config.NodeID})
	table.Append([]string{"Allow Cardano", fmt.Sprintf("%t", config.AllowCardano)})
	table.Append([]string{"Dashboard", config.Dashboard})
	table.Append([]string{"NTX Price Per Minute", fmt.Sprintf("%f", config.NTXPricePerMinute)})

	table.Render()
}
