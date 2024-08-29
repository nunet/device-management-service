package cmd

import (
	"fmt"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/cmd/backend"
)

// support for bitwise operations according to each flag
const (
	bitFullFlag      = 1 << iota // 1 (001)
	bitAvailableFlag             // 2 (010)
	bitOnboardedFlag             // 4 (100)
)

var (
	capacityCmd                            = NewCapacityCmd(networkService, resourceService, utilsService)
	flagFull, flagAvailable, flagOnboarded bool
)

func NewCapacityCmd(net backend.NetworkManager, resources backend.ResourceManager, utilsService backend.Utility) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "capacity",
		Short:   "Display capacity of device resources",
		Long:    `Retrieve capacity of the machine, onboarded or available amount of resources`,
		PreRunE: isDMSRunning(net),
		RunE: func(cmd *cobra.Command, _ []string) error {
			onboarded, _ := cmd.Flags().GetBool("onboarded")
			full, _ := cmd.Flags().GetBool("full")
			available, _ := cmd.Flags().GetBool("available")

			// flagCombination stores the bitwise value of combined flags
			var flagCombination int
			if full {
				flagCombination |= bitFullFlag
			}
			if available {
				flagCombination |= bitAvailableFlag
			}
			if onboarded {
				flagCombination |= bitOnboardedFlag
			}

			if flagCombination == 0 {
				return cmd.Help()
			}

			var table *tablewriter.Table
			// setups table in case of full, available or onboarded flags are passed
			if flagCombination&bitFullFlag != 0 || flagCombination&bitAvailableFlag != 0 || flagCombination&bitOnboardedFlag != 0 {
				table = setupTable(cmd.OutOrStdout())
			}

			if flagCombination&bitFullFlag != 0 {
				handleFull(table, resources)
			}
			// nolint
			if flagCombination&bitAvailableFlag != 0 {
				// XXX done in !430
			}
			if flagCombination&bitOnboardedFlag != 0 {
				err := handleOnboarded(table, utilsService)
				if err != nil {
					return fmt.Errorf("cannot fetch onboarded data: %w", err)
				}
			}

			if table != nil {
				table.Render()
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&flagFull, "full", "f", false, "display device ")
	cmd.Flags().BoolVarP(&flagAvailable, "available", "a", false, "display amount of resources still available for onboarding")
	cmd.Flags().BoolVarP(&flagOnboarded, "onboarded", "o", false, "display amount of onboarded resources")
	return cmd
}
