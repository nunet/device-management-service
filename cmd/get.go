package cmd

import (
	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/cmd/actor"
	"gitlab.com/nunet/device-management-service/cmd/cli"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
)

func newGetCmd(dmsCli *cli.DmsCLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get deployments, allocations etc.",
	}
	cmd.AddCommand(newGetDeployments(dmsCli))
	cmd.AddCommand(newGetAllocations(dmsCli))
	return cmd
}

func newGetDeployments(dmsCli *cli.DmsCLI) *cobra.Command {
	behavior := behaviors.DeploymentListBehavior
	cmd, err := actor.NewActorCmdWrapper(dmsCli, behavior)
	if err != nil {
		return &cobra.Command{
			Use: "deployments",
			RunE: func(_ *cobra.Command, _ []string) error {
				return err
			},
		}
	}
	cmd.Use = "deployments"
	cmd.Short = "Get all deployments"
	cmd.Long = `Get all deployments. It will show running deployments as well as completed or stopped ones.
Each deployment will be referenced by its ensemble ID along with their status.`
	return cmd
}

func newGetAllocations(dmsCli *cli.DmsCLI) *cobra.Command {
	behavior := behaviors.AllocationsListBehavior
	cmd, err := actor.NewActorCmdWrapper(dmsCli, behavior)
	if err != nil {
		return &cobra.Command{
			Use: "allocations",
			RunE: func(_ *cobra.Command, _ []string) error {
				return err
			},
		}
	}
	cmd.Use = "allocations"
	cmd.Long = `Get all allocations. It will show running allocations as well as completed or stopped ones.

This returns all allocations running on the host, from one acting as Compute Provider. This will not show not the allocations from deployed ensembles.`
	return cmd
}
