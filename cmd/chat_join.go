package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/cmd/backend"
	"gitlab.com/nunet/device-management-service/utils"
)

var chatJoinCmd = NewChatJoinCmd(utilsService, webSocketClient)

// XXX NewChatJoinCmd and NewChatStartCmd are similar, consider refactoring
func NewChatJoinCmd(utilsService backend.Utility, wsClient backend.WebSocketClient) *cobra.Command {
	return &cobra.Command{
		Use:     "join",
		Short:   "Join open chat stream",
		Example: "nunet chat join <chatID>",
		Long:    "",
		RunE: func(cmd *cobra.Command, args []string) error {
			chatList, err := utilsService.ResponseBody(nil, "GET", "/api/v1/peers/chat", "", nil)
			if err != nil {
				return fmt.Errorf("could not obtain incoming chats list: %w", err)
			}

			err = validateJoinChatInput(args, chatList)
			if err != nil {
				return fmt.Errorf("join chat failed: %w", err)
			}

			query := "id=" + args[0]

			joinURL, err := utils.InternalAPIURL("ws", "/api/v1/peers/chat/join", query)
			if err != nil {
				return fmt.Errorf("could not compose WebSocket URL: %w", err)
			}

			log.SetOutput(cmd.OutOrStderr())

			err = wsClient.Initialize(joinURL)
			if err != nil {
				return fmt.Errorf("failed to initialize WebSocket client: %w", err)
			}

			ctx, cancel := context.WithCancel(context.Background())

			defer func() {
				cancel()
				wsClient.Close()
			}()

			go func() {
				interrupt := make(chan os.Signal, 1)
				signal.Notify(interrupt, os.Interrupt)
				<-interrupt
				cancel()
				wsClient.Close()
			}()

			var wg sync.WaitGroup
			wg.Add(3)

			go func() {
				defer wg.Done()

				err := wsClient.ReadMessage(ctx, cmd.OutOrStdout())
				if err != nil {
					log.Printf("Error: %v\n", err)
				}
			}()
			go func() {
				defer wg.Done()

				err := wsClient.WriteMessage(ctx, cmd.InOrStdin())
				if err != nil {
					log.Printf("Error: %v\n", err)
				}
			}()

			go func() {
				defer wg.Done()

				err := wsClient.Ping(ctx, cmd.OutOrStderr())
				if err != nil {
					log.Printf("Error: %v\n", err)
				}
			}()

			wg.Wait()
			return nil
		},
	}
}
