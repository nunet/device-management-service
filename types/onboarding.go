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

	// OnboardedResources - resources that are onboarded
	// this is a transient field and not stored in the database directly
	// it is populated using the ResourceManager
	OnboardedResources Resources `json:"onboarded_resources,omitempty" clover:"-"`
}

// OnboardingManager - interface for onboarding
type OnboardingManager interface {
	IsOnboarded() (bool, error)
	Info(ctx context.Context) (OnboardingConfig, error)
	Onboard(ctx context.Context, config OnboardingConfig) (OnboardingConfig, error)
	Offboard(ctx context.Context) error
}
