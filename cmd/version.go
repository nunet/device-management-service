package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version   string
	GoVersion string
	BuildDate string
	Commit    string
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Information about current version",
		Long:  `Display information about the current Device Management Service (DMS) version`,
		Run: func(_ *cobra.Command, _ []string) {
			// TODO get the version from git; make a top level version.go file
			fmt.Println("NuNet Device Management Service")
			fmt.Printf("Version: %s\nCommit: %s\n\nGo Version: %s\nBuild Date: %s\n",
				Version, Commit, GoVersion, BuildDate)
		},
	}
}
