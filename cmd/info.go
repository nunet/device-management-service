package cmd

import (
	"fmt"
	"io"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"gitlab.com/nunet/device-management-service/cmd/backend"
	gdb "gitlab.com/nunet/device-management-service/db/repositories/gorm"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/types"
)

var infoCmd = NewInfoCmd(networkService, utilsService)

func NewInfoCmd(net backend.NetworkManager, utilsService backend.Utility) *cobra.Command {
	return &cobra.Command{
		Use:     "info",
		Short:   "Display information about onboarded device",
		Long:    "Display onboarding config of onboarded device on Nunet Device Management Service",
		PreRunE: isDMSRunning(net),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := checkOnboarded(utilsService)
			if err != nil {
				return err
			}

			// XXX: don't leave me like this
			db, err := gorm.Open(sqlite.Open(fmt.Sprintf("%s/nunet.db", config.GetConfig().General.WorkDir)), &gorm.Config{})
			if err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}

			onboardR := gdb.NewOnboardingParamsRepository(db)
			oConf, err := onboardR.Get(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to get onboarding config: %w", err)
			}

			displayInTable(cmd.OutOrStdout(), &oConf)

			return nil
		},
	}
}

func displayInTable(w io.Writer, oConf *types.OnboardingConfig) {
	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"Info", "Value"})

	table.Append([]string{"Name", oConf.Name})
	table.Append([]string{"Update Timestamp", fmt.Sprintf("%d", oConf.UpdateTimestamp)})
	table.Append([]string{"Memory Max", fmt.Sprintf("%d", oConf.Resource.MemoryMax)})
	table.Append([]string{"Total Core", fmt.Sprintf("%d", oConf.Resource.TotalCore)})
	table.Append([]string{"CPU Max", fmt.Sprintf("%d", oConf.Resource.CPUMax)})
	table.Append([]string{"Available CPU", fmt.Sprintf("%d", oConf.Available.CPU)})
	table.Append([]string{"Available Memory", fmt.Sprintf("%d", oConf.Available.Memory)})
	table.Append([]string{"Reserved CPU", fmt.Sprintf("%d", oConf.Reserved.CPU)})
	table.Append([]string{"Reserved Memory", fmt.Sprintf("%d", oConf.Reserved.Memory)})
	table.Append([]string{"Network", oConf.Network})
	table.Append([]string{"Public Key", oConf.PublicKey})
	table.Append([]string{"Node ID", oConf.NodeID})
	table.Append([]string{"Allow Cardano", fmt.Sprintf("%t", oConf.AllowCardano)})
	table.Append([]string{"Dashboard", oConf.Dashboard})
	table.Append([]string{"NTX Price Per Minute", fmt.Sprintf("%f", oConf.NTXPricePerMinute)})

	table.Render()
}
