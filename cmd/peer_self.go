package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/buger/jsonparser"
	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/utils"
)

// NewPeerSelfCmd is a constructor for `peer self` subcommand
func newPeerSelfCmd(client *utils.HTTPClient) *cobra.Command {
	return &cobra.Command{
		Use:   "self",
		Short: "Display self peer info",
		Long:  ``,
		RunE: func(cmd *cobra.Command, _ []string) error {
			onboarded, err := checkOnboarded(client)
			if err != nil {
				return fmt.Errorf("could not check onboard status: %w", err)
			}
			if !onboarded {
				return fmt.Errorf("machine is not onboarded")
			}

			resBody, resCode, err := client.MakeRequest("GET", "/peers/self", nil)
			if err != nil {
				return fmt.Errorf("unable to make internal request: %w", err)
			}

			if resCode != 200 {
				return fmt.Errorf("request failed with status code: %d", resCode)
			}

			var response map[string]any
			if err := json.Unmarshal(resBody, &response); err != nil {
				return fmt.Errorf("could not unmarshal response body: %w", err)
			}

			id, ok := response["id"]
			if !ok {
				return fmt.Errorf("no self peer ID returned")
			}

			addrsByte, _, _, err := jsonparser.Get(resBody, "listen_addr")
			if err != nil {
				return fmt.Errorf("failed to get addresses field: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Host ID:", id)
			fmt.Fprintln(cmd.OutOrStdout(), "Listening Addresses:", string(addrsByte))

			return nil
		},
	}
}
