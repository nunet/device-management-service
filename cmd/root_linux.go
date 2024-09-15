//go:build linux
// +build linux

package cmd

import (
	"fmt"

	"github.com/coreos/go-systemd/sdjournal"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/utils"
)

// Execute is a wrapper for cobra.Command method with same name
// It makes use of cobra.CheckErr to facilitate error handling
func Execute() {
	afs := afero.Afero{Fs: afero.NewOsFs()}

	client := utils.NewHTTPClient(
		fmt.Sprintf("http://%s:%d",
			config.GetConfig().Addr,
			config.GetConfig().Port),
		"/api/v1",
	)

	journal, err := sdjournal.NewJournal()
	if err != nil {
		cobra.CheckErr(fmt.Errorf("failed to get new sdjournal; Error: %w", err))
	}
	cobra.CheckErr(newRootCmd(client, afs, journal).Execute())
}
