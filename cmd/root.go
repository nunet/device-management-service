package cmd

import (
	"fmt"

	"github.com/coreos/go-systemd/sdjournal"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/api/docs"
	"gitlab.com/nunet/device-management-service/cmd/backend"
	"gitlab.com/nunet/device-management-service/cmd/cap"
	"gitlab.com/nunet/device-management-service/executor/docker"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/utils"
)

func newRootCmd(client *utils.HTTPClient, afs afero.Afero, dockerExec *docker.Client, logger backend.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "nunet",
		Short:   "NuNet Device Management Service",
		Long:    `The Device Management Service (DMS) Command Line Interface (CLI)`,
		Version: docs.SwaggerInfo.Version,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: false,
			HiddenDefaultCmd:  true,
		},
		SilenceErrors: true,
		SilenceUsage:  true,
		Run: func(cmd *cobra.Command, _ []string) {
			_ = cmd.Help()
		},
	}
	cmd.AddCommand(newGPUCmd(afs))
	cmd.AddCommand(newOffboardCmd(client))
	cmd.AddCommand(newOnboardMLCmd(afs, dockerExec))
	cmd.AddCommand(newRunCmd())
	cmd.AddCommand(newPeerCmd(client))
	cmd.AddCommand(newOnboardCmd(client))
	cmd.AddCommand(newInfoCmd(client))
	cmd.AddCommand(newKeyCmd(afs))
	cmd.AddCommand(newDeviceCmd(client))
	cmd.AddCommand(newCapacityCmd(client))
	cmd.AddCommand(newResourceConfigCmd(client))
	cmd.AddCommand(newLogCmd(afs, logger))
	cmd.AddCommand(newWalletCmd(client))
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newAutoCompleteCmd())
	cmd.AddCommand(cap.NewCapCmd(afs))
	return cmd
}

// Execute is a wrapper for cobra.Command method with same name
// It makes use of cobra.CheckErr to facilitate error handling
func Execute() {
	config.LoadConfig()

	afs := afero.Afero{Fs: afero.NewOsFs()}

	client := utils.NewHTTPClient(
		fmt.Sprintf("http://%s:%d",
			config.GetConfig().Addr,
			config.GetConfig().Port),
		"/api/v1",
	)

	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		cobra.CheckErr(fmt.Errorf("couldn't instantiate new docker client; Error: %w", err))
	}

	journal, err := sdjournal.NewJournal()
	if err != nil {
		cobra.CheckErr(fmt.Errorf("failed to get new sdjournal; Error: %w", err))
	}
	cobra.CheckErr(newRootCmd(client, afs, dockerClient, journal).Execute())
}
