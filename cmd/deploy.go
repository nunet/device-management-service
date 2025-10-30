// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cmd

import (
	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/cmd/actor"
	"gitlab.com/nunet/device-management-service/cmd/cli"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
)

func newDeployCmd(dmsCli *cli.DmsCLI) *cobra.Command {
	behavior := behaviors.NewDeploymentBehavior
	cmd, err := actor.NewActorCmdWrapper(dmsCli, behavior)
	if err != nil {
		return &cobra.Command{
			Use: "deploy",
			RunE: func(_ *cobra.Command, _ []string) error {
				return err
			},
		}
	}
	cmd.Use = "deploy"
	cmd.Short = "Create a deployment"
	cmd.Long = `This command creates a new deployment. It receives an ensemble file as argument.

Example:

Deploy ensemble with a 5 minute timeout
  nunet -c alice deploy -f foo.yaml -t 5m`
	return cmd
}
