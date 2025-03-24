// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cmd

import (
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/cmd/actor"
	"gitlab.com/nunet/device-management-service/cmd/cap"
	"gitlab.com/nunet/device-management-service/internal/config"
)

// NewRootCMD returns the cmds
func NewRootCMD(afs afero.Afero, cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nunet",
		Short: "NuNet Device Management Service",
		Long:  `The Device Management Service (DMS) Command Line Interface (CLI)`,
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

	cmd.AddCommand(newRunCmd(cfg))
	cmd.AddCommand(newKeyCmd(afs, cfg))
	cmd.AddCommand(cap.NewCapCmd(afs, cfg))
	cmd.AddCommand(actor.NewActorCmd(afs, cfg))
	cmd.AddCommand(newConfigCmd(afs.Fs, cfg))
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newTapCommand())

	cmd.AddCommand(newGPUCommand())
	return cmd
}

func Execute(cfg *config.Config) {
	afs := afero.Afero{Fs: afero.NewOsFs()}
	cobra.CheckErr(NewRootCMD(afs, cfg).Execute())
}
