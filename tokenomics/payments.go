// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build ignore

package tokenomics

import (
	"dbms"      // Assuming dbms package location
	"jobs"     // Assuming jobs package location
	"tokenomics" // Assuming tokenomics package location
	"telemetry"  //// Assuming tokenomics package location
)

// PaymentChannel represents the type of payment channel (fiat or blockchain)
type PaymentChannel string

const (
    FiatPayment      PaymentChannel = "fiat"
    BlockchainPayment PaymentChannel = "blockchain"
    // Add more payment channels as needed
)

// Payment represents a payment transaction
type Payment struct {
    Requestor      string
    Provider       string
	Currency       string 
    Timestamp      time.Time
    PaymentType    string          //PaymentType (like escrow vs. direct) 
    PaymentChannel PaymentChannel 
	Pricing        PricingMethod
   
}


// PricingMethod is an interface that can hold either FixedJobPricing or PeriodicPricing
type PricingMethod interface {
    GetFixedJobPricing() *FixedJobPricing
    GetPeriodicPricing() *PeriodicPricing
}

// FixedJobPricing represents the details for fixed job pricing
type FixedJobPricing struct {
    Price       int
    PlatformFee int 
}


// PeriodicPricing represents the details for periodic pricing
type PeriodicPricing struct {
    Price       int
    Period      string
    UsageLimits UsageLimits
    PlatformFee int
}

// UsageLimits represents the usage limits or quotas for periodic pricing
type UsageLimits struct {
    MaxCPUHours        int
    MaxMemoryUsage     int
    MaxStorageUsage    int
    MaxNetworkBandwidth int
}



// PaymentGateway defines the operations for managing payments and settlements
type PaymentGateway interface {
	telemetry telemetry.message // Importing messages from telemetry package
	Deposit(contractID int, payment Payment) error
	SettleContract(contractID int, verificationResult jobs.JobVerificationResult) error
}


// Deposit handles the deposit logic for payments
func ( PaymentGateway) Deposit(contractID int, payment Payment, pricing PricingMethod) error {
	switch payment.Method {
	case "direct":
		return nil

	case "escrow":
		switch payment.PaymentType {
		case "fiat":
		case "crypto":
		default:
			return fmt.Errorf("invalid payment type")
		}
	default:
		return fmt.Errorf("invalid payment method")
	}
	return nil
}

// SettleContract handles the settlement based on job completion and payment type
func (PaymentGateway) SettleContract(contractID int, verificationResult jobs.JobVerificationResult) error {

	
	if verificationResult.Success {
		paymentAmount := contract.Price * (float64(verificationResult.Percentage) / 100) //based on job completion percentage

		switch contract.Method {
		case "escrow":
			err := pg.processEscrowPayment(contract, paymentAmount)
			
		case "direct":
			err := pg.processDirectPayment(contract, paymentAmount)
		
		default:
			return fmt.Errorf("invalid payment method")
		}
	}

	if verificationResult.Error {
		fmt.Printf("Job failed. %d\n", contractID)
		if contract.Method == "escrow" {
			err := pg.refundEscrowPayment(contract)
			if err != nil {
				return err
			}
		}
		return nil
	}
	return nil
}

