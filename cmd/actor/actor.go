package actor

import (
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/utils"
)

const (
	CapstoreDir            = "cap/"
	DefaultUserContextName = "user"
	KeystoreDir            = "key/"
)

// NewActorCmd is a constructor for `actor` parent command
func NewActorCmd(client *utils.HTTPClient, afs afero.Afero) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "actor",
		Short: "Actor related operations",
		Long:  "Interact with the actor system",
		Run: func(cmd *cobra.Command, _ []string) {
			err := cmd.Help()
			if err != nil {
				cmd.Println(err)
			}
		},
	}
	cmd.AddCommand(newActorMsgCmd(client, afs))
	cmd.AddCommand(newActorSendCmd(client))
	cmd.AddCommand(newActorInvokeCmd(client))
	cmd.AddCommand(newActorBroadcastCmd(client))
	cmd.AddCommand(newActorCmdGroup(client, afs))
	return cmd
}
