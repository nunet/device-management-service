package cmd

import (
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/cmd/actor"
	"gitlab.com/nunet/device-management-service/cmd/cap"
	"gitlab.com/nunet/device-management-service/utils"
)

func newRootCmd(client *utils.HTTPClient, afs afero.Afero) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nunet",
		Short: "NuNet Device Management Service",
		Long:  `The Device Management Service (DMS) Command Line Interface (CLI)`,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: false,
			HiddenDefaultCmd:  true,
		},
		SilenceErrors: true,
		SilenceUsage:  true,
		Run: func(cmd *cobra.Command, _ []string) {
			_ = cmd.Help()
		},
	}
	cmd.AddCommand(newRunCmd())
	cmd.AddCommand(newKeyCmd(afs))
	cmd.AddCommand(cap.NewCapCmd(afs))
	cmd.AddCommand(actor.NewActorCmd(client, afs))
	cmd.AddCommand(newConfigCmd(afs.Fs))
	cmd.AddCommand(newAutoCompleteCmd())
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newTapCommand())
	cmd.AddCommand(newGPUCommand())
	return cmd
}
