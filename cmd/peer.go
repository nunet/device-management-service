package cmd

import (
	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/utils"
)

func newPeerCmd(client *utils.HTTPClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "peer",
		Short: "Peer-related operations",
		Long:  ``,
		Run: func(cmd *cobra.Command, _ []string) {
			_ = cmd.Help()
		},
	}
	cmd.AddCommand(newPeerListCmd(client))
	cmd.AddCommand(newPeerSelfCmd(client))
	return cmd
}
