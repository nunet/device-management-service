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
	"gitlab.com/nunet/device-management-service/cmd/cli"
	"gitlab.com/nunet/device-management-service/lib/env"
)

// NewRootCMD returns the cmds
func NewRootCMD(dmsCli *cli.DmsCLI) *cobra.Command {
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
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			_, _ = dmsCli.ConfigLoader().Load()
		},
	}

	dmsCli.ConfigLoader().BindFlags(cmd.PersistentFlags())

	cmd.AddCommand(newRunCmd(dmsCli))
	cmd.AddCommand(newKeyCmd(dmsCli))
	cmd.AddCommand(cap.NewCapCmd(dmsCli))
	cmd.AddCommand(actor.NewActorCmd(dmsCli))
	cmd.AddCommand(newConfigCmd(dmsCli))
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newGPUCommand())
	cmd.AddCommand(newNetworkCommand(dmsCli))

	return cmd
}

func Execute() {
	dmsCli := cli.New(
		cli.WithFS(afero.NewOsFs()),
		cli.WithEnv(env.NewOSEnvironment()),
	)

	cobra.CheckErr(NewRootCMD(dmsCli).Execute())
}
