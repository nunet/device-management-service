//go:build darwin

package cmd

import (
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func newLogCmd(afs afero.Afero, loggerArg interface{}) *cobra.Command {
	return &cobra.Command{
		Use:   "log",
		Short: "Gather all logs into a tarball",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), "Log collection on MacOS is not yet supported.")
		},
	}
}
