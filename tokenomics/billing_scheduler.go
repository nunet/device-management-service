package tokenomics

import (
	"errors"
	"fmt"
	"sync"
	"time"

	bt "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
	contractstore "gitlab.com/nunet/device-management-service/tokenomics/store"
	"gitlab.com/nunet/device-management-service/tokenomics/store/usage"
)

// BillingTaskArgs contains the arguments for a billing task
type BillingTaskArgs struct {
	ContractDID    did.DID
	ContractStore  *contractstore.Store
	UsageStore     *usage.Store
	ExecuteBilling func(contractDID did.DID) error
}

// ContractBillingScheduler manages all contract billing tasks using a centralized scheduler
type ContractBillingScheduler struct {
	scheduler     *bt.Scheduler
	contractStore *contractstore.Store
	usageStore    *usage.Store
	tasks         map[string]*bt.Task // contract DID URI -> task
	mu            sync.RWMutex
	billingFunc   func(contractDID did.DID) error // Function to execute billing
}

// NewContractBillingScheduler creates a new billing scheduler
func NewContractBillingScheduler(
	contractStore *contractstore.Store,
	usageStore *usage.Store,
	billingFunc func(contractDID did.DID) error,
) (*ContractBillingScheduler, error) {
	// Create scheduler with reasonable limits
	// - Max 10 concurrent billing tasks (adjust based on needs)
	// - Poll every 30 seconds to check for ready tasks
	scheduler := bt.NewScheduler(10, 30*time.Second)

	if contractStore == nil {
		return nil, errors.New("contract store is required")
	}
	if usageStore == nil {
		return nil, errors.New("usage store is required")
	}
	if billingFunc == nil {
		return nil, errors.New("billing function is required")
	}

	return &ContractBillingScheduler{
		scheduler:     scheduler,
		contractStore: contractStore,
		usageStore:    usageStore,
		tasks:         make(map[string]*bt.Task),
		billingFunc:   billingFunc,
	}, nil
}

// Start starts the billing scheduler
func (cbs *ContractBillingScheduler) Start() {
	cbs.scheduler.Start()
}

// Stop stops the billing scheduler and waits for all tasks to complete
func (cbs *ContractBillingScheduler) Stop() {
	cbs.scheduler.Stop()
}

// RegisterContract registers a contract for automatic billing
// This method is idempotent - calling it multiple times for the same contract is safe
func (cbs *ContractBillingScheduler) RegisterContract(contractDID did.DID) error {
	cbs.mu.Lock()
	defer cbs.mu.Unlock()

	contractURI := contractDID.URI

	// Check if already registered (idempotent check)
	// This prevents duplicate registration if called from both createContractOnHost
	// and StartContracts
	if _, exists := cbs.tasks[contractURI]; exists {
		return nil // Already registered, no-op
	}

	// Get contract to check payment model
	contract, err := cbs.contractStore.GetContract(contractURI)
	if err != nil {
		return fmt.Errorf("failed to get contract: %w", err)
	}

	// Check if payment model supports automatic billing
	processor, err := contracts.GetPaymentModelProcessor(contract.PaymentDetails.PaymentModel)
	if err != nil {
		return fmt.Errorf("failed to get payment processor: %w", err)
	}

	if !processor.SupportsAutomaticBilling() {
		// Not applicable for this payment model
		return nil
	}

	// Calculate billing cycle and check interval
	billingCycle := calculateBillingCycle(contract)
	checkInterval := calculateCheckInterval(billingCycle)

	// Get last invoice time
	lastInvoiceAt, err := cbs.usageStore.GetLastProcessedAt(contractURI)
	if err != nil {
		// If not found, use contract start date
		lastInvoiceAt = contract.Duration.StartDate
	}
	if lastInvoiceAt.IsZero() {
		lastInvoiceAt = contract.Duration.StartDate
	}

	// Create billing cycle trigger - calculates exact next invoice time
	trigger := NewBillingCycleTrigger(billingCycle, lastInvoiceAt, checkInterval)

	// Create task arguments
	taskArgs := &BillingTaskArgs{
		ContractDID:    contractDID,
		ContractStore:  cbs.contractStore,
		UsageStore:     cbs.usageStore,
		ExecuteBilling: cbs.billingFunc,
	}

	// Create task - inlined function for simplicity
	task := &bt.Task{
		Name:        fmt.Sprintf("billing-%s", contractURI),
		Description: fmt.Sprintf("Automatic billing for contract %s (%s)", contractURI, contract.PaymentDetails.PaymentModel),
		Triggers:    []bt.Trigger{trigger},
		Function: func(args interface{}) error {
			// The scheduler passes task.Args ([]interface{}) directly, so we need to extract the first element
			argsSlice, ok := args.([]interface{})
			if !ok || len(argsSlice) == 0 {
				return fmt.Errorf("invalid billing task args")
			}
			billingArgs, ok := argsSlice[0].(*BillingTaskArgs)
			if !ok {
				return fmt.Errorf("invalid billing task args type")
			}
			return billingArgs.ExecuteBilling(billingArgs.ContractDID)
		},
		Args:    []interface{}{taskArgs},
		Enabled: true,
	}

	// Register task with scheduler
	cbs.scheduler.AddTask(task)
	cbs.tasks[contractURI] = task

	log.Infow("registered contract for automatic billing",
		"labels", string(observability.LabelContract),
		"contract_did", contractURI,
		"payment_model", contract.PaymentDetails.PaymentModel,
		"billing_cycle", billingCycle,
		"check_interval", checkInterval)

	return nil
}

