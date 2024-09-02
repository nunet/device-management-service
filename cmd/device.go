package cmd

import (
	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/utils"
)

// NewDeviceCmd is a constructor for `device` parent command
func newDeviceCmd(client *utils.HTTPClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "device",
		Short: "device related operations",
		Long:  `manage onboarded device`,
		Run: func(cmd *cobra.Command, _ []string) {
			err := cmd.Help()
			if err != nil {
				cmd.Println(err)
			}
		},
	}
	cmd.AddCommand(newDeviceStatusCmd(client))
	cmd.AddCommand(newDeviceSetCmd(client))
	return cmd
}
