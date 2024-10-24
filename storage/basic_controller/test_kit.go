// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package basiccontroller

import (
	"context"
	"fmt"

	"github.com/spf13/afero"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	rGorm "gitlab.com/nunet/device-management-service/db/repositories/gorm"
	"gitlab.com/nunet/device-management-service/telemetry"
	"gitlab.com/nunet/device-management-service/types"
)

type VolumeControllerTestKit struct {
	BasicVolController *BasicVolumeController
	Fs                 afero.Fs
	Volumes            map[string]*types.StorageVolume
}

// SetupVolumeControllerTestKit sets up a volume controller with 0-n volumes given a base path.
// If volumes are inputed, directories will be created and volumes will be stored in the database
func SetupVolumeControllerTestKit(basePath string, volumes map[string]*types.StorageVolume) (*VolumeControllerTestKit, error) {
	// Initialize telemetry in test mode, replacing the global st
	// It's initiated here too, besides on basic_controller_test.go, because
	// s3 tests depend on basicController (which in turn depends on telemetry instantiation).
	// S3 are calling this SetupVolControllerTestSuite, so it's one way to initialize telemetry
	// for basic controller
	st = telemetry.NewTelemetry(nil, nil, true)

	db, err := gorm.Open(
		sqlite.Open("file:?mode=memory&cache=shared"),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create in-memory mock database: %w", err)
	}

	err = db.AutoMigrate(&types.StorageVolume{})
	if err != nil {
		return nil, fmt.Errorf("failed to automigrate: %w", err)
	}

	fs := afero.NewMemMapFs()

	err = fs.MkdirAll(basePath, 0o755)
	if err != nil {
		return nil, fmt.Errorf("failed to create base path: %w", err)
	}

	repo := rGorm.NewStorageVolume(db)
	vc, err := NewDefaultVolumeController(repo, basePath, fs)
	if err != nil {
		return nil, fmt.Errorf("failed to create volume controller: %w", err)
	}

	for _, vol := range volumes {
		// create root volume dir
		err = fs.MkdirAll(vol.Path, 0o755)
		if err != nil {
			return nil, fmt.Errorf("failed to create volume dir: %w", err)
		}

		// create volume record in db
		_, err = repo.Create(context.Background(), *vol)
		if err != nil {
			return nil, fmt.Errorf("failed to create volume record: %w", err)
		}
	}

	return &VolumeControllerTestKit{
		BasicVolController: vc,
		Fs:                 fs,
		Volumes:            volumes,
	}, nil
}
