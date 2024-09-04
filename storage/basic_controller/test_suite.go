package basiccontroller

import (
	"context"
	"fmt"
	"os"
	"testing"

	clover "github.com/ostafen/clover/v2"
	"github.com/spf13/afero"

	rclover "gitlab.com/nunet/device-management-service/db/repositories/clover"
	"gitlab.com/nunet/device-management-service/telemetry"
	"gitlab.com/nunet/device-management-service/types"
)

type VolControllerTestSuiteHelper struct {
	BasicVolController *BasicVolumeController
	Fs                 afero.Fs
	DB                 *clover.DB
	Volumes            map[string]*types.StorageVolume
	TempDBDir          string
}

// SetupVolControllerTestSuite sets up a volume controller with 0-n volumes given a base path.
// If volumes are inputed, directories will be created and volumes will be stored in the database
func SetupVolControllerTestSuite(t *testing.T, basePath string, volumes map[string]*types.StorageVolume) (*VolControllerTestSuiteHelper, error) {
	// Initialize telemetry in test mode, replacing the global st
	// It's initiated here too, besides on basic_controller_test.go, because
	// s3 tests depend on basicController (which in turn depends on telemetry instantiation).
	// S3 are calling this SetupVolControllerTestSuite, so it's one way to initialize telemetry
	// for basic controller
	st = telemetry.NewTelemetry(nil, nil, true)

	tempDir, err := os.MkdirTemp("", "clover-test-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	db, err := rclover.NewDB(tempDir, []string{"storage_volume"})
	if err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to open clover db: %w", err)
	}

	fs := afero.NewMemMapFs()

	err = fs.MkdirAll(basePath, 0o755)
	if err != nil {
		db.Close()
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to create base path: %w", err)
	}

	repo := rclover.NewStorageVolume(db)
	vc, err := NewDefaultVolumeController(repo, basePath, fs)
	if err != nil {
		db.Close()
		os.Remove(tempDir)
		return nil, fmt.Errorf("failed to create volume controller: %w", err)
	}

	for _, vol := range volumes {
		// create root volume dir
		err = fs.MkdirAll(vol.Path, 0o755)
		if err != nil {
			db.Close()
			os.Remove(tempDir)
			return nil, fmt.Errorf("failed to create volume dir: %w", err)
		}

		// create volume record in db
		_, err = repo.Create(context.Background(), *vol)
		if err != nil {
			db.Close()
			os.Remove(tempDir)
			return nil, fmt.Errorf("failed to create volume record: %w", err)
		}
	}

	helper := &VolControllerTestSuiteHelper{vc, fs, db, volumes, tempDir}

	t.Cleanup(func() {
		TearDownVolControllerTestSuite(helper)
	})

	return helper, nil
}

// TearDownVolControllerTestSuite cleans up resources created during setup
func TearDownVolControllerTestSuite(helper *VolControllerTestSuiteHelper) {
	if helper.DB != nil {
		helper.DB.Close()
	}
	if helper.TempDBDir != "" {
		os.RemoveAll(helper.TempDBDir)
	}
}
