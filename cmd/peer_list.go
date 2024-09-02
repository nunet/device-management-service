package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/utils"
)

// NewPeerListCmd is a constructor for `peer list` subcommand
func newPeerListCmd(client *utils.HTTPClient) *cobra.Command {
	fnDHT := "dht"
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Display list of peers in the network",
		Long:  ``,
		RunE: func(cmd *cobra.Command, _ []string) error {
			onboarded, err := checkOnboarded(client)
			if err != nil {
				return fmt.Errorf("could not check onboard status: %w", err)
			}
			if !onboarded {
				return fmt.Errorf("machine is not onboarded")
			}

			dht, _ := cmd.Flags().GetBool(fnDHT)
			if !dht {
				bootPeer, err := getBootstrapPeers(cmd.OutOrStdout(), client)
				if err != nil {
					return fmt.Errorf("could not fetch bootstrap peers: %w", err)
				}

				fmt.Fprintf(cmd.OutOrStdout(), "Bootstrap peers (%d)\n", len(bootPeer))
				for _, b := range bootPeer {
					fmt.Fprintf(cmd.OutOrStdout(), "%s\n", b)
				}

				fmt.Fprintf(cmd.OutOrStdout(), "\n")
			}

			dhtPeer, err := getDHTPeers(client)
			if err != nil {
				return fmt.Errorf("could not fetch DHT peers: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "DHT peers (%d)\n", len(dhtPeer))
			for _, d := range dhtPeer {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n", d)
			}

			return nil
		},
	}
	cmd.Flags().BoolP(fnDHT, "d", false, "list only DHT peers")
	return cmd
}
