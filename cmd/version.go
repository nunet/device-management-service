package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/api/docs"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Display the Nunet DMS version",
		Long:  `This command prints the version of the Nunet Device Management Service.`,
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("Nunet Device Management Service Version: %s\n", docs.SwaggerInfo.Version)
		},
	}
}
