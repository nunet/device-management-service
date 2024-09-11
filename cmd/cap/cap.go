package cap

import (
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const (
	flagContext  = "context"
	flagAudience = "audience"
	flagAction   = "action"
	flagCap      = "cap"
	flagTopic    = "topic"
	flagExpire   = "expire"
	flagDuration = "duration"
	flagDepth    = "depth"
	flagProvide  = "provide"
	flagRoot     = "root"
	flagRequire  = "require"
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
