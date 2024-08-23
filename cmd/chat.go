package cmd

import (
	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/cmd/backend"
)

var chatCmd = NewChatCmd(networkService)

func NewChatCmd(net backend.NetworkManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "chat",
		Short:             "Chat-related operations",
		PersistentPreRunE: isDMSRunning(net),
		Run: func(cmd *cobra.Command, _ []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(chatClearCmd)
	cmd.AddCommand(chatJoinCmd)
	cmd.AddCommand(chatListCmd)
	cmd.AddCommand(chatStartCmd)
	return cmd
}
