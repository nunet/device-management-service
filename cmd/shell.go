package cmd

import "github.com/spf13/cobra"

var flagNode string

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Send commands to DMS instance",
	Long:  "",
	Run: func(_ *cobra.Command, _ []string) {
	},
}
