package cap

import (
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const (
	fnContext  = "context"
	fnAudience = "audience"
	fnAction   = "action"
	fnCap      = "cap"
	fnTopic    = "topic"
	fnExpiry   = "expiry"
	fnDuration = "duration"
	fnDepth    = "depth"
	fnProvide  = "provide"
	fnRoot     = "root"
	fnRequire  = "require"
)

// NewCapCmd returns the cap command that adds other commands
func NewCapCmd(afs afero.Afero) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cap",
		Short: "Manage capabilities",
		Long:  `Manage capabilities for the Device Management Service`,
	}

	cmd.AddCommand(newGrantCmd(afs))
	cmd.AddCommand(newAnchorCmd(afs))
	cmd.AddCommand(newNewCmd(afs))
	cmd.AddCommand(newDelegateCmd(afs))

	return cmd
}
