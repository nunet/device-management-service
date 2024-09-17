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
		Short: "Interact with the actor system",
		Long: `Interact with the actor system

Actors are the entities which compose the NuActor system, a secure decentralized programming framework based on the Actor Model.
Actors are connected through the libp2p network substrate and communication is achieved via immutable messages.

For more information on the actor system, please refer to actor/README.md`,
	}
	cmd.AddCommand(newActorMsgCmd(client, afs))
	cmd.AddCommand(newActorSendCmd(client))
	cmd.AddCommand(newActorInvokeCmd(client))
	cmd.AddCommand(newActorBroadcastCmd(client))
	cmd.AddCommand(newActorCmdGroup(client, afs))
	return cmd
}
