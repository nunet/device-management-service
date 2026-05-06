// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package actor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/client"
	"gitlab.com/nunet/device-management-service/cmd/cli"
	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
	"gitlab.com/nunet/device-management-service/utils/convert"
)

type BehaviorAction string

const (
	bBroadcast BehaviorAction = "broadcast"
	bInvoke    BehaviorAction = "invoke"
	bSend      BehaviorAction = "send"
)

var ErrInvalidArgument = errors.New("invalid argument")

type Command = cobra.Command

type behaviorConfig struct {
	Payload     func() any
	Behavior    string
	Action      BehaviorAction
	SetFlags    func(cmd *Command, payload any)
	RunFn       func(ctx context.Context, dmsCli *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error)
	PreRunFn    func(cmd *Command, dmsCli *cli.DmsCLI, opts actorCmdOptions) error
	ValidArgsFn func(cmd *Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)
	Args        cobra.PositionalArgs
	Long        string
	Short       string
}

func (b *behaviorConfig) Run(ctx context.Context, dmsCli *cli.DmsCLI, opts actorCmdOptions, streams cli.Streams) error {
	// Create security context first
	sctx, err := utils.NewSecurityContext(dmsCli, opts.Context)
	if err != nil {
		return fmt.Errorf("could not create security context: %w", err)
	}

	// Check if timeout was set via -t flag
	var timeout time.Duration
	for _, opt := range opts.MsgOpts {
		// Apply the option to a temporary MessageOptions to check if it sets timeout
		tempOpts := &client.MessageOptions{}
		opt(tempOpts)
		if tempOpts.Timeout > 0 {
			timeout = tempOpts.Timeout
			break
		}
	}

	var dmsClient client.DmsClient
	if timeout > 0 {
		// Create client with timeout from -t flag
		dmsClient, err = dmsCli.NewClientWithTimeout(sctx, timeout)
		if err != nil {
			return fmt.Errorf("could not create client with timeout: %w", err)
		}
	} else {
		// Create client with default timeout
		dmsClient, err = dmsCli.NewClient(sctx)
		if err != nil {
			return fmt.Errorf("could not create client: %w", err)
		}
	}

	res, err := b.RunFn(ctx, dmsCli, dmsClient, opts)
	if err != nil {
		return fmt.Errorf("could not run behavior: %w", err)
	}
	return displayResponse(streams.Out, res)
}

