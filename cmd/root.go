package cmd

import (
	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/api/docs"
)

var rootCmd = &cobra.Command{
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

func Execute() {
	// CheckErr prints formatted error message, if there is any, and exits
	cobra.CheckErr(rootCmd.Execute())
}
