package cmd

import (
	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/dms"
)

func newRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Start the Device Management Service",
		Long:  `The Device Management Service (DMS) is a system application for computing and service providers. It handles networking and device management.`,
		Run: func(_ *cobra.Command, _ []string) {
			dms.Run()
		},
	}
}
