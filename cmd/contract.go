package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/cmd/actor"
	"gitlab.com/nunet/device-management-service/cmd/cli"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
)

func newContractCmd(dmsCli *cli.DmsCLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contracts",
		Short: "Interact with contracts",
	}

	cmd.AddCommand(newContractListCmd(dmsCli))
	return cmd
}

func newContractListCmd(dmsCli *cli.DmsCLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List contracts",
	}

	cmd.AddCommand(newContractListAlias(dmsCli, "incoming", "List contracts where this node is the provider", contracts.ContractRoleProvider))
	cmd.AddCommand(newContractListAlias(dmsCli, "outgoing", "List contracts where this node is the requestor", contracts.ContractRoleRequestor))
	return cmd
}

func newContractListAlias(dmsCli *cli.DmsCLI, use, short string, role contracts.ContractListIncomingRole) *cobra.Command {
	cmd, err := actor.NewActorCmdWrapper(dmsCli, behaviors.ContractListBehavior)
	if err != nil {
		return &cobra.Command{
			Use: use,
			RunE: func(_ *cobra.Command, _ []string) error {
				return err
			},
		}
	}

	cmd.Use = fmt.Sprintf("%s [flags]", use)
	cmd.Short = short
	cmd.Long = `This command lists contracts for the given role.
		
	Examples:

		nunet contract list incoming
		nunet contract list outgoing`

	prevPreRun := cmd.PreRunE
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if err := cmd.Flags().Set("role", string(role)); err != nil {
			return fmt.Errorf("failed to set role flag: %w", err)
		}
		if prevPreRun != nil {
			return prevPreRun(cmd, args)
		}
		return nil
	}

	return cmd
}
