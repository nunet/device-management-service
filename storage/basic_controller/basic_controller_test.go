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
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/storage"
	"gitlab.com/nunet/device-management-service/telemetry"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	mockFileData = "hello world"
	mockFileSize = 11
)

type VolumeControllerTestSuite struct {
	suite.Suite
	vcKit *VolumeControllerTestKit
}

func TestVolumeControllerTestSuite(t *testing.T) {
	suite.Run(t, new(VolumeControllerTestSuite))
}

// SetupTest prepares a VolumeController with two volumes (with db
// and fs configured) in which data is written to the second volume.
func (s *VolumeControllerTestSuite) SetupTest() {
	// Initialize telemetry in test mode, replacing the global st
	st = telemetry.NewTelemetry(nil, nil, true)

	basePath := "/home/.nunet/volumes/"
	volumes := map[string]*types.StorageVolume{
		"volume1": {
			Path:           basePath + "volume1",
			ReadOnly:       false,
			Private:        false,
			EncryptionType: types.EncryptionTypeNull,
		},
		"volume2": {
			CID:            "baf222",
			Path:           basePath + "volume2",
			ReadOnly:       false,
			Private:        false,
			EncryptionType: types.EncryptionTypeNull,
		},
	}

	var err error
	s.vcKit, err = SetupVolumeControllerTestKit(basePath, volumes)
	assert.NoError(s.T(), err)

	// Write a file in volume2
	err = afero.WriteFile(s.vcKit.Fs, s.vcKit.Volumes["volume2"].Path+"/file.txt", []byte(mockFileData), 0o644)
	assert.NoError(s.T(), err)
}

func (s *VolumeControllerTestSuite) TearDownTest() {
	// Clean up the test environment
	s.vcKit = nil
}

func (s *VolumeControllerTestSuite) TestCreateVolume() {
	// Test case 1: Create a volume without options
	vol1, err := s.vcKit.BasicVolController.CreateVolume(storage.VolumeSourceS3)
	assert.NoError(s.T(), err)

	// Test case 2: Create a volume with private option
	vol2, err := s.vcKit.BasicVolController.CreateVolume(storage.VolumeSourceS3, WithPrivate[storage.CreateVolOpt]())
	assert.NoError(s.T(), err)

	// Verify returned volume details for test case 1
	assert.NotEmpty(s.T(), vol1.Path)
	assert.Empty(s.T(), vol1.CID)
	assert.Equal(s.T(), false, vol1.Private)
	assert.Equal(s.T(), false, vol1.ReadOnly)
	assert.Equal(s.T(), types.EncryptionTypeNull, vol1.EncryptionType)

	// Verify returned volume details for test case 2
	assert.NotEmpty(s.T(), vol2.Path)
	assert.Empty(s.T(), vol2.CID)
	assert.Equal(s.T(), true, vol2.Private)
	assert.Equal(s.T(), false, vol2.ReadOnly)
	assert.Equal(s.T(), types.EncryptionTypeNull, vol2.EncryptionType)

	// Verify that the volumes are stored in the database
	volumes, err := s.vcKit.BasicVolController.repo.FindAll(
		context.Background(),
		s.vcKit.BasicVolController.repo.GetQuery(),
	)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), volumes, 4) // there are already 2 volumes created in the suite
	// TODO-maybe: should we also check the DB content for each volume?

	// check if directories were created based on volumes path
	fileInfoVol1, err := s.vcKit.Fs.Stat(vol1.Path)
	assert.NoError(s.T(), err)
	assert.True(s.T(), fileInfoVol1.IsDir())

	fileInfoVol2, err := s.vcKit.Fs.Stat(vol2.Path)
	assert.NoError(s.T(), err)
	assert.True(s.T(), fileInfoVol2.IsDir())
}

func (s *VolumeControllerTestSuite) TestLockVolume() {
	testCases := []struct {
		name         string
		volumePath   string
		cid          string
		private      bool
		expectError  bool
		setupFunc    func()
		teardownFunc func()
	}{
		{
			name:        "Lock volume with CID",
			volumePath:  s.vcKit.Volumes["volume1"].Path,
			cid:         "abcdef",
			private:     false,
			expectError: false,
		},
		{
			name:        "Lock volume with private option",
			volumePath:  s.vcKit.Volumes["volume2"].Path,
			cid:         s.vcKit.Volumes["volume2"].CID,
			private:     true,
			expectError: false,
		},
		{
			name:        "Lock non-existent volume",
			volumePath:  "/path/to/non-existent-volume",
			cid:         "",
			private:     false,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			var opts []storage.LockVolOpt
			if tc.cid != "" {
				opts = append(opts, WithCID(tc.cid))
			}
			if tc.private {
				opts = append(opts, WithPrivate[storage.LockVolOpt]())
			}

			err := s.vcKit.BasicVolController.LockVolume(tc.volumePath, opts...)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// verify database fields (readOnly must be true)
				query := s.vcKit.BasicVolController.repo.GetQuery()
				query.Conditions = append(query.Conditions, repositories.EQ("Path", tc.volumePath))
				vol, err := s.vcKit.BasicVolController.repo.Find(context.Background(), query)
				assert.NoError(t, err)
				assert.True(t, vol.ReadOnly)

				// checking CID and Private as some test cases inputed them
				assert.Equal(t, tc.cid, vol.CID)
				assert.Equal(t, tc.private, vol.Private)

				// verifying volume dir is read-only
				fileInfo, err := s.vcKit.Fs.Stat(tc.volumePath)
				assert.NoError(t, err)
				assert.Equal(t, os.FileMode(0o400), fileInfo.Mode().Perm())
			}
		})
	}
}