// UnregisterContract removes a contract from automatic billing
func (cbs *ContractBillingScheduler) UnregisterContract(contractDID did.DID) {
	cbs.mu.Lock()
	defer cbs.mu.Unlock()

	contractURI := contractDID.URI
	task, exists := cbs.tasks[contractURI]
	if !exists {
		return
	}

	cbs.scheduler.RemoveTask(task.ID)
	delete(cbs.tasks, contractURI)

	log.Infow("unregistered contract from automatic billing",
		"labels", string(observability.LabelContract),
		"contract_did", contractURI)
}

// GetTask returns the task for a contract (for observability)
func (cbs *ContractBillingScheduler) GetTask(contractDID did.DID) (*bt.Task, bool) {
	cbs.mu.RLock()
	defer cbs.mu.RUnlock()

	task, exists := cbs.tasks[contractDID.URI]
	return task, exists
}

// UpdateContract updates billing schedule after successful invoice
// Updates the trigger's lastInvoiceAt to use actual invoice time
func (cbs *ContractBillingScheduler) UpdateContract(contractDID did.DID) error {
	cbs.mu.Lock()
	defer cbs.mu.Unlock()

	contractURI := contractDID.URI
	task, exists := cbs.tasks[contractURI]
	if !exists {
		return nil
	}

	// Get updated lastInvoiceAt from store
	// Note: This should be called after SaveLastProcessedAt in contract_host.go
	lastInvoiceAt, err := cbs.usageStore.GetLastProcessedAt(contractURI)
	if err != nil {
		// If we can't get the time from store, log warning but don't fail
		// The trigger will still work, just might not be perfectly accurate
		log.Warnw("failed to get last invoice time for trigger update, using current time",
			"labels", string(observability.LabelContract),
			"contract_did", contractURI,
			"error", err)
		// Use current time as fallback
		lastInvoiceAt = time.Now().UTC()
	}

	// Update trigger's lastInvoiceAt
	for _, trigger := range task.Triggers {
		if billingTrigger, ok := trigger.(*BillingCycleTrigger); ok {
			billingTrigger.UpdateLastInvoiceAt(lastInvoiceAt)
			log.Debugw("updated billing trigger after invoice",
				"labels", string(observability.LabelContract),
				"contract_did", contractURI,
				"last_invoice_at", lastInvoiceAt,
				"next_check", billingTrigger.nextCheck)
			break
		}
	}

	return nil
}

// calculateBillingCycle calculates the full billing cycle duration from PaymentPeriod × PaymentPeriodCount
func calculateBillingCycle(contract *contracts.Contract) time.Duration {
	paymentPeriod := contract.PaymentDetails.PaymentPeriod
	paymentPeriodCount := contract.PaymentDetails.PaymentPeriodCount
	if paymentPeriodCount <= 0 {
		paymentPeriodCount = 1
	}

	periodDuration, err := parsePaymentPeriod(paymentPeriod)
	if err != nil {
		// Default to 1 hour if invalid period
		periodDuration = time.Hour
	}

	return periodDuration * time.Duration(paymentPeriodCount)
}

// calculateCheckInterval calculates the dynamic checker interval based on billing period
// Formula: clamp(billingCycle / 10, 30s, min(billingCycle / 2, 24h))
// This ensures:
// - Short periods (minutes): frequent checks (30s minimum)
// - Long periods (days/weeks/months): efficient checks (24h maximum)
func calculateCheckInterval(billingCycle time.Duration) time.Duration {
	// Use 1/10 of billing cycle as base for reasonable granularity
	// This means we'll check 10 times per billing cycle (good for short periods)
	checkInterval := billingCycle / 10

	// Apply minimum bound: 30 seconds (for fast testing with 1-minute periods)
	minInterval := 30 * time.Second
	if checkInterval < minInterval {
		checkInterval = minInterval
	}

	// Apply maximum bound: Use 1/2 of period, but cap at 24 hours
	// This prevents excessive checks for very long periods
	// For 1-month period: 1/2 month = 15 days → capped at 24 hours
	// For 1-week period: 1/2 week = 3.5 days → capped at 24 hours
	// For 1-day period: 1/2 day = 12 hours → use 12 hours
	maxInterval := billingCycle / 2
	absoluteMaxInterval := 24 * time.Hour
	if maxInterval > absoluteMaxInterval {
		maxInterval = absoluteMaxInterval
	}

	if checkInterval > maxInterval {
		checkInterval = maxInterval
	}

	return checkInterval
}

// parsePaymentPeriod converts a payment period string to a time.Duration
func parsePaymentPeriod(period string) (time.Duration, error) {
	switch period {
	case contracts.PaymentPeriodMinute:
		return time.Minute, nil
	case contracts.PaymentPeriodHour:
		return time.Hour, nil
	case contracts.PaymentPeriodDay:
		return 24 * time.Hour, nil
	case contracts.PaymentPeriodWeek:
		return 7 * 24 * time.Hour, nil
	case contracts.PaymentPeriodMonth:
		// Approximate: 30 days (could be enhanced to handle exact calendar months)
		return 30 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("invalid payment_period: %s", period)
	}
}
