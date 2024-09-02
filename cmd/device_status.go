package cmd

import (
	"fmt"

	"github.com/buger/jsonparser"
	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/utils"
)

// NewDeviceStatusCmd is a constructor for `device status` subcommand
func newDeviceStatusCmd(client *utils.HTTPClient) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Display current device status",
		Args:  cobra.NoArgs,
		Long:  ``,
		RunE: func(cmd *cobra.Command, _ []string) error {
			onboarded, err := checkOnboarded(client)
			if err != nil {
				return fmt.Errorf("could not check onboard status: %w", err)
			}
			if !onboarded {
				return fmt.Errorf("machine is not onboarded")
			}

			resBody, resCode, err := client.MakeRequest("GET", "/device/status", nil)
			if err != nil {
				return fmt.Errorf("unable to make request: %w", err)
			}

			if resCode != 200 {
				return fmt.Errorf("request failed with status code: %d", resCode)
			}

			online, err := jsonparser.GetBoolean(resBody, "online")
			if err != nil {
				return fmt.Errorf("failed to get device status from json response: %w", err)
			}

			if online {
				fmt.Fprintln(cmd.OutOrStdout(), "Status: online")
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Status: offline")
			}
			return nil
		},
	}
}
