package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// autocompleteCmd represents the command to generate shell autocompletion scripts
func newAutoCompleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "autocomplete [shell]",
		Short: "Generate autocomplete script for your shell",
		Long: `Generate an autocomplete script for the nunet CLI.
This command supports Bash and Zsh shells.`,
		DisableFlagsInUseLine: true,
		Hidden:                true,
		ValidArgs:             []string{"bash", "zsh"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				_ = cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				_ = cmd.Root().GenZshCompletion(os.Stdout)
			}
		},
	}
}
