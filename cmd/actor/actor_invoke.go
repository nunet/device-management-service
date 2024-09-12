package actor

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/dms/actor"
	"gitlab.com/nunet/device-management-service/utils"
)

// NewActorInvokeCmd is a constructor for `actor invoke` subcommand
func newActorInvokeCmd(client *utils.HTTPClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invoke <msg>",
		Short: "Invoke a behaviour in an actor and return the result",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var msg actor.Envelope

			if err := json.Unmarshal([]byte(args[0]), &msg); err != nil {
				return fmt.Errorf("could not unmarshal message: %w", err)
			}

			if msg.Options.ReplyTo == "" {
				return fmt.Errorf("missing replyTo field in message")
			}

			resBody, resCode, err := client.MakeRequest("POST", "/actor/invoke", []byte(args[0]))
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
