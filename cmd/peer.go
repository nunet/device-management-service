package cmd

import (
	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/cmd/backend"
)

var peerCmd = NewPeerCmd(networkService)

func NewPeerCmd(net backend.NetworkManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "peer",
		Short:             "Peer-related operations",
		Long:              ``,
		PersistentPreRunE: isDMSRunning(net),
		Run: func(cmd *cobra.Command, _ []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(peerListCmd)
	cmd.AddCommand(peerSelfCmd)
	cmd.AddCommand(peerDefaultCmd)
	return cmd
}
