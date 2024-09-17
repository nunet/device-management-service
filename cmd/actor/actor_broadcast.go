package actor

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/utils"
)

func newActorBroadcastCmd(client *utils.HTTPClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "broadcast <msg>",
		Short: "Broadcast a message",
		Long: `Broadcast a message to a topic

If a topic is specified in the message's payload, the message will be published to all subscribers of that topic.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var msg actor.Envelope

			if err := json.Unmarshal([]byte(args[0]), &msg); err != nil {
				return fmt.Errorf("could not unmarshal message: %w", err)
			}

			resBody, resCode, err := client.MakeRequest("POST", "/actor/broadcast", []byte(args[0]))
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
