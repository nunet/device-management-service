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
	"os"
	"os/exec"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/internal/config"
)

func newConfigCmd(fs afero.Fs) *cobra.Command {
	if fs == nil {
		cobra.CheckErr("Fs is nil")
	}
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration file",
		Long: `Utility to manage user's configuration file via command-line

Search for the configuration file is done in the following locations and order:

1. "." (current directory)
2. "$HOME/.nunet"
3. "/etc/nunet"`,
	}
	cmd.AddCommand(newConfigGetCmd(fs))
	cmd.AddCommand(newConfigSetCmd(fs))
	cmd.AddCommand(newConfigEditCmd(fs))
	return cmd
}

func newConfigGetCmd(fs afero.Fs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Display configuration",
		Long: `Display the value for a configuration key

It reads the value from configuration file, otherwise it return default values

Example:
  nunet config get rest.port`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := config.CreateConfigFileIfNotExists(fs)
			if err != nil {
				return fmt.Errorf("failed to create config file: %w", err)
			}
			cmd.Println("Found config file at:", config.GetPath())

			if len(args) == 0 {
				info, err := json.MarshalIndent(config.GetConfig(), "", "    ")
				if err != nil {
					return fmt.Errorf("failed to indent config JSON: %w", err)
				}
				cmd.Println(string(info))
				return nil
			}
			value, err := config.Get(args[0])
			if err != nil {
				return fmt.Errorf("could not get key's value: %w", err)
			}
			pretty, err := json.MarshalIndent(value, "", "    ")
			if err != nil {
				return fmt.Errorf("failed to indent JSON: %w", err)
			}
			cmd.Println(string(pretty))
			return nil
		},
	}
	return cmd
}

func newConfigSetCmd(fs afero.Fs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Update configuration",
		Long: `Set value for a configuration key

It creates a configuration file if does not exists, otherwise it updates the existing file

Example:
  nunet config set rest.port 4444`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			exists, err := config.FileExists(fs)
			if err != nil {
				return fmt.Errorf("failed to check if config file exists: %w", err)
			}
			if !exists {
				cmd.Println("Config file did not exist. Creating new file...")
			} else {
				cmd.Println("Updating existing config file...")
			}

			if err := config.Set(fs, args[0], args[1]); err != nil {
				return fmt.Errorf("failed to set config: %w", err)
			}

			cmd.Println("Applied changes.")

			return nil
		},
	}
	return cmd
}

func newConfigEditCmd(fs afero.Fs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit configuration",
		Long: `Open configuration file with default text editor

This command search the configuration file and open it with the default text editor
It reads the $EDITOR environment variable and it fails if it's not set`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			editor := os.Getenv("EDITOR")
			if editor == "" {
				return fmt.Errorf("$EDITOR not set")
			}

			err := config.CreateConfigFileIfNotExists(fs)
			if err != nil {
				return fmt.Errorf("failed to create config file: %w", err)
			}
			cmd.Printf("Text editor: %s\n", editor)
			cmd.Printf("Config path: %s\n", config.GetPath())

			// do we need better sanitization?
			// do we check if editor is valid?
			proc := exec.Command(editor, config.GetPath())
			proc.Stdout = cmd.OutOrStdout()
			proc.Stdin = cmd.InOrStdin()
			proc.Stderr = cmd.OutOrStderr()

			return proc.Run()
		},
	}
	return cmd
}
