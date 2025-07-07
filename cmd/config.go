// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cmd

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/cmd/cli"
	"gitlab.com/nunet/device-management-service/internal/config"
)

func newConfigCmd(dmsCli *cli.DmsCLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration file",
		Long: `Utility to manage user's configuration file via command-line

Search for the configuration file is done in the following locations and order:

1. "." (current directory)
2. "$HOME/.nunet"
3. "/etc/nunet"`,
	}
	cmd.AddCommand(newConfigGetCmd(dmsCli))
	cmd.AddCommand(newConfigSetCmd(dmsCli))
	cmd.AddCommand(newConfigEditCmd(dmsCli))
	return cmd
}

func newConfigGetCmd(dmsCli *cli.DmsCLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Display configuration",
		Long: `Display the value for a configuration key

It reads the value from configuration file, otherwise it return default values

Example:
  nunet config get rest.port`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ldr := dmsCli.ConfigLoader()
			_ = ensureConfigFile(dmsCli.FS(), ldr)
			cfg, err := ldr.GetConfig()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			cmd.Println("Found config file at:", ldr.ConfigFile())

			// No key  print the whole struct as JSON
			if len(args) == 0 {
				all, err := json.MarshalIndent(cfg, "", "    ")
				if err != nil {
					return fmt.Errorf("indent config JSON: %w", err)
				}
				cmd.Println(string(all))
				return nil
			}

			val, found := ldr.GetValue(strings.ToLower(args[0]))
			if !found {
				return fmt.Errorf("key %q not found", args[0])
			}

			pretty, _ := json.MarshalIndent(val, "", "    ")
			cmd.Println(string(pretty))
			return nil
		},
	}
	return cmd
}

func newConfigSetCmd(dmsCli *cli.DmsCLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Update configuration",
		Long: `Set value for a configuration key.

Creates the configuration file if it does not yet exist.

Examples:
  nunet config set rest.port 4444
  nunet config set general.work_dir ~/.config/dms`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := strings.ToLower(args[0])
			raw := args[1]

			ldr := dmsCli.ConfigLoader()
			_ = ensureConfigFile(dmsCli.FS(), ldr)

			exists, err := afero.Exists(dmsCli.FS(), ldr.ConfigFile())
			if err != nil {
				return fmt.Errorf("stat config file: %w", err)
			}
			if !exists {
				cmd.Println("Config file did not exist. Creating new file...")
			} else {
				cmd.Println("Updating existing config file...")
			}

			// Parse numeric and bool literals, keep string otherwise.
			value := parseLiteral(raw)

			if err := ldr.Set(key, value); err != nil {
				return fmt.Errorf("failed to set config: %w", err)
			}

			cmd.Println("Applied changes.")
			return nil
		},
	}
	return cmd
}

func newConfigEditCmd(dmsCli *cli.DmsCLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit configuration",
		Long: `Open configuration file with the default text editor.

The command reads the $EDITOR environment variable and fails if it is unset.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			editor := dmsCli.Env().Getenv("EDITOR")
			if editor == "" {
				return fmt.Errorf("$EDITOR not set")
			}

			ldr := dmsCli.ConfigLoader()
			_ = ensureConfigFile(dmsCli.FS(), ldr)

			cmd.Printf("Text editor: %s\n", editor)
			cmd.Printf("Config path: %s\n", ldr.ConfigFile())

			proc := exec.Command(editor, ldr.ConfigFile())
			proc.Stdout = cmd.OutOrStdout()
			proc.Stdin = cmd.InOrStdin()
			proc.Stderr = cmd.OutOrStderr()

			return proc.Run()
		},
	}
	return cmd
}

// Helpers
func ensureConfigFile(fs afero.Fs, ldr *config.Loader) error {
	if path := ldr.ConfigFile(); path != "" {
		if ok, err := afero.Exists(fs, path); err == nil && ok {
			return nil
		}
	}
	return ldr.Write()
}

func parseLiteral(s string) interface{} {
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return int(i)
	}
	if b, err := strconv.ParseBool(s); err == nil {
		return b
	}
	return s
}
