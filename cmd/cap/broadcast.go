// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cap

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/cmd/cli"
	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/dms/node"
)

// BroadcastOptions holds the command-line options for the broadcast command.
type BroadcastOptions struct {
	Context string
}

func newBroadcastCmd(dmsCLI *cli.DmsCLI) *cobra.Command {
	var opts BroadcastOptions

	cmd := &cobra.Command{
		Use:   "broadcast",
		Short: "Broadcast revocation tokens to all peers",
		Long: `Broadcast all revocation tokens from a capability context to all peers in the network.

This command retrieves all revocation tokens from the specified capability context
and broadcasts them to all connected peers via the /nunet/revocation topic.
Each peer will receive the revocation tokens and update their local capability contexts.

The --context flag specifies which capability context to broadcast revocations from.
If not specified, the DMS context is used by default.

Usage examples:
  nunet cap broadcast --context dms
  nunet cap broadcast --context org`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runBroadcastCmd(cmd.Context(), dmsCLI, opts)
		},
	}

	useFlagContext(cmd, &opts.Context)

	return cmd
}

func runBroadcastCmd(ctx context.Context, dmsCLI *cli.DmsCLI, opts BroadcastOptions) error {
	// Create security context for the specified capability context
	sctx, err := utils.NewSecurityContext(dmsCLI, opts.Context)
	if err != nil {
		return fmt.Errorf("failed to create security context: %w", err)
	}

	// Get DMS client with the security context
	client, err := dmsCLI.NewClient(sctx)
	if err != nil {
		return fmt.Errorf("failed to create DMS client: %w", err)
	}

	// Prepare the broadcast request
	req := node.CapBroadcastRequest{
		Context: opts.Context,
	}

	// Invoke the /dms/cap/broadcast behavior
	respEnvelope, err := client.InvokeBehavior(ctx, behaviors.BroadcastRevokeCapBehavior, req)
	if err != nil {
		return fmt.Errorf("failed to broadcast revocation tokens: %w", err)
	}

	var resp node.CapBroadcastResponse
	if err := json.Unmarshal(respEnvelope.Message, &resp); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !resp.OK {
		return fmt.Errorf("broadcast failed: %s", resp.Error)
	}

	fmt.Printf("Successfully broadcast %d revocation token(s) to all peers\n", resp.TokensCount)
	return nil
}
