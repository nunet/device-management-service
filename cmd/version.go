package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Display the Nunet DMS version",
		Long:  `This command prints the version of the Nunet Device Management Service.`,
		Run: func(_ *cobra.Command, _ []string) {
			// TODO get the version from git; make a top level version.go file
			fmt.Println("Nunet Device Management Service Version: v0.5-boot")
		},
	}
}
