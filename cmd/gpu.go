package cmd

import (
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func newGPUCmd(afs afero.Afero) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gpu",
		Short: "GPU-related operations",
		Long:  ``,
		Run: func(cmd *cobra.Command, _ []string) {
			err := cmd.Help()
			if err != nil {
				cmd.Println(err)
			}
		},
	}
	cmd.AddCommand(newGPUCapacityCmd())
	cmd.AddCommand(newGPUStatusCmd())
	cmd.AddCommand(newGPUOnboardCmd(afs))
	return cmd
}
