// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package db

import (
	"fmt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	job_types "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/types"
)

func ConnectDatabase(dbPath string) (*gorm.DB, error) {
	database, err := gorm.Open(sqlite.Open(fmt.Sprintf("%s/nunet.db", dbPath)), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database!")
	}

	_ = database.AutoMigrate(&types.FreeResources{})
	_ = database.AutoMigrate(&types.RequestTracker{})
	_ = database.AutoMigrate(&types.OnboardedResources{})
	_ = database.AutoMigrate(&types.MachineResources{})
	_ = database.AutoMigrate(&types.OnboardingConfig{})
	_ = database.AutoMigrate(&types.ResourceAllocation{})
	_ = database.AutoMigrate(&job_types.OrchestratorView{})
	_ = database.AutoMigrate(&types.GPU{})

	return database, nil
}
