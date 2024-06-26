package cmd

import (
	"fmt"

	"github.com/buger/jsonparser"
	"github.com/ethereum/go-ethereum/log"
	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/cmd/backend"
)

var chatListCmd = NewChatListCmd(utilsService)

func NewChatListCmd(utilsService backend.Utility) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Display table of open chat streams",
		Long:  "",
		RunE: func(cmd *cobra.Command, args []string) error {
			chatBody, err := utilsService.ResponseBody(nil, "GET", "/api/v1/peers/chat", "", nil)
			if err != nil {
				return fmt.Errorf("unable to get chat list response body: %w", err)
			}

			errMsg, err := jsonparser.GetString(chatBody, "error")
			if err == nil {
				return fmt.Errorf(errMsg)
			}

			// chatList, err := getIncomingChatList(chatBody)
			// if err != nil {
			// 	return fmt.Errorf("could not get incoming chat list: %w", err)
			// }

			table := setupChatTable(cmd.OutOrStdout())

			// for _, chat := range chatList {
			// 	table.Append([]string{strconv.Itoa(chat.ID), chat.StreamID, chat.FromPeer, chat.TimeOpened})
			// }
			log.Warn("chat list has incomplete implementation")

			table.Render()
			return nil
		},
	}
}
