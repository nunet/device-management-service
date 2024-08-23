package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/cmd/backend"
)

var (
	peerListCmd = NewPeerListCmd(utilsService)
	flagD       bool
)

func NewPeerListCmd(utilsService backend.Utility) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Display list of peers in the network",
		Long:  ``,
		RunE: func(cmd *cobra.Command, _ []string) error {
			err := checkOnboarded(utilsService)
			if err != nil {
				return err
			}

			dhtFlag, _ := cmd.Flags().GetBool("dht")

			if !dhtFlag {
				bootPeer, err := getBootstrapPeers(cmd.OutOrStderr(), utilsService)
				if err != nil {
					return fmt.Errorf("could not fetch bootstrap peers: %w", err)
				}

				fmt.Fprintf(cmd.OutOrStdout(), "Bootstrap peers (%d)\n", len(bootPeer))
				for _, b := range bootPeer {
					fmt.Fprintf(cmd.OutOrStdout(), "%s\n", b)
				}

				fmt.Fprintf(cmd.OutOrStdout(), "\n")
			}

			dhtPeer, err := getDHTPeers(utilsService)
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

	cmd.Flags().BoolVarP(&flagD, "dht", "d", false, "list only DHT peers")
	return cmd
}
