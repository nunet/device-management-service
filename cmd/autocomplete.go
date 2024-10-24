// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

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
