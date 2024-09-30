//go:build darwin
// +build darwin

package cmd

import (
	"fmt"

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
			config.GetConfig().Rest.Addr,
			config.GetConfig().Rest.Port),
		"/api/v1",
	)

	cobra.CheckErr(newRootCmd(client, afs).Execute())
}
