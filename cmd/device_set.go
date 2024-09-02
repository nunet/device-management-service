package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/utils"
)

// NewDeviceSetCmd is a constructor for `device set` subcommand
func newDeviceSetCmd(client *utils.HTTPClient) *cobra.Command {
	validArgs := []string{"online", "offline"}
	return &cobra.Command{
		Use:       "set",
		Short:     "Set device online status",
		ValidArgs: validArgs,
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Long:      ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			onboarded, err := checkOnboarded(client)
			if err != nil {
				return fmt.Errorf("could not check onboard status: %w", err)
			}
			if !onboarded {
				return fmt.Errorf("machine is not onboarded")
			}

			var statusJSON []byte
			if args[0] == "online" {
				statusJSON = []byte(`{"is_available": true}`)
			} else if args[0] == "offline" {
				statusJSON = []byte(`{"is_available": false}`)
			}

			resBody, resCode, err := client.MakeRequest("POST", "/device/status", statusJSON)
			if err != nil {
				return fmt.Errorf("could not make request: %w", err)
			}

			if resCode != 201 {
				return fmt.Errorf("request failed with status code: %d", resCode)
			}

			var response map[string]any
			if err := json.Unmarshal(resBody, &response); err != nil {
				return fmt.Errorf("could not unmarshal response body: %w", err)
			}

			msg, ok := response["message"]
			if ok {
				fmt.Fprintln(cmd.OutOrStdout(), msg)
			}
			return nil
		},
	}
}
