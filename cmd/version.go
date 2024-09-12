package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is the version of the Nunet Device Management Service
// TODO: use git describe after this release
var Version = "v0.5.0-boot"

var (
	GoVersion string
	BuildDate string
	Commit    string
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Display the Nunet DMS version",
		Long:  `This command prints the version of the Nunet Device Management Service.`,
		Run: func(_ *cobra.Command, _ []string) {
			// TODO get the version from git; make a top level version.go file
			fmt.Println("Nunet Device Management Service")
			fmt.Printf("Version: %s\nCommit: %s\n\nGo Version: %s\nBuild Date: %s\n",
				Version, Commit, GoVersion, BuildDate)
		},
	}
}
