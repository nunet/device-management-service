package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/utils"
)

// NewOffboardCmd is a constructor for `offboard` command
func newOffboardCmd(client *utils.HTTPClient) *cobra.Command {
	fnForce := "force"
	cmd := &cobra.Command{
		Use:   "offboard",
		Short: "Offboard the device from NuNet",
		Long:  ``,
		RunE: func(cmd *cobra.Command, _ []string) error {
			onboarded, err := checkOnboarded(client)
			if err != nil {
				return fmt.Errorf("could not check onboard status: %w", err)
			}
			if !onboarded {
				return fmt.Errorf("machine is not onboarded")
			}

			fmt.Println("Warning: Offboarding will remove all your data and you will not be able to onboard again with the same identity")
			answer, err := utils.PromptYesNo(cmd.InOrStdin(), cmd.OutOrStdout(), "Are you sure you want to offboard?")
			if err != nil {
				return fmt.Errorf("unable to read response: %w", err)
			}
			if !answer {
				return nil
			}

			force, _ := cmd.Flags().GetBool(fnForce)
			query := fmt.Sprintf("?force=%t", force)

			resBody, resCode, err := client.MakeRequest("DELETE", "/onboarding/offboard"+query, nil)
			if err != nil {
				return fmt.Errorf("could not make request: %w", err)
			}

			if resCode != 200 {
				return fmt.Errorf("request failed with status code: %d", resCode)
			}

			// TODO:what to do with the response body?
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n", resBody)
			return nil
		},
	}
	cmd.Flags().BoolP(fnForce, "f", false, "force offboarding")
	return cmd
}
