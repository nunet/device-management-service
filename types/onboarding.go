// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package types

import "context"

// OnboardingConfig - parameters to configure onboarding
type OnboardingConfig struct {
	BaseDBModel
	IsOnboarded bool `json:"is_onboarded"`

	// These are not stored in the database, but are part of the onboarding config
	// during the get onboarding config call these are populated from the resource manager and hardware
	OnboardedResources Resources `json:"onboarded_resources,omitempty" gorm:"-" clover:"-"`
	MachineResources   Resources `json:"machine_resources,omitempty" gorm:"-" clover:"-"`
}

// OnboardingManager - interface for onboarding
type OnboardingManager interface {
	IsOnboarded() (bool, error)
	Info(ctx context.Context) (OnboardingConfig, error)
	Onboard(ctx context.Context, config OnboardingConfig) (OnboardingConfig, error)
	Offboard(ctx context.Context) error
}
