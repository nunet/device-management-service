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
	"gitlab.com/nunet/device-management-service/utils"
)

func newRootCmd(client *utils.HTTPClient, afs afero.Afero) *cobra.Command {
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
	cmd.AddCommand(newRunCmd())
	cmd.AddCommand(newKeyCmd(afs))
	cmd.AddCommand(cap.NewCapCmd(afs))
	cmd.AddCommand(actor.NewActorCmd(client, afs))
	cmd.AddCommand(newConfigCmd(afs.Fs))
	cmd.AddCommand(newAutoCompleteCmd())
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newTapCommand())
	cmd.AddCommand(newGPUCommand())
	return cmd
}
