package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/buger/jsonparser"
	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"
)

// NewCapacityCmd is a constructor for `capacity` command
func newCapacityCmd(client *utils.HTTPClient) *cobra.Command {
	fnAvailable := "available"
	fnOnboarded := "onboarded"
	fnFull := "full"

	cmd := &cobra.Command{
		Use:   "capacity",
		Short: "Display capacity of device resources",
		Long:  `Retrieve capacity of the machine, onboarded or available amount of resources`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			table := setupTable(cmd.OutOrStdout())
			defer table.Render()

			resBody, resCode, err := client.MakeRequest("GET", "/onboarding/info", nil)
			if err != nil {
				return fmt.Errorf("could not make request: %w", err)
			}

			if resCode != 200 {
				return fmt.Errorf("request failed with status code: %d", resCode)
			}

			info, _, _, err := jsonparser.Get(resBody, "info")
			if err != nil {
				return fmt.Errorf("could not get info value by key: %w", err)
			}

			var config *types.OnboardingConfig
			if err := json.Unmarshal(info, &config); err != nil {
				return fmt.Errorf("could not unmarshal response body: %w", err)
			}

			full, _ := cmd.Flags().GetBool(fnFull)
			available, _ := cmd.Flags().GetBool(fnAvailable)
			onboarded, _ := cmd.Flags().GetBool(fnOnboarded)

			if full {
				table.Append(formatCapacityData("Full", &config.TotalResources.Resources))
			}

			if onboarded {
				table.Append(formatCapacityData("Onboarded", &config.OnboardedResources.Resources))
			}

			if available {
				resources := config.TotalResources
				if err := resources.Subtract(config.OnboardedResources.Resources); err != nil {
					return fmt.Errorf("no available resources: %w", err)
				}
				table.Append(formatCapacityData("Available", &resources.Resources))
			}
			return nil
		},
	}
	cmd.Flags().BoolP(fnFull, "f", false, "display device capacity")
	cmd.Flags().BoolP(fnAvailable, "a", false, "display amount of resources still available for onboarding")
	cmd.Flags().BoolP(fnOnboarded, "o", false, "display amount of onboarded resources")
	cmd.MarkFlagsOneRequired(fnAvailable, fnFull, fnOnboarded)
	return cmd
}

func formatCapacityData(resourceType string, resources *types.Resources) []string {
	return []string{
		resourceType,
		fmt.Sprintf("%d", resources.RAM.Size),
		fmt.Sprintf("%f", resources.CPU.Compute),
		fmt.Sprintf("%.2f", resources.CPU.Cores),
	}
}
