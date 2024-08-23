package cmd

import "github.com/spf13/cobra"

var gpuCmd = &cobra.Command{
	Use:   "gpu",
	Short: "GPU-related operations",
	Long:  ``,
	Run: func(cmd *cobra.Command, _ []string) {
		_ = cmd.Help()
	},
}
