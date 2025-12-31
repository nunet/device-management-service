// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package processors

import (
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
	"gitlab.com/nunet/device-management-service/tokenomics/store/usage"
)

// InitPaymentModelProcessors initializes and registers all payment model processors.
// This should be called during application startup.
func InitPaymentModelProcessors(store *usage.Store) {
	if store == nil {
		panic("usage store cannot be nil")
	}

	contracts.RegisterPaymentModelProcessor(
		contracts.PayPerAllocation,
		NewPayPerAllocationProcessor(store),
	)

	contracts.RegisterPaymentModelProcessor(
		contracts.PayPerDeployment,
		NewPayPerDeploymentProcessor(store),
	)

	contracts.RegisterPaymentModelProcessor(
		contracts.PayPerTimeUtilization,
		NewPayPerTimeUtilizationProcessor(store),
	)

	contracts.RegisterPaymentModelProcessor(
		contracts.PayPerResourceUtilization,
		NewPayPerResourceUtilizationProcessor(store),
	)

	contracts.RegisterPaymentModelProcessor(
		contracts.FixedRental,
		NewFixedRentalProcessor(store),
	)

	contracts.RegisterPaymentModelProcessor(
		contracts.Periodic,
		NewPeriodicProcessor(store),
	)
}
