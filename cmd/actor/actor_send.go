package actor

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/utils"
)

// NewActorSendCmd is a constructor for `actor send` subcommand
func newActorSendCmd(client *utils.HTTPClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send <msg>",
		Short: "Send a message",
		Long: `Send a message to an actor

Actors only communicate via messages. For more information on constructing a message, see:

  nunet actor msg --help

The message is encoded into an actor envelope, which then is sent across the network through the API.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var msg actor.Envelope

			if err := json.Unmarshal([]byte(args[0]), &msg); err != nil {
				return fmt.Errorf("could not unmarshal message: %w", err)
			}

			resBody, resCode, err := client.MakeRequest("POST", "/actor/send", []byte(args[0]))
			fmt.Fprintln(cmd.OutOrStdout(), string(resBody))
			if err != nil {
				return fmt.Errorf("unable to make internal request: %w", err)
			}
			if resCode != 200 {
				return fmt.Errorf("request failed with status code: %d", resCode)
			}

			return nil
		},
	}
	return cmd
}
