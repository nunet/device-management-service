package cmd

import (
	"fmt"

	"github.com/buger/jsonparser"
	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/cmd/backend"
)

var peerSelfCmd = NewPeerSelfCmd(utilsService)

func NewPeerSelfCmd(utilsService backend.Utility) *cobra.Command {
	return &cobra.Command{
		Use:   "self",
		Short: "Display self peer info",
		Long:  ``,
		RunE: func(cmd *cobra.Command, _ []string) error {
			err := checkOnboarded(utilsService)
			if err != nil {
				return err
			}

			body, err := utilsService.ResponseBody(nil, "GET", "/api/v1/peers/self", "", nil)
			if err != nil {
				return fmt.Errorf("unable to get response body: %w", err)
			}

			id, err := selfPeerID(body)
			if err != nil {
				return fmt.Errorf("failed to fetch self peer ID: %w", err)
			}

			addrsByte, err := selfPeerAddrs(body)
			if err != nil {
				return fmt.Errorf("failed to fetch self peer addresses: %w", err)
			}

			var addrs []string

			// iterate through array and append each value to slice
			_, err = jsonparser.ArrayEach(addrsByte, func(value []byte, _ jsonparser.ValueType, _ int, _ error) {
				addrs = append(addrs, string(value))
			})
			if err != nil {
				return fmt.Errorf("failed to itterate through parser: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Host ID:", id)
			for index, addr := range addrs {
				if index+1 == len(addrs) { // if element is the last one
					fmt.Fprintf(cmd.OutOrStdout(), "%s\n", addr)
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "%s, ", addr)
				}
			}

			return nil
		},
	}
}