func (s *VolumeControllerTestSuite) TestDeleteVolume() {
	testCases := []struct {
		name           string
		identifier     string
		identifierType storage.IDType
		expectError    bool
		setupFunc      func()
		teardownFunc   func()
	}{
		{
			name:           "Delete volume by path",
			identifier:     s.vcKit.Volumes["volume1"].Path,
			identifierType: storage.IDTypePath,
			expectError:    false,
		},
		{
			name:           "Delete volume by CID",
			identifier:     s.vcKit.Volumes["volume2"].CID,
			identifierType: storage.IDTypeCID,
			expectError:    false,
		},
		{
			name:           "Delete non-existent volume by path",
			identifier:     "/path/to/non-existent-volume",
			identifierType: storage.IDTypePath,
			expectError:    true,
		},
		{
			name:           "Delete non-existent volume by CID",
			identifier:     "non-existent-cid",
			identifierType: storage.IDTypeCID,
			expectError:    true,
		},
		{
			name:           "Delete with unsupported identifier type",
			identifier:     "some-identifier",
			identifierType: storage.IDType(999), // An unsupported type
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			err := s.vcKit.BasicVolController.DeleteVolume(tc.identifier, tc.identifierType)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Verify that the volume is deleted from the filesystem
				_, err = s.vcKit.Fs.Stat(tc.identifier)
				assert.True(t, os.IsNotExist(err), "Volume directory should not exist after deletion")

				// Verify that the volume is deleted from the database
				query := s.vcKit.BasicVolController.repo.GetQuery()
				if tc.identifierType == storage.IDTypeCID {
					query.Conditions = append(query.Conditions, repositories.EQ("CID", tc.identifier))
				} else {
					query.Conditions = append(query.Conditions, repositories.EQ("Path", tc.identifier))
				}
				_, err = s.vcKit.BasicVolController.repo.Find(context.Background(), query)
				assert.Equal(t, repositories.ErrNotFound, err, "Volume should not exist in the database after deletion")
			}
		})
	}

	// Verify that all volumes are deleted from the database
	volumes, err := s.vcKit.BasicVolController.repo.FindAll(context.Background(), s.vcKit.BasicVolController.repo.GetQuery())
	assert.NoError(s.T(), err)
	assert.Len(s.T(), volumes, 0)
}

func (s *VolumeControllerTestSuite) TestListVolumes() {
	volumes, err := s.vcKit.BasicVolController.ListVolumes()
	assert.NoError(s.T(), err)

	assert.Len(s.T(), volumes, len(s.vcKit.Volumes))

	// assert details of returned volumes
	for _, retVol := range volumes {
		// Find the corresponding volume in the test suite's volumes map
		expectedVol, ok := s.vcKit.Volumes[filepath.Base(retVol.Path)]
		assert.True(s.T(), ok, "Unexpected volume returned: %s", retVol.Path)

		// Assert the properties of the returned volume
		assert.Equal(s.T(), expectedVol.CID, retVol.CID)
		assert.Equal(s.T(), expectedVol.Path, retVol.Path)
		assert.Equal(s.T(), expectedVol.ReadOnly, retVol.ReadOnly)
		assert.Equal(s.T(), expectedVol.Private, retVol.Private)
		assert.Equal(s.T(), expectedVol.EncryptionType, retVol.EncryptionType)
	}
}

func (s *VolumeControllerTestSuite) TestGetSize() {
	testCases := []struct {
		name         string
		identifier   string
		idType       storage.IDType
		expectedSize int64
		expectErr    bool
	}{
		{
			name:         "Get size by path",
			identifier:   s.vcKit.Volumes["volume1"].Path,
			idType:       storage.IDTypePath,
			expectedSize: 0,
			expectErr:    false,
		},
		{
			name:         "Get size by CID",
			identifier:   s.vcKit.Volumes["volume2"].CID,
			idType:       storage.IDTypeCID,
			expectedSize: mockFileSize,
			expectErr:    false,
		},
		{
			name:         "Get size with unsupported ID type",
			identifier:   "some-identifier",
			idType:       storage.IDType(999),
			expectedSize: 0,
			expectErr:    true,
		},
		{
			name:         "Get size of non-existent volume",
			identifier:   "non-existent-path",
			idType:       storage.IDTypePath,
			expectedSize: 0,
			expectErr:    true,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			size, err := s.vcKit.BasicVolController.GetSize(tc.identifier, tc.idType)

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedSize, size)
			}
		})
	}
}
