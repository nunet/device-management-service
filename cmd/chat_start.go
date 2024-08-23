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

var chatStartCmd = NewChatStartCmd(p2pService, utilsService, webSocketClient)

func NewChatStartCmd(p2pService backend.PeerManager, _ backend.Utility, wsClient backend.WebSocketClient) *cobra.Command {
	return &cobra.Command{
		Use:     "start",
		Short:   "Start chat with a peer",
		Example: "nunet chat start <peerID>",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.SetOutput(cmd.OutOrStderr())

			err := validateStartChatInput(p2pService, args)
			if err != nil {
				return fmt.Errorf("start chat failed: %w", err)
			}

			query := "peerID=" + args[0]

			startURL, err := utils.InternalAPIURL("ws", "/api/v1/peers/chat/start", query)
			if err != nil {
				return fmt.Errorf("could not compose WebSocket URL: %w", err)
			}

			err = wsClient.Initialize(startURL)
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
