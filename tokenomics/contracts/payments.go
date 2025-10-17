// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package contracts

import (
	"time"

	"gitlab.com/nunet/device-management-service/types"
)

const (
	FiatMethod       PaymentType = "fiat"
	BlockchainMethod PaymentType = "blockchain"
)

type PaymentType string

// Payment represents a payment transaction
type PaymentDetails struct {
	PaymentType       PaymentType                `json:"payment_type"`
	Timestamp         time.Time                  `json:"timestamp"`
	FeesPerAllocation string                     `json:"fees_per_allocation"`
	Addresses         []types.PaymentAddressInfo `json:"addresses"`
}