var registeredBehaviors = map[string]*behaviorConfig{
	// /dms/tokenomics/contract/settle
	behaviors.ContractSettleBehavior: {
		Payload: func() any { return &ContractSettleCmd{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*ContractSettleCmd)
			cmd.Flags().StringVarP(&p.ContractDID, "contract-did", "", "", "contract did (required)")
			cmd.Flags().StringVarP(&p.ContractHost, "contract-host-did", "", "", "contract host did (required)")
			_ = cmd.MarkFlagRequired("contract-did")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*ContractSettleCmd)
			if !ok {
				return nil, fmt.Errorf("failed to decode contract settle payload")
			}

			request := contracts.ContractSettleRequest{
				ContractDID: req.ContractDID,
			}

			if req.ContractHost != "" {
				destination, err := getDestinationHandle(req.ContractDID, req.ContractHost)
				if err != nil {
					return nil, err
				}

				opts.MsgOpts = append(opts.MsgOpts, client.WithDestination(destination))
			}

			resp, err := dmsClient.SettleContract(ctx, request, opts.MsgOpts...)
			if err != nil {
				return resp, err
			}
			return resp, nil
		},
		Action: bInvoke,
		Short:  "Send a settle request",
		Long: `Invoke the /dms/tokenomics/contract/settle behavior on an actor
									
This behavior calls the contract settle behaviour.

Examples:

	nunet actor cmd --context user /dms/tokenomics/contract/settle --contract-did <did> --contract-host-did <hostdid>`,
	},
	// /dms/tokenomics/contract/terminate
	behaviors.ContractTerminationBehavior: {
		Payload: func() any { return &ContractTerminateCmd{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*ContractTerminateCmd)
			cmd.Flags().StringVarP(&p.ContractDID, "contract-did", "", "", "contract did (required)")
			cmd.Flags().StringVarP(&p.ContractHost, "contract-host-did", "", "", "contract host did (required)")
			_ = cmd.MarkFlagRequired("contract-did")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*ContractTerminateCmd)
			if !ok {
				return nil, fmt.Errorf("failed to decode contract terminate payload")
			}

			request := contracts.ContractTerminationRequest{
				ContractDID: req.ContractDID,
			}

			if req.ContractHost != "" {
				destination, err := getDestinationHandle(req.ContractDID, req.ContractHost)
				if err != nil {
					return nil, err
				}

				opts.MsgOpts = append(opts.MsgOpts, client.WithDestination(destination))
			}

			resp, err := dmsClient.TerminateContract(ctx, request, opts.MsgOpts...)
			if err != nil {
				return resp, err
			}
			return resp, nil
		},
		Action: bInvoke,
		Short:  "Send a termination request",
		Long: `Invoke the /dms/tokenomics/contract/terminate behavior on an actor
								
This behavior calls the contract terminate behaviour.

Examples:

	nunet actor cmd --context user /dms/tokenomics/contract/terminate --contract-did <did> --contract-host-did <hostdid>`,
	},
	// /dms/tokenomics/contract/complete
	behaviors.ContractCompleteBehavior: {
		Payload: func() any { return &ContractCompleteCmd{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*ContractCompleteCmd)
			cmd.Flags().StringVarP(&p.ContractDID, "contract-did", "", "", "contract did (required)")
			cmd.Flags().StringVarP(&p.ContractHost, "contract-host-did", "", "", "contract host did (required)")
			_ = cmd.MarkFlagRequired("contract-did")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*ContractCompleteCmd)
			if !ok {
				return nil, fmt.Errorf("failed to decode contract complete payload")
			}

			request := contracts.ContractCompletionRequest{
				ContractDID: req.ContractDID,
			}

			if req.ContractHost != "" {
				destination, err := getDestinationHandle(req.ContractDID, req.ContractHost)
				if err != nil {
					return nil, err
				}

				opts.MsgOpts = append(opts.MsgOpts, client.WithDestination(destination))
			}

			resp, err := dmsClient.CompleteContract(ctx, request, opts.MsgOpts...)
			if err != nil {
				return resp, err
			}
			return resp, nil
		},
		Action: bInvoke,
		Short:  "Send a contract complete request",
		Long: `Invoke the /dms/tokenomics/contract/complete behavior on an actor
								
This behavior calls the contract complete behaviour.

Examples:

	nunet actor cmd --context user /dms/tokenomics/contract/complete --contract-did <did> --contract-host-did <hostdid>`,
	},
	// /dms/tokenomics/contract/validate
	behaviors.ContractValidationBehavior: {
		Payload: func() any { return &ContractValidateCmd{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*ContractValidateCmd)
			cmd.Flags().StringVarP(&p.ContractDID, "contract-did", "", "", "contract did (required)")
			cmd.Flags().StringVarP(&p.ContractHost, "contract-host-did", "", "", "contract host did (required)")
			_ = cmd.MarkFlagRequired("contract-did")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*ContractValidateCmd)
			if !ok {
				return nil, fmt.Errorf("failed to decode contract complete payload")
			}

			request := contracts.ContractValidateRequest{
				ContractDID: req.ContractDID,
			}

			if req.ContractHost != "" {
				destination, err := getDestinationHandle(req.ContractDID, req.ContractHost)
				if err != nil {
					return nil, err
				}

				opts.MsgOpts = append(opts.MsgOpts, client.WithDestination(destination))
			}

			resp, err := dmsClient.ValidateContract(ctx, request, opts.MsgOpts...)
			if err != nil {
				return resp, err
			}
			return resp, nil
		},
		Action: bInvoke,
		Short:  "Send a contract validate request",
		Long: `Invoke the /dms/tokenomics/contract/validate behavior on an actor
								
This behavior calls the contract validate behaviour.

Examples:

	nunet actor cmd --context user /dms/tokenomics/contract/validate --contract-did <did> --contract-host-did <hostdid>`,
	},
	// /dms/tokenomics/contract/state
	behaviors.ContractStatusBehavior: {
		Payload: func() any { return &ContractStatusRequestCmd{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*ContractStatusRequestCmd)
			cmd.Flags().StringVarP(&p.ContractDID, "contract-did", "", "", "contract-did (required)")
			cmd.Flags().StringVarP(&p.ContractHost, "contract-host-did", "", "", "contract host did (required)")
			_ = cmd.MarkFlagRequired("contract-did")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*ContractStatusRequestCmd)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}

			contractReq := contracts.ContractStatusRequest{
				ContractDID: req.ContractDID,
			}

			if req.ContractHost != "" {
				destination, err := getDestinationHandle(req.ContractDID, req.ContractHost)
				if err != nil {
					return nil, err
				}

				opts.MsgOpts = append(opts.MsgOpts, client.WithDestination(destination))
			}

			resp, err := dmsClient.ContractStatus(ctx, contractReq, opts.MsgOpts...)
			if err != nil {
				return resp, err
			}

			return resp, nil
		},
		Action: bInvoke,
		Short:  "Send a contract state request",
		Long: `Invoke the /dms/tokenomics/contract/state behavior on an actor
		
This behavior calls the actors contract state behaviour.

Examples:

	nunet actor cmd --context user /dms/tokenomics/contract/state --contract-did <did> --contract-host-did <hostdid>`,
	},
	// /dms/tokenomics/contract/payment/status
	behaviors.ContractPaymentStatusBehavior: {
		Payload: func() any { return &ContractPaymentStatusCmd{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*ContractPaymentStatusCmd)
			cmd.Flags().StringVarP(&p.UniqueID, "unique-id", "", "", "unique id (required)")
			_ = cmd.MarkFlagRequired("unique-id")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*ContractPaymentStatusCmd)
			if !ok {
				return nil, fmt.Errorf("failed to decode payment status payload")
			}

			request := contracts.ContractPaymentStatusRequest{
				UniqueID: req.UniqueID,
			}

			resp, err := dmsClient.GetPaymentStatus(ctx, request, opts.MsgOpts...)
			if err != nil {
				return resp, err
			}
			return resp, nil
		},
		Action: bInvoke,
		Short:  "Send a payment status request",
		Long: `Invoke the /dms/tokenomics/contract/payment/status behavior on an actor
							
This behavior calls the payment status behaviour.

Examples:

	nunet actor cmd --context user /dms/tokenomics/contract/payment/status --unique-id <uniqueid>`,
	},
	// /dms/tokenomics/contract/usages/calculate
	behaviors.ContractUsagesCalculateBehavior: {
		Payload: func() any { return &CollectUsagesAndForwardToPaymentProvidersCmd{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*CollectUsagesAndForwardToPaymentProvidersCmd)
			cmd.Flags().StringVar(&p.ContractDID, "contract-did", "", "Contract DID to process (optional, processes all contracts if not specified)")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req := contracts.CollectUsagesAndForwardToPaymentProvidersRequest{
				ContractDID: opts.Payload.(*CollectUsagesAndForwardToPaymentProvidersCmd).ContractDID,
			}
			resp, err := dmsClient.CollectUsagesAndForwardToPaymentProviders(ctx, req, opts.MsgOpts...)
			if err != nil {
				return resp, err
			}

			return resp, nil
		},
		Action: bInvoke,
		Short:  "Send a usage calculation request",
		Long: `Invoke the /dms/tokenomics/contract/usages/calculate behavior on an actor
	
This behavior calls the actors contract calculate usages behaviour.

Examples:

	nunet actor cmd --context user /dms/tokenomics/contract/usages/calculate
	nunet actor cmd --context user /dms/tokenomics/contract/usages/calculate --contract-did did:key:...`,
	},
	// /dms/tokenomics/contract/transactions/confirm
	behaviors.ContractConfirmLocalTransactionBehavior: {
		Payload: func() any { return &ContractConfirmLocalTransactionCmd{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*ContractConfirmLocalTransactionCmd)
			cmd.Flags().StringVarP(&p.UniqueID, "unique-id", "", "", "transaction unique id (required)")
			cmd.Flags().StringVarP(&p.TxHash, "tx-hash", "", "", "transaction hash (required)")
			cmd.Flags().StringVarP(&p.Blockchain, "blockchain", "", "", "which blockchain was used (required)")
			cmd.Flags().StringVarP(&p.QuoteID, "quote-id", "", "", "payment quote id (optional)")
			_ = cmd.MarkFlagRequired("unique-id")
			_ = cmd.MarkFlagRequired("tx-hash")
			_ = cmd.MarkFlagRequired("blockchain")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*ContractConfirmLocalTransactionCmd)
			if !ok {
				return nil, fmt.Errorf("failed to decode ContractConfirmLocalTransactionCmd payload")
			}

			request := contracts.ContractConfirmLocalTransactionRequest{
				UniqueID:   req.UniqueID,
				TxHash:     req.TxHash,
				Blockchain: req.Blockchain,
				QuoteID:    req.QuoteID,
			}

			resp, err := dmsClient.ConfirmTransaction(ctx, request, opts.MsgOpts...)
			if err != nil {
				return resp, err
			}
			return resp, nil
		},
		Action: bInvoke,
		Short:  "Send a confirm transactions request",
		Long: `Invoke the /dms/tokenomics/contract/transactions/confirm behavior on an actor
							
This behavior calls the actors contract confirm transactions behaviour.

Examples:

	nunet actor cmd --context user /dms/tokenomics/contract/transactions/confirm --unique-id <uniqueid> --tx-hash <txhash> --blockchain ETHEREUM`,
	},
	// /dms/tokenomics/contract/transactions/list
	behaviors.ContractListLocalTransactionsBehavior: {
		Payload: func() any { return &ContractListLocalTransactionsCmd{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*ContractListLocalTransactionsCmd)
			cmd.Flags().StringToStringVarP(&p.Metadata, "filter", "f", nil, "metadata filter to match transactions (optional)")
			cmd.Flags().StringSliceVar(&p.Status, "status", nil, "filter by transaction status (can specify multiple)")
			cmd.Flags().StringVar(&p.ContractDID, "contract-did", "", "filter by contract DID")
			cmd.Flags().StringVar(&p.PaymentValidatorDID, "payment-validator-did", "", "filter by payment validator DID")
			cmd.Flags().StringVar(&p.UniqueID, "unique-id", "", "filter by transaction unique ID")
			cmd.Flags().StringVar(&p.TxHash, "tx-hash", "", "filter by transaction hash")
			cmd.Flags().StringVar(&p.Blockchain, "blockchain", "", "filter by blockchain used")
			cmd.Flags().StringVar(&p.ToAddress, "to-address", "", "filter by receiver address")
			cmd.Flags().StringVar(&p.FromAddress, "from-address", "", "filter by sender address")
			cmd.Flags().IntVar(&p.Limit, "limit", 0, "maximum number of results to return (0 = no limit)")
			cmd.Flags().IntVar(&p.Offset, "offset", 0, "number of results to skip")
			cmd.Flags().StringVar(&p.SortBy, "sort", "", "sort field and direction (only status/-status and created_at/-created_at supported)")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			payload, ok := opts.Payload.(*ContractListLocalTransactionsCmd)
			if !ok {
				return nil, fmt.Errorf("failed to decode ContractListLocalTransactionsCmd payload")
			}

			req := contracts.ContractListLocalTransactionsRequest{
				Metadata:            payload.Metadata,
				Status:              payload.Status,
				ContractDID:         payload.ContractDID,
				PaymentValidatorDID: payload.PaymentValidatorDID,
				UniqueID:            payload.UniqueID,
				Blockchain:          payload.Blockchain,
				FromAddress:         payload.FromAddress,
				ToAddress:           payload.ToAddress,
				TxHash:              payload.TxHash,
				Limit:               payload.Limit,
				Offset:              payload.Offset,
				SortBy:              payload.SortBy,
			}

			resp, err := dmsClient.ListTransactions(ctx, req, opts.MsgOpts...)
			if err != nil {
				return resp, err
			}
			return resp, nil
		},
		Action: bInvoke,
		Short:  "Send a list transactions request",
		Long: `Invoke the /dms/tokenomics/contract/transactions/list behavior on an actor
						
This behavior calls the actors contract list transactions behaviour.

Examples:

	nunet actor cmd --context user /dms/tokenomics/contract/transactions/list
	nunet actor cmd --context user /dms/tokenomics/contract/transactions/list --status unpaid --limit 50 --offset 100
	nunet actor cmd --context user /dms/tokenomics/contract/transactions/list --contract-did did:key:... --sort -unique_id`,
	},
	// /dms/tokenomics/contract/payment/quote/get
	behaviors.ContractGetPaymentQuoteBehavior: {
		Payload: func() any { return &ContractGetPaymentQuoteCmd{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*ContractGetPaymentQuoteCmd)
			cmd.Flags().StringVarP(&p.UniqueID, "unique-id", "", "", "transaction unique id (required)")
			_ = cmd.MarkFlagRequired("unique-id")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*ContractGetPaymentQuoteCmd)
			if !ok {
				return nil, fmt.Errorf("failed to decode ContractGetPaymentQuoteCmd payload")
			}

			request := contracts.ContractGetPaymentQuoteRequest{
				UniqueID: req.UniqueID,
			}

			resp, err := dmsClient.InvokeBehavior(ctx, behaviors.ContractGetPaymentQuoteBehavior, request, opts.MsgOpts...)
			if err != nil {
				return nil, fmt.Errorf("failed to invoke behavior: %w", err)
			}

			var quoteResp contracts.ContractGetPaymentQuoteResponse
			if err := json.Unmarshal(resp.Message, &quoteResp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal response: %w", err)
			}

			return quoteResp, nil
		},
		Action: bInvoke,
		Short:  "Get a payment quote for a transaction",
		Long: `Invoke the /dms/tokenomics/contract/payment/quote/get behavior on an actor
						
This behavior gets a real-time payment quote for a transaction requiring currency conversion.

Examples:

	nunet actor cmd --context user /dms/tokenomics/contract/payment/quote/get --unique-id <unique_id>`,
	},
	// /dms/tokenomics/contract/payment/quote/validate
	behaviors.ContractValidatePaymentQuoteBehavior: {
		Payload: func() any { return &ContractValidatePaymentQuoteCmd{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*ContractValidatePaymentQuoteCmd)
			cmd.Flags().StringVarP(&p.QuoteID, "quote-id", "", "", "quote id (required)")
			_ = cmd.MarkFlagRequired("quote-id")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*ContractValidatePaymentQuoteCmd)
			if !ok {
				return nil, fmt.Errorf("failed to decode ContractValidatePaymentQuoteCmd payload")
			}

			request := contracts.ContractValidatePaymentQuoteRequest{
				QuoteID: req.QuoteID,
			}

			resp, err := dmsClient.InvokeBehavior(ctx, behaviors.ContractValidatePaymentQuoteBehavior, request, opts.MsgOpts...)
			if err != nil {
				return nil, fmt.Errorf("failed to invoke behavior: %w", err)
			}

			var validateResp contracts.ContractValidatePaymentQuoteResponse
			if err := json.Unmarshal(resp.Message, &validateResp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal response: %w", err)
			}

			return validateResp, nil
		},
		Action: bInvoke,
		Short:  "Validate a payment quote",
		Long: `Invoke the /dms/tokenomics/contract/payment/quote/validate behavior on an actor
						
This behavior validates a payment quote before payment execution.

Examples:

	nunet actor cmd --context user /dms/tokenomics/contract/payment/quote/validate --quote-id <quote_id>`,
	},
	// /dms/tokenomics/contract/payment/quote/cancel
	behaviors.ContractCancelPaymentQuoteBehavior: {
		Payload: func() any { return &ContractCancelPaymentQuoteCmd{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*ContractCancelPaymentQuoteCmd)
			cmd.Flags().StringVarP(&p.QuoteID, "quote-id", "", "", "quote id (required)")
			_ = cmd.MarkFlagRequired("quote-id")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*ContractCancelPaymentQuoteCmd)
			if !ok {
				return nil, fmt.Errorf("failed to decode ContractCancelPaymentQuoteCmd payload")
			}

			request := contracts.ContractCancelPaymentQuoteRequest{
				QuoteID: req.QuoteID,
			}

			resp, err := dmsClient.InvokeBehavior(ctx, behaviors.ContractCancelPaymentQuoteBehavior, request, opts.MsgOpts...)
			if err != nil {
				return nil, fmt.Errorf("failed to invoke behavior: %w", err)
			}

			var cancelResp contracts.ContractCancelPaymentQuoteResponse
			if err := json.Unmarshal(resp.Message, &cancelResp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal response: %w", err)
			}

			return cancelResp, nil
		},
		Action: bInvoke,
		Short:  "Cancel a payment quote",
		Long: `Invoke the /dms/tokenomics/contract/payment/quote/cancel behavior on an actor
						
This behavior cancels/invalidates a payment quote.

Examples:

	nunet actor cmd --context user /dms/tokenomics/contract/payment/quote/cancel --quote-id <quote_id>`,
	},
	// /dms/tokenomics/contract/list_incoming
	behaviors.ContractListBehavior: {
		Payload: func() any { return &contracts.ContractListIncomingRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*contracts.ContractListIncomingRequest)
			cmd.Flags().StringVarP((*string)(&p.Role), "role", "", "", "role filter (provider|requestor)")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, _ := opts.Payload.(*contracts.ContractListIncomingRequest)
			if req == nil {
				req = &contracts.ContractListIncomingRequest{}
			}
			resp, err := dmsClient.ListIncoming(ctx, *req, opts.MsgOpts...)
			if err != nil {
				return resp, err
			}

			return resp, nil
		},
		Action: bInvoke,
		Short:  "Send a list incoming contract request",
		Long: `Invoke the /dms/tokenomics/contract/list_incoming behavior on an actor
					
This behavior calls the actors contract list behaviour.

Examples:

	nunet actor cmd --context user /dms/tokenomics/contract/list_incoming`,
	},
	// /dms/tokenomics/contract/aprove_local
	behaviors.ContractApproveLocalBehavior: {
		Payload: func() any { return &ContractApproveLocalRequestCmd{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*ContractApproveLocalRequestCmd)
			cmd.Flags().StringVarP(&p.ContractDID, "contract-did", "", "", "contract-did (required)")
			_ = cmd.MarkFlagRequired("contract-did")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*ContractApproveLocalRequestCmd)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}

			contractReq := contracts.ContractApproveLocalRequest{
				ContractDID: req.ContractDID,
			}

			resp, err := dmsClient.ApproveLocal(ctx, contractReq, opts.MsgOpts...)
			if err != nil {
				return resp, err
			}

			return resp, nil
		},
		Action: bInvoke,
		Short:  "Send a contract approval request",
		Long: `Invoke the /dms/tokenomics/contract/aprove_local behavior on an actor
				
This behavior calls the actors contract approval behaviour.

Examples:

	nunet actor cmd --context user /dms/tokenomics/contract/aprove_local --contract-did <did>`,
	},
	// /dms/tokenomics/contract/create
	behaviors.ContractCreateBehavior: {
		Payload: func() any { return &CreateContractRequestCmd{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*CreateContractRequestCmd)
			cmd.Flags().StringVarP(&p.ContractFile, "contract-file", "", "", "contract-file (required)")
			_ = cmd.MarkFlagRequired("contract-file")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*CreateContractRequestCmd)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}

			data, err := os.ReadFile(req.ContractFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read contract file: %w", err)
			}

			var contractReq contracts.CreateContractRequest
			err = json.Unmarshal(data, &contractReq)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal create contract request payload: %w", err)
			}

			resp, err := dmsClient.NewContract(ctx, contractReq, opts.MsgOpts...)
			if err != nil {
				return resp, err
			}

			return resp, nil
		},
		Action: bInvoke,
		Short:  "Send a create contract message",
		Long: `Invoke the /dms/tokenomics/contract/create behavior on an actor
		
This behavior calls the actors create contract behaviour.

Examples:

	nunet actor cmd --context user /dms/tokenomics/contract/create --contract-file <file> --dest <did_of_solution_enabler>`,
	},
	// /dms/volume/create
	behaviors.VolumeCreateBehavior: {
		Payload: func() any { return &CreateVolumeRequestCmd{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*CreateVolumeRequestCmd)
			cmd.Flags().StringVarP(&p.VolumeName, "name", "n", "", "name (required)")
			cmd.Flags().StringVarP(&p.ClientPEMFile, "client-pem-file", "p", "", "client-pem-file (required)")
			cmd.Flags().StringVarP(&p.CAOutputDir, "ca-output-dir", "", "", "ca-output-dir (required)")

			_ = cmd.MarkFlagRequired("name")
			_ = cmd.MarkFlagRequired("client-pem-file")
			_ = cmd.MarkFlagRequired("ca-output-dir")
		},
		RunFn: func(ctx context.Context, dmsCli *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*CreateVolumeRequestCmd)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}

			afs := afero.Afero{Fs: dmsCli.FS()}
			data, err := afs.ReadFile(req.ClientPEMFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}

			// validate client pem

			cfg := &node.CreateVolumeRequest{
				Name:      req.VolumeName,
				ClientPEM: string(data),
			}

			resp, err := dmsClient.CreateVolume(ctx, *cfg, opts.MsgOpts...)
			if err != nil {
				return resp, err
			}

			err = afs.WriteFile(filepath.Join(req.CAOutputDir, "glusterfs.ca"), []byte(resp.CAData), 0o775)
			if err != nil {
				return resp, err
			}

			return resp, nil
		},
		Action: bInvoke,
		Short:  "Send a create volume message",
		Long: `Invoke the /dms/volume/create behavior on an actor
	
This behavior calls the actors create volume behaviour.

Examples:

	nunet actor cmd --context user /dms/volume/create --name <volname> --client-pem-file <filename>`,
	},
	// /dms/volume/delete
	behaviors.VolumeDeleteBehavior: {
		Payload: func() any { return &node.DeleteVolumeRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.DeleteVolumeRequest)

			cmd.Flags().StringVarP(&p.Name, "name", "n", "", "name (required)")

			_ = cmd.MarkFlagRequired("name")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*node.DeleteVolumeRequest)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}

			return dmsClient.DeleteVolume(ctx, *req, opts.MsgOpts...)
		},
		Action: bInvoke,
		Short:  "Send a delete volume message",
		Long: `Invoke the /dms/volume/delete behavior on an actor
		
This behavior calls the actors delete volume behaviour.

Examples:

	nunet actor cmd --context user /dms/volume/delete --name <volname>`,
	},
	// /dms/volume/start
	behaviors.VolumeStartBehavior: {
		Payload: func() any { return &node.StartVolumeRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.StartVolumeRequest)

			cmd.Flags().StringVarP(&p.Name, "name", "n", "", "name (required)")

			_ = cmd.MarkFlagRequired("name")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*node.StartVolumeRequest)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}

			return dmsClient.StartVolume(ctx, *req, opts.MsgOpts...)
		},
		Action: bInvoke,
		Short:  "Send a start volume message",
		Long: `Invoke the /dms/volume/start behavior on an actor
		
		This behavior calls the actors start volume behaviour.
		
		Examples:
		
		  nunet actor cmd --context user /dms/volume/start --name <volname>`,
	},
	// /public/hello
	behaviors.PublicHelloBehavior: {
		Action: bInvoke,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.Hello(ctx, opts.MsgOpts...)
		},
		Short: "Invoke a 'hello' message",
		Long: `Invoke the /public/hello behavior on an actor

This behavior invokes a "hello" for a polite introduction.

Examples:

  nunet actor cmd --context user /public/hello
  nunet actor cmd --context user /public/hello --dest <did/peer_id/actor_handle>`,
	},
	// /broadcast/hello
	behaviors.BroadcastHelloBehavior: {
		Action: bBroadcast,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.BroadcastHello(ctx, opts.MsgOpts...)
		},
		Short: "Broadcast a 'hello' message to a topic",
		Long: `Invokes the /broadcast/hello behavior on an actor

This behavior sends a "hello" message to a broadcast topic for polite introduction.

Examples:

  nunet actor cmd --context user /broadcast/hello`,
	},
	// /public/status
	behaviors.PublicStatusBehavior: {
		Action: bInvoke,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.Status(ctx, opts.MsgOpts...)
		},
		Short: "Retrieve actor status",
		Long: `Invokes the /public/status behavior on an actor

This behavior retrieves the status and resources information.

Examples:
  nunet actor cmd --context user /public/status # own actor status
  nunet actor cmd --context user /public/status --dest <did/peer_id/actor_handle> # status of specified destination`,
	},
	// /dms/node/status
	behaviors.StatusDiscoveryBehavior: {
		Action: bInvoke,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.Discovery(ctx, opts.MsgOpts...)
		},
		Short: "Invoke a 'status discovery' message",
		Long: `Invoke the /dms/node/status behavior on an actor
	
	This behavior invokes a "status discovery" behavior for fleet discovery.
	
	Examples:
	
	  nunet actor cmd --context user /dms/node/status
	  nunet actor cmd --context user /dms/node/status --dest <did/peer_id/actor_handle>`,
	},
	// /broadcast/dms/status
	behaviors.BroadcastStatusDiscoveryBehavior: {
		Action: bBroadcast,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.DiscoveryBroadcast(ctx, opts.MsgOpts...)
		},
		Short: "Broadcast a 'status discovery' message to a topic",
		Long: `Broadcast the /broadcast/dms/status behavior to nodes in the network
	
	This behavior broadcasts a "status discovery" message to topic /nunet/status for fleet discovery.
	
	Examples:
	
	  nunet actor cmd --context user /broadcast/dms/status`,
	},
	// /dms/node/peers/list
	behaviors.PeersListBehavior: {
		Action: bInvoke,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.PeersList(ctx, opts.MsgOpts...)
		},
		Short: "List connected peers",
		Long: `Invokes the /dms/node/peers/list behavior on an actor

This behavior retrieves a list of connected peers.

Examples:
  nunet actor cmd --context user /dms/node/peers/list # own node actor peer list
  nunet actor cmd --context user /dms/node/peers/list --dest <did/peer_id/actor_handle> # specified node actor peer list`,
	},
	// /dms/node/peers/self
	behaviors.PeerAddrInfoBehavior: {
		Action: bInvoke,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.PeersSelf(ctx, opts.MsgOpts...)
		},
		Short: "Get peer's ID and addresses",
		Long: `Invokes the /dms/node/peers/self behavior on an actor

This behavior retrieves information about the node itself, such as its ID or addresses.

Examples:
  nunet actor cmd --context user /dms/node/peers/self # own node actor peer ID
  nunet actor cmd --context user /dms/node/peers/self --dest <did/peer_id/actor_handle> # specified node actor peer ID`,
	},
	// /dms/node/peers/ping
	behaviors.PeerPingBehavior: {
		Action:  bInvoke,
		Payload: func() any { return &node.PingRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.PingRequest)

			cmd.Flags().StringVarP(&p.Host, "host", "H", "", "host address to ping (required)")
			_ = cmd.MarkFlagRequired("host")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*node.PingRequest)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}
			return dmsClient.PeerPing(ctx, *req, opts.MsgOpts...)
		},
		Short: "Ping a peer",
		Long: `Invokes the /dms/node/peers/ping behavior on an actor

This behavior establishes a ping connection with a peer.

Examples:
  nunet actor cmd --context user /dms/node/peers/ping --host <peer_id>`,
	},
	// /dms/node/peers/dht
	behaviors.PeerDHTBehavior: {
		Action: bInvoke,
		// TODO: Check the actual implementation?
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.PeersListFromDHT(ctx, opts.MsgOpts...)
		},
		Short: "List peers connected to DHT",
		Long: `Invokes the /dms/node/peers/dht behavior on an actor

This behavior returns a list of peers from the  Distributed Hash Table (DHT) used for peer discovery and content routing.

Examples:
  nunet actor cmd --context user /dms/node/peers/dht`,
	},
	// /dms/node/peers/connect
	behaviors.PeerConnectBehavior: {
		Action:  bInvoke,
		Payload: func() any { return &node.PeerConnectRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.PeerConnectRequest)

			cmd.Flags().StringVarP(&p.Address, "address", "a", "", "peer address to connect to (required)")
			_ = cmd.MarkFlagRequired("address")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*node.PeerConnectRequest)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}
			return dmsClient.PeerConnect(ctx, *req, opts.MsgOpts...)
		},
		Short: "Connect to a peer",
		Long: `Invokes the /dms/node/peers/connect behavior on an actor

This behavior initiates a connection to a specified peer.

Examples:
  nunet actor cmd --context user /dms/node/peers/connect --address /p2p/<peer_id>`,
	},
	// /dms/node/peers/score
	behaviors.PeerScoreBehavior: {
		Action: bInvoke,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.PeerScore(ctx, opts.MsgOpts...)
		},
		Short: "Retrieves gossipsub broadcast score",
		Long: `Invokes the /dms/node/peers/score behavior on an actor

This behavior retrieves a snapshot of the peer's gossipsub broadcast score.

Examples:
  nunet actor cmd --context user /dms/node/peers/score`,
	},
	// /dms/debug/flightrec
	behaviors.DebugFlightrecBehavior: {
		Action: bInvoke,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.Flightrec(ctx, opts.MsgOpts...)
		},
		Short: "Dumps a flight recorder snapshot",
		Long: `Invokes the /dms/debug/flightrec behavior on an actor

This behavior dumps a flight recorder snapshot.

Examples:
  nunet actor cmd --context user /dms/debug/flightrec`,
	},
	// /dms/node/onboarding/onboard
	behaviors.OnboardBehavior: {
		Action:  bInvoke,
		Payload: func() any { return &onboardingInput{} },
		SetFlags: func(cmd *Command, payload any) {
			// infer the type of the payload
			p := payload.(*onboardingInput)
			cmd.Flags().StringVarP(&p.RAMSize, "ram", "R", "0GiB", "set the amount of memory to reserve for NuNet (defaults to GiB)")
			cmd.Flags().Float32VarP(&p.CPUCores, "cpu", "C", 0, "set the number of CPU cores to reserve for NuNet")
			cmd.Flags().StringVarP(&p.DiskSize, "disk", "D", "0GiB", "set the amount of disk size to reserve for NuNet (defaults to GiB)")
			cmd.Flags().StringVarP(&p.GPUsStr, "gpus", "G", "", "comma-separated list of GPU Index and VRAM in GiB (e.g. 0:4,1:8). The gpu index can be obtained from 'nunet gpu list' command. Unit can be specified for the VRAM but defaults to GiB")
			cmd.Flags().BoolVarP(&p.NoGPU, "no-gpu", "N", false, "do not reserve any GPU resources")
			cmd.MarkFlagsOneRequired("ram", "cpu", "disk")
			cmd.MarkFlagsRequiredTogether("ram", "cpu", "disk")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			p, ok := opts.Payload.(*onboardingInput)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}

			if err := processOnboardInput(ctx, dmsClient, opts); err != nil {
				return nil, err
			}

			req := node.OnboardRequest{}
			req.Config.OnboardedResources.CPU.Cores = p.CPUCores
			req.Config.OnboardedResources.CPU.ClockSpeed = p.CPUCLock
			req.NoGPU = p.NoGPU
			req.Config.OnboardedResources.GPUs = p.GPUs

			var err error
			// convert RAM and Disk from specified unit to bytes if specified otherwise, default to GiB
			req.Config.OnboardedResources.RAM.Size, err = convert.ParseBytesWithDefaultUnit(p.RAMSize, "GiB")
			if err != nil {
				return nil, fmt.Errorf("failed to decode RAM size. Expected Unit in GiB")
			}
			req.Config.OnboardedResources.Disk.Size, err = convert.ParseBytesWithDefaultUnit(p.DiskSize, "GiB")
			if err != nil {
				return nil, fmt.Errorf("failed to decode Disk size. Expected Unit in GiB")
			}

			return dmsClient.Onboard(ctx, req, opts.MsgOpts...)
		},
		Short: "Onboard a node to the network",
		Long: `Invokes the /dms/node/onboarding/onboard behavior on an actor

This behavior is used to onboard a node to the DMS, making its resources available for use.

Examples:
  nunet actor cmd --context user /dms/node/onboarding/onboard --disk 1 --ram 1 --cpu 2`,
	},
	// /dms/node/onboarding/offboard
	behaviors.OffboardBehavior: {
		Action: bInvoke,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req := node.OffboardRequest{}
			return dmsClient.Offboard(ctx, req, opts.MsgOpts...)
		},
		Short: "Offboard a node from the network",
		Long: `Invokes the /dms/node/onboarding/offboard behavior on an actor

This behavior is used to offboard a node from the DMS (Device Management Service).

Examples:
  nunet actor cmd --context user /dms/node/onboarding/offboard
  nunet actor cmd --context user /dms/node/onboarding/offboard --force`,
		// TODO: there is no flag set for --force
	},
	// /dms/node/onboarding/status
	behaviors.OnboardStatusBehavior: {
		Action: bInvoke,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.OnboardStatus(ctx, opts.MsgOpts...)
		},
		Short: "Retrieve onboarding status of a node",
		Long: `Invokes the /dms/node/onboarding/status behavior on an actor

This behavior is used to check the onboarding status of a node.

Examples:
  nunet actor cmd --context user /dms/node/onboarding/status`,
	},

	// /dms/node/deployment/list
	behaviors.DeploymentListBehavior: {
		Action:  bInvoke,
		Payload: func() any { return &DeploymentListCmd{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*DeploymentListCmd)

			// Existing metadata filter
			cmd.Flags().StringToStringVarP(&p.Metadata, "filter", "f", nil, "metadata filter to filter deployments (optional)")

			// Pagination
			cmd.Flags().IntVar(&p.Limit, "limit", 0, "Maximum number of results to return (0 = no limit)")
			cmd.Flags().IntVar(&p.Offset, "offset", 0, "Number of results to skip")

			// Status filter
			cmd.Flags().StringSliceVar(&p.Status, "status", nil, "Filter by deployment status (can specify multiple, e.g., --status Running --status Failed)")

			// Date filters
			cmd.Flags().StringVar(&p.CreatedAfter, "created-after", "", "Filter deployments created after this date (RFC3339 or relative: 1h, 1d, etc.)")
			cmd.Flags().StringVar(&p.CreatedBefore, "created-before", "", "Filter deployments created before this date (RFC3339 or relative)")
			cmd.Flags().StringVar(&p.UpdatedAfter, "updated-after", "", "Filter deployments updated after this date (RFC3339 or relative)")
			cmd.Flags().StringVar(&p.UpdatedBefore, "updated-before", "", "Filter deployments updated before this date (RFC3339 or relative)")

			// Sorting
			cmd.Flags().StringVar(&p.SortBy, "sort", "-created_at", "Sort field and direction (e.g., 'created_at', '-created_at', 'status')")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			payload, ok := opts.Payload.(*DeploymentListCmd)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}

			req := &node.DeploymentListRequest{
				Metadata: payload.Metadata,
				Limit:    payload.Limit,
				Offset:   payload.Offset,
				SortBy:   payload.SortBy,
			}

			// set status
			if len(payload.Status) > 0 {
				req.Status = make([]jobtypes.DeploymentStatus, 0, len(payload.Status))
				for _, statusStr := range payload.Status {
					statusStr = strings.TrimSpace(statusStr)
					for i := jobtypes.DeploymentStatusPreparing; i <= jobtypes.DeploymentStatusCompleted; i++ {
						if strings.EqualFold(i.String(), statusStr) {
							req.Status = append(req.Status, i)
							break
						}
					}
				}
			}

			// Parse date strings from CLI if provided
			if payload.CreatedAfter != "" {
				parsed, err := parseDateString(payload.CreatedAfter)
				if err != nil {
					return nil, fmt.Errorf("invalid created-after date: %w", err)
				}
				req.CreatedAfter = &parsed
			}
			if payload.CreatedBefore != "" {
				parsed, err := parseDateString(payload.CreatedBefore)
				if err != nil {
					return nil, fmt.Errorf("invalid created-before date: %w", err)
				}
				req.CreatedBefore = &parsed
			}
			if payload.UpdatedAfter != "" {
				parsed, err := parseDateString(payload.UpdatedAfter)
				if err != nil {
					return nil, fmt.Errorf("invalid updated-after date: %w", err)
				}
				req.UpdatedAfter = &parsed
			}
			if payload.UpdatedBefore != "" {
				parsed, err := parseDateString(payload.UpdatedBefore)
				if err != nil {
					return nil, fmt.Errorf("invalid updated-before date: %w", err)
				}
				req.UpdatedBefore = &parsed
			}
			return dmsClient.DeploymentList(ctx, *req, opts.MsgOpts...)
		},
		Short: "List deployments",
		Long: `Invokes the /dms/node/deployment/list behavior on an actor

This behavior retrieves a list of deployments on the node with support for pagination, filtering, and sorting.

Examples:
  # List first 10 deployments
  nunet actor cmd --context user /dms/node/deployment/list --limit 10

  # List with pagination
  nunet actor cmd --context user /dms/node/deployment/list --limit 10 --offset 0

  # Filter by status
  nunet actor cmd --context user /dms/node/deployment/list --status Running --status Failed

  # Filter by creation date (relative)
  nunet actor cmd --context user /dms/node/deployment/list --created-after "7d"

  # Filter by creation date (absolute)
  nunet actor cmd --context user /dms/node/deployment/list --created-after "2024-01-01T00:00:00Z"

  # Combine filters with pagination
  nunet actor cmd --context user /dms/node/deployment/list --status Running --created-after "1d" --limit 50 --sort "-created_at"

  # With metadata filter
  nunet actor cmd --context user /dms/node/deployment/list --filter "environment=production" --status Running --limit 10`,
	},

	// /dms/node/deployment/prune
	behaviors.DeploymentPruneBehavior: {
		Action:  bInvoke,
		Payload: func() any { return &node.DeploymentPruneRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.DeploymentPruneRequest)

			cmd.Flags().StringVar(&p.Before, "before", "", "remove deployments created before this time: RFC3339 or duration (e.g. 1m, 1h, 1s, 1d)")
			cmd.Flags().BoolVarP(&p.All, "all", "a", false, "remove all deployments whose status is greater than Running")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*node.DeploymentPruneRequest)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}
			if req.Before == "" && !req.All {
				return nil, fmt.Errorf("must provide --before or --all")
			}
			return dmsClient.DeploymentPrune(ctx, *req, opts.MsgOpts...)
		},
		Short: "Prune old deployments",
		Long: `Invokes the /dms/node/deployment/prune behavior on an actor

This behavior removes deployments before a specified datetime or duration, or deletes all deployments with status greater than Running when --all is used.

Examples:
	  nunet actor cmd --context user /dms/node/deployment/prune --before 2025-01-01T00:00:00Z
	  nunet actor cmd --context user /dms/node/deployment/prune --all`,
	},

	// /dms/node/deployment/status
	behaviors.DeploymentStatusBehavior: {
		Action:  bInvoke,
		Payload: func() any { return &node.DeploymentStatusRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.DeploymentStatusRequest)
			cmd.Flags().StringVarP(&p.ID, "id", "i", "", "deployment ID (required)")
			cmd.Flags().BoolVarP(&p.IncludeUsage, "include-usage", "u", false, "include allocation resource usage statistics")
			_ = cmd.MarkFlagRequired("id")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*node.DeploymentStatusRequest)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}
			return dmsClient.DeploymentStatus(ctx, *req, opts.MsgOpts...)
		},
		Short: "Get deployment status",
		Long: `Invokes the /dms/node/deployment/status behavior on an actor

This behavior retrieves the status of a specific deployment.

Examples:
  nunet actor cmd --context user /dms/node/deployment/status --id <deployment_id>
  nunet actor cmd --context user /dms/node/deployment/status --id <deployment_id> --include-usage`,
	},

	// /dms/node/deployment/logs
	behaviors.DeploymentLogsBehavior: {
		Action:  bInvoke,
		Payload: func() any { return &node.DeploymentLogsRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.DeploymentLogsRequest)
			cmd.Flags().StringVarP(&p.EnsembleID, "id", "i", "", "ensemble ID (required)")
			cmd.Flags().StringVarP(&p.AllocationName, "allocation", "a", "", "allocation name (required)")
			_ = cmd.MarkFlagRequired("id")
			_ = cmd.MarkFlagRequired("allocation")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*node.DeploymentLogsRequest)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}
			return dmsClient.DeploymentLogs(ctx, *req, opts.MsgOpts...)
		},
		Short: "Get deployment logs",
		Long: `Invokes the /dms/node/deployment/logs behavior on an actor

This behavior retrieves the logs of a specific deployment, writing it to a file
with path returned in the response.

Examples:
  nunet actor cmd --context user /dms/node/deployment/logs --id <deployment_id> --allocation <allocation_name>`,
	},

	// /dms/node/deployment/manifest
	behaviors.DeploymentManifestBehavior: {
		Action:  bInvoke,
		Payload: func() any { return &node.DeploymentManifestRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.DeploymentManifestRequest)
			cmd.Flags().StringVarP(&p.ID, "id", "i", "", "deployment ID (required)")
			_ = cmd.MarkFlagRequired("id")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*node.DeploymentManifestRequest)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}
			return dmsClient.DeploymentManifest(ctx, *req, opts.MsgOpts...)
		},
		Short: "Get deployment manifest",
		Long: `Invokes the /dms/node/deployment/manifest behavior on an actor

This behavior retrieves the manifest of a specific deployment.

Examples:
  nunet actor cmd --context user /dms/node/deployment/manifest --id <deployment_id>`,
	},

	// /dms/node/deployment/info
	behaviors.DeploymentInfoBehavior: {
		Action:  bInvoke,
		Payload: func() any { return &node.DeploymentInfoRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.DeploymentInfoRequest)
			cmd.Flags().StringVarP(&p.ID, "id", "i", "", "deployment ID (required)")
			cmd.Flags().BoolVar(&p.IncludeUsage, "usage", false, "include resource usage statistics")
			cmd.Flags().BoolVar(&p.IncludeLogs, "logs", false, "include log file paths for allocations")
			cmd.Flags().StringSliceVar(&p.AllocationNames, "allocations", nil, "specific allocation names to include logs for (empty = all)")
			_ = cmd.MarkFlagRequired("id")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*node.DeploymentInfoRequest)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}
			return dmsClient.DeploymentInfo(ctx, *req, opts.MsgOpts...)
		},
		Short: "Get comprehensive deployment information",
		Long: `Invokes the /dms/node/deployment/info behavior on an actor

This behavior retrieves comprehensive information about a deployment including status,
manifest, allocation details, optional resource usage, and optional log file paths.
Logs are returned as file paths (not content) for optimal performance.

Examples:
  nunet actor cmd --context user /dms/node/deployment/info --id <deployment_id>
  nunet actor cmd --context user /dms/node/deployment/info --id <deployment_id> --usage
  nunet actor cmd --context user /dms/node/deployment/info --id <deployment_id> --logs
  nunet actor cmd --context user /dms/node/deployment/info --id <deployment_id> --usage --logs
  nunet actor cmd --context user /dms/node/deployment/info --id <deployment_id> --logs --allocations alloc1 alloc2`,
	},

	// /dms/node/deployment/shutdown
	behaviors.DeploymentShutdownBehavior: {
		Action:  bInvoke,
		Payload: func() any { return &node.DeploymentShutdownRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.DeploymentShutdownRequest)
			cmd.Flags().StringVarP(&p.ID, "id", "i", "", "deployment ID (required)")
			_ = cmd.MarkFlagRequired("id")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*node.DeploymentShutdownRequest)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}
			return dmsClient.DeploymentShutdown(ctx, *req, opts.MsgOpts...)
		},
		Short: "Shutdown a deployment",
		Long: `Invokes the /dms/node/deployment/shutdown behavior on an actor

This behavior shuts down a specific deployment.

Examples:
  nunet actor cmd --context user /dms/node/deployment/shutdown --id <deployment_id>`,
	},

	behaviors.NewDeploymentBehavior: {
		Action: bInvoke,
		Short:  "Create a new deployment",
		Long: `Invokes the /dms/node/deployment/new behavior on an actor

This behavior creates a new deployment.

Examples:
  nunet actor cmd --context user /dms/node/deployment/new --spec-file <path to ensemble specification file>`,
		Payload: func() any { return &NewDeploymentRequestCmd{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*NewDeploymentRequestCmd)
			cmd.Flags().StringVarP(&p.Config, "spec-file", "f", "ensemble.yaml", "path of the ensemble specification file")
		},
		RunFn: func(ctx context.Context, dmsCli *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			reqCmd, ok := opts.Payload.(*NewDeploymentRequestCmd)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}

			req := &node.NewDeploymentRequest{}

			cfg, err := ProcessEnsembleYaml(afero.Afero{Fs: dmsCli.FS()}, dmsCli.Env(), reqCmd.Config)
			if err != nil {
				return nil, fmt.Errorf("failed to process ensemble config file: %w", err)
			}

			req.Ensemble = *cfg

			return dmsClient.DeploymentNew(ctx, *req, opts.MsgOpts...)
		},
	},

	// /dms/node/deployment/update
	behaviors.DeploymentUpdateBehavior: {
		Action: bInvoke,
		Short:  "Updates an existing deployment",
		Long: `Invokes the /dms/node/deployment/update behavior on an actor

This behavior updates an existing deployment.

Examples:
  nunet actor cmd --context user /dms/node/deployment/update --spec-file <path to ensemble specification file> --id <ensemble_id>`,
		Payload: func() any { return &UpdateDeploymentRequestCmd{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*UpdateDeploymentRequestCmd)
			cmd.Flags().StringVarP(&p.Config, "spec-file", "f", "ensemble.yaml", "path of the ensemble specification file")
			cmd.Flags().StringVarP(&p.EnsembleID, "id", "i", "", "id of the ensemble to update (required)")
			_ = cmd.MarkFlagRequired("id")
		},
		RunFn: func(
			ctx context.Context, dmsCli *cli.DmsCLI,
			dmsClient client.DmsClient, opts actorCmdOptions,
		) (any, error) {
			reqCmd, ok := opts.Payload.(*UpdateDeploymentRequestCmd)
			if !ok {
				return nil, fmt.Errorf("failed to encode payload")
			}

			req := &node.UpdateDeploymentRequest{
				EnsembleID: reqCmd.EnsembleID,
			}

			cfg, err := ProcessEnsembleYaml(afero.Afero{Fs: dmsCli.FS()}, dmsCli.Env(), reqCmd.Config)
			if err != nil {
				return nil, fmt.Errorf("failed to process ensemble config file: %w", err)
			}

			req.Ensemble = *cfg
			return dmsClient.DeploymentUpdate(ctx, *req, opts.MsgOpts...)
		},
	},

	behaviors.ResourcesAllocatedBehavior: {
		Action: bInvoke,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.ResourcesAllocated(ctx, opts.MsgOpts...)
		},
		Short: "Get allocated resources",
		Long: `Invokes the /dms/node/resources/allocated behavior on an actor

This behavior retrieves the resources allocated by the node. The resources include CPU, RAM, GPU and disk space.
The returned units are in Hz for CPU clock speed, bytes for RAM, VRAM and disk space.

Examples:
	  nunet actor cmd --context user /dms/node/resources/allocated`,
	},

	behaviors.ResourcesFreeBehavior: {
		Action: bInvoke,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.ResourcesFree(ctx, opts.MsgOpts...)
		},
		Short: "Get free resources",
		Long: `Invokes the /dms/node/resources/free behavior on an actor

This behavior retrieves the free resources available on the node. The resources include CPU, RAM, GPU and disk space.
The returned units are in Hz for CPU clock speed, bytes for RAM, VRAM and disk space.

Examples:
	  nunet actor cmd --context user /dms/node/resources/free`,
	},

	behaviors.ResourcesOnboardedBehavior: {
		Action: bInvoke,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.ResourcesOnboarded(ctx, opts.MsgOpts...)
		},
		Short: "Get onboarded resources",
		Long: `Invokes the /dms/node/resources/onboarded behavior on an actor

This behavior retrieves the resources onboarded to the node. The resources include CPU, RAM, GPU and disk space.
The returned units are in Hz for CPU clock speed, bytes for RAM, VRAM and disk space.

Examples:
	  nunet actor cmd --context user /dms/node/resources/onboarded`,
	},
	behaviors.AllocationsListBehavior: {
		Action: bInvoke,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.AllocationsList(ctx, opts.MsgOpts...)
		},
		Short: "List allocations",
		Long: `Invokes the /dms/node/allocations/list behavior on an actor

This behavior retrieves information about all running allocations within your onboarded DMS.
The information includes allocation ID, status, executor type, container ID, resources, and port mappings.

Examples:
	  nunet actor cmd --context user /dms/node/allocations/list`,
	},
	behaviors.LoggerConfigBehavior: {
		Payload: func() any { return &node.LoggerConfigRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.LoggerConfigRequest)

			cmd.Flags().StringVarP(&p.URL, "url", "u", "", "Elasticsearch URL")
			cmd.Flags().StringVarP(&p.Level, "level", "l", "", "logging level (info, warn, debug etc.)")
			cmd.Flags().IntVarP(&p.Interval, "interval", "i", 0, "flush interval in seconds")
			cmd.MarkFlagsOneRequired("url", "level", "interval")
			cmd.Flags().StringVar(&p.APIKey, "api-key", "", "API Key for Elasticsearch and APM")
			cmd.Flags().StringVar(&p.APMURL, "apm-url", "", "APM Server URL")
			cmd.Flags().Bool("enable-elastic", false, "Enable Elasticsearch logging")
		},
		PreRunFn: func(cmd *cobra.Command, _ *cli.DmsCLI, opts actorCmdOptions) error {
			p, ok := opts.Payload.(*node.LoggerConfigRequest)
			if !ok {
				return fmt.Errorf("failed to decode payload")
			}
			flag := cmd.Flags().Lookup("enable-elastic")
			if flag != nil && flag.Changed {
				val, err := strconv.ParseBool(flag.Value.String())
				if err != nil {
					return fmt.Errorf("invalid value for --enable-elastic: %v", err)
				}
				p.ElasticEnabled = &val
			}
			return nil
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*node.LoggerConfigRequest)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}
			return dmsClient.LoggerConfig(ctx, *req, opts.MsgOpts...)
		},
		Action: bInvoke,
		Short:  "Adjust logger settings",
		Long: `Invokes the /dms/node/logger/config behavior on an actor

This behavior allows the user to adjust logger settings, i.e. logging level, flush interval and Elasticsearch URL.

Examples:

  nunet actor cmd --context user /dms/node/logger/config --level debug # set debug level
  nunet actor cmd --context user /dms/node/logger/config --url <elasticsearch-url>
  nunet actor cmd --context user /dms/node/logger/config --interval 10 # flush logs each 10 seconds
  nunet actor cmd --context user /dms/node/logger/config --api-key <api-key>
  nunet actor cmd --context user /dms/node/logger/config --apm-url <apm-url>
  nunet actor cmd --context user /dms/node/logger/config --enable-elastic`,
	},
	behaviors.HardwareSpecBehavior: {
		Action: bInvoke,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.HardwareSpec(ctx, opts.MsgOpts...)
		},
		Short: "Get hardware specifications",
		Long: `Invokes the /dms/node/hardware/spec behavior on an actor

This behavior retrieves the hardware specifications of the system.

Examples:

	nunet actor cmd --context user /dms/node/hardware/spec`,
	},
	behaviors.HardwareUsageBehavior: {
		Action: bInvoke,
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			return dmsClient.HardwareUsage(ctx, opts.MsgOpts...)
		},
		Short: "Get hardware usage",
		Long: `Invokes the /dms/node/hardware/usage behavior on an actor

This behavior retrieves the hardware usage of the system.

Examples:

	nunet actor cmd --context user /dms/node/hardware/usage`,
	},
	behaviors.CapListBehavior: {
		Action:  bInvoke,
		Payload: func() any { return &node.CapListRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.CapListRequest)
			cmd.Flags().StringVarP(&p.Context, "context", "c", "", "context name")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*node.CapListRequest)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}
			return dmsClient.CapList(ctx, *req, opts.MsgOpts...)
		},
		Short: "List capabilities",
		Long: `Invokes the /dms/cap/list behavior on an actor

This behavior retrieves a list of capabilities available on the node.

Examples:
  nunet actor cmd --context user /dms/cap/list`,
	},
	behaviors.ProvideCapAnchorBehavior: {
		Action:  bInvoke,
		Payload: func() any { return &CapAnchorRequestCmd{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*CapAnchorRequestCmd)
			cmd.Flags().StringVar(&p.Token, "token", "", "add revoke anchor")
			cmd.MarkFlagsOneRequired("token")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			payload, ok := opts.Payload.(*CapAnchorRequestCmd)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}

			var token ucan.Token
			if err := json.Unmarshal([]byte(payload.Token), &token); err != nil {
				return nil, err
			}

			req := &node.CapTokenAnchorRequest{
				Token: ucan.TokenList{
					Tokens: []*ucan.Token{
						&token,
					},
				},
			}

			return dmsClient.ProvideCapAnchor(ctx, *req, opts.MsgOpts...)
		},
		Short: "Anchors a capability token on the provide anchor of a node",
		Long: `Invokes the /dms/cap/provide/anchor behavior on an actor and requests to anchor on provide anchor.

This behavior invokes a node to anchor a token on the provide anchor.

Examples:

  nunet actor cmd --context user /dms/cap/provide/anchor --dest <peerID|did> --token <token>`,
	},

	behaviors.RequireCapAnchorBehavior: {
		Action:  bInvoke,
		Payload: func() any { return &CapAnchorRequestCmd{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*CapAnchorRequestCmd)
			cmd.Flags().StringVar(&p.Token, "token", "", "add revoke anchor")
			cmd.MarkFlagsOneRequired("token")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			payload, ok := opts.Payload.(*CapAnchorRequestCmd)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}

			var token ucan.Token
			if err := json.Unmarshal([]byte(payload.Token), &token); err != nil {
				return nil, err
			}

			req := &node.CapTokenAnchorRequest{
				Token: ucan.TokenList{
					Tokens: []*ucan.Token{
						&token,
					},
				},
			}

			return dmsClient.RequireCapAnchor(ctx, *req, opts.MsgOpts...)
		},
		Short: "Anchors a capability token on the require anchor of a node",
		Long: `Invokes the /dms/cap/require/anchor behavior on an actor and anchors a token on the require anchor.

This behavior invokes a node to anchor a token on the require anchor.

Examples:

  nunet actor cmd --context user /dms/cap/require/anchor --dest <peerID|did> --token <token>`,
	},

	behaviors.RevokeCapAnchorBehavior: {
		Action:  bInvoke,
		Payload: func() any { return &CapAnchorRequestCmd{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*CapAnchorRequestCmd)
			cmd.Flags().StringVar(&p.Token, "token", "", "add revoke anchor")
			cmd.MarkFlagsOneRequired("token")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			payload, ok := opts.Payload.(*CapAnchorRequestCmd)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}

			var token ucan.Token
			if err := json.Unmarshal([]byte(payload.Token), &token); err != nil {
				return nil, err
			}

			req := &node.CapTokenAnchorRequest{
				Token: ucan.TokenList{
					Tokens: []*ucan.Token{
						&token,
					},
				},
			}

			return dmsClient.RevokeCapAnchor(ctx, *req, opts.MsgOpts...)
		},
		Short: "Anchors revocation tokens on a node",
		Long: `Invokes the /dms/cap/revoke/anchor behavior on an actor and anchors a revocation token.

This behavior invokes a node to anchor a revocation token.

Examples:

  nunet actor cmd --context user /dms/cap/revoke/anchor --dest <peerID|did> --token <revocation_token>`,
	},

	behaviors.BroadcastRevokeCapBehavior: {
		Action:  bInvoke,
		Payload: func() any { return &CapAnchorRequestCmd{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*CapAnchorRequestCmd)
			cmd.Flags().StringVar(&p.Token, "token", "", "add revoke token")
			cmd.MarkFlagsOneRequired("token")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			payload, ok := opts.Payload.(*CapAnchorRequestCmd)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}

			var token ucan.Token
			if err := json.Unmarshal([]byte(payload.Token), &token); err != nil {
				return nil, err
			}

			req := &node.CapTokenAnchorRequest{
				Token: ucan.TokenList{
					Tokens: []*ucan.Token{
						&token,
					},
				},
			}

			return dmsClient.BroadcastCapRevoke(ctx, *req, opts.MsgOpts...)
		},
		Short: "Broadcast revocation capability anchors",
		Long: `Invokes the /dms/cap/revoke/broadcast behavior on an actor

This behavior broadcasts a revocation token.

Examples:

  nunet actor cmd --context user /dms/cap/revoke/broadcast --token <revocation_token>`,
	},

	behaviors.DeploymentDeleteBehavior: {
		Action:  bInvoke,
		Payload: func() any { return &node.DeploymentDeleteRequest{} },
		SetFlags: func(cmd *cobra.Command, payload any) {
			p := payload.(*node.DeploymentDeleteRequest)
			cmd.Flags().StringVar(&p.OrchestratorID, "id", "", "deployment id to delete (required)")
			_ = cmd.MarkFlagRequired("id")
		},
		RunFn: func(ctx context.Context, _ *cli.DmsCLI, dmsClient client.DmsClient, opts actorCmdOptions) (any, error) {
			req, ok := opts.Payload.(*node.DeploymentDeleteRequest)
			if !ok {
				return nil, fmt.Errorf("failed to decode payload")
			}
			return dmsClient.DeploymentDelete(ctx, *req, opts.MsgOpts...)
		},
		Short: "Delete a specific deployment",
		Long: `Invokes the /dms/node/deployment/delete behavior on an actor

This behavior removes a specific deployment by its deployment id.

Examples:
	nunet actor cmd --context user /dms/node/deployment/delete --id <deployment-id>`,
	},
}

func getDestinationHandle(cDID, cHost string) (string, error) {
	contractDID, err := did.FromString(cDID)
	if err != nil {
		return "", fmt.Errorf("failed to get contract did")
	}

	pubKey, err := did.PublicKeyFromDID(contractDID)
	if err != nil {
		return "", fmt.Errorf("failed to get contract did public key")
	}

	contractHostDID, err := did.FromString(cHost)
	if err != nil {
		return "", fmt.Errorf("failed to get contract host did")
	}

	hostPubKey, err := did.PublicKeyFromDID(contractHostDID)
	if err != nil {
		return "", fmt.Errorf("failed to get contract host public key")
	}

	hostPeerID, err := peer.IDFromPublicKey(hostPubKey)
	if err != nil {
		return "", fmt.Errorf("failed to get contract host peer id")
	}

	destination, err := actor.HandleFromPublicKeyWithInboxAddress(pubKey, cDID, hostPeerID.String())
	if err != nil {
		return "", fmt.Errorf("failed to get create remote handle")
	}

	d, err := json.Marshal(destination)
	if err != nil {
		return "", fmt.Errorf("failed to marshal destination handle")
	}

	return string(d), nil
}
