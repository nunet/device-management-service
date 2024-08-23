package cmd

import (
	"fmt"

	"github.com/buger/jsonparser"
	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/cmd/backend"
)

var deviceCmd = NewDeviceCmd(networkService)

var deviceStatusCmd = NewDeviceStatusCmd(utilsService)

var deviceSetCmd = NewDeviceSetCmd(utilsService)

func NewDeviceCmd(net backend.NetworkManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "device",
		Short:             "device related operations",
		Long:              `manage onboarded device`,
		PersistentPreRunE: isDMSRunning(net),
		Run: func(cmd *cobra.Command, _ []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(deviceStatusCmd)
	cmd.AddCommand(deviceSetCmd)
	return cmd
}

func NewDeviceStatusCmd(utilsService backend.Utility) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Display current device status",
		Long:  ``,
		RunE: func(cmd *cobra.Command, _ []string) error {
			err := checkOnboarded(utilsService)
			if err != nil {
				return err
			}

			body, err := utilsService.ResponseBody(nil, "GET", "/api/v1/device/status", "", nil)
			if err != nil {
				return fmt.Errorf("unable to get /device/status response body: %w", err)
			}

			online, err := jsonparser.GetBoolean(body, "online")
			if err != nil {
				return fmt.Errorf("failed to get device status from json response: %w", err)
			}

			if online {
				fmt.Fprintln(cmd.OutOrStdout(), "Status: online")
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Status: offline")
			}

			return nil
		},
	}
}

func NewDeviceSetCmd(utilsService backend.Utility) *cobra.Command {
	return &cobra.Command{
		Use:   "set",
		Short: "Set device online status",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := checkOnboarded(utilsService)
			if err != nil {
				return err
			}

			if len(args) != 1 {
				return fmt.Errorf("invalid number of arguments")
			}

			var statusJSON []byte

			switch args[0] {
			case "online":
				statusJSON = []byte(`{"is_available": true}`)
			case "offline":
				statusJSON = []byte(`{"is_available": false}`)
			default:
				return fmt.Errorf("invalid argument")
			}

			body, err := utilsService.ResponseBody(nil, "POST", "/api/v1/device/status", "", statusJSON)
			if err != nil {
				return fmt.Errorf("could not get response body: %w", err)
			}

			// check if error response
			errResponse, err := jsonparser.GetString(body, "error")
			if err == nil {
				return fmt.Errorf("%s", errResponse)
			}

			// if no error
			response, err := jsonparser.GetString(body, "message")
			if err != nil {
				return fmt.Errorf("failed to get device status from json response: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), response)

			return nil
		},
	}
}
