// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package gorm

import (
	"gorm.io/gorm"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/types"
)

type OnboardingConfigGORM struct {
	repositories.GenericEntityRepository[types.OnboardingConfig]
}

func NewOnboardingConfig(db *gorm.DB) repositories.OnboardingConfig {
	return &OnboardingConfigGORM{
		NewGenericEntityRepository[types.OnboardingConfig](db),
	}
}
