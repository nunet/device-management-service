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

type PaymentModel string

const (
	PayPerAllocation          PaymentModel = "pay_per_allocation"
	PayPerDeployment          PaymentModel = "pay_per_deployment"
	PayPerTimeUtilization     PaymentModel = "pay_per_time_utilization"
	PayPerResourceUtilization PaymentModel = "pay_per_resource_utilization"
)

const (
	FiatMethod       PaymentType = "fiat"
	BlockchainMethod PaymentType = "blockchain"
)

type PaymentType string

// Payment represents a payment transaction
type PaymentDetails struct {
	PaymentType PaymentType `json:"payment_type"`
	Timestamp   time.Time   `json:"timestamp"`

	// payment model
	PaymentModel PaymentModel `json:"payment_model"`

	// pay per deployment payment model
	FeePerDeployment string `json:"fee_per_deployment,omitempty"`

	// pay per allocation payment model
	FeesPerAllocation string `json:"fees_per_allocation"`

	// pay per time utilization payment model
	FeePerTimeUnit string `json:"fee_per_time_unit,omitempty"` // e.g., "0.01" per second
	TimeUnit       string `json:"time_unit,omitempty"`         // "second", "minute", "hour"

	// pay per resource utilization payment model
	FeePerCPUCorePerTimeUnit string `json:"fee_per_cpu_core_per_time_unit,omitempty"` // e.g., "0.10" per core per hour
	FeePerRAMGBPerTimeUnit   string `json:"fee_per_ram_gb_per_time_unit,omitempty"`   // e.g., "0.05" per GB per hour
	FeePerDiskGBPerTimeUnit  string `json:"fee_per_disk_gb_per_time_unit,omitempty"`  // e.g., "0.01" per GB per hour
	FeePerGPUPerTimeUnit     string `json:"fee_per_gpu_per_time_unit,omitempty"`      // e.g., "5.00" per GPU per hour (optional)
	ResourceTimeUnit         string `json:"resource_time_unit,omitempty"`             // "second", "minute", "hour"

	Addresses []types.PaymentAddressInfo `json:"addresses"`
}
