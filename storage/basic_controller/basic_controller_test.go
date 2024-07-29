package basic_controller

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/models"
	"gitlab.com/nunet/device-management-service/storage"
)

type VolumeControllerTestSuite struct {
	suite.Suite
	vcHelper *VolControllerTestSuiteHelper
}

func TestVolumeControllerTestSuite(t *testing.T) {
	suite.Run(t, new(VolumeControllerTestSuite))
}

func (s *VolumeControllerTestSuite) SetupTest() {
	basePath := "/home/.nunet/volumes/"

	volumes := map[string]*models.StorageVolume{
		"volume1": {
			Path:           basePath + "volume1",
			ReadOnly:       false,
			Private:        false,
			EncryptionType: models.EncryptionTypeNull,
		},
		"volume2": {
			CID:            "baf222",
			Path:           basePath + "volume2",
			ReadOnly:       false,
			Private:        false,
			EncryptionType: models.EncryptionTypeNull,
		},
	}

	var err error
	s.vcHelper, err = SetupVolControllerTestSuite(s.T(), basePath, volumes)
	assert.NoError(s.T(), err)

	// Write a file in volume2
	err = afero.WriteFile(s.vcHelper.Fs, s.vcHelper.Volumes["volume2"].Path+"/file.txt", []byte("hello world"), 0644)
	assert.NoError(s.T(), err)
}

func (s *VolumeControllerTestSuite) TearDownTest() {
	// Clean up the test environment
	TearDownVolControllerTestSuite(s.vcHelper)
	s.vcHelper = nil
}

func (s *VolumeControllerTestSuite) TestCreateVolume() {
	// Test case 1: Create a volume without options
	vol1, err := s.vcHelper.BasicVolController.CreateVolume(storage.VolumeSourceS3)
	assert.NoError(s.T(), err)

	// Test case 2: Create a volume with private option
	vol2, err := s.vcHelper.BasicVolController.CreateVolume(storage.VolumeSourceS3, WithPrivate[storage.CreateVolOpt]())
	assert.NoError(s.T(), err)

	// Verify returned volume details for test case 1
	assert.NotEmpty(s.T(), vol1.Path)
	assert.Empty(s.T(), vol1.CID)
	assert.Equal(s.T(), false, vol1.Private)
	assert.Equal(s.T(), false, vol1.ReadOnly)
	assert.Equal(s.T(), models.EncryptionTypeNull, vol1.EncryptionType)

	// Verify returned volume details for test case 2
	assert.NotEmpty(s.T(), vol2.Path)
	assert.Empty(s.T(), vol2.CID)
	assert.Equal(s.T(), true, vol2.Private)
	assert.Equal(s.T(), false, vol2.ReadOnly)
	assert.Equal(s.T(), models.EncryptionTypeNull, vol2.EncryptionType)

	// Verify that the volumes are stored in the database
	volumes, err := s.vcHelper.BasicVolController.repo.FindAll(
		context.Background(),
		s.vcHelper.BasicVolController.repo.GetQuery(),
	)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), volumes, 4) // there are already 2 volumes created in the suite
	// TODO-maybe: should we also check the DB content for each volume?

	// check if directories were created based on volumes path
	fileInfoVol1, err := s.vcHelper.Fs.Stat(vol1.Path)
	assert.NoError(s.T(), err)
	assert.True(s.T(), fileInfoVol1.IsDir())

	fileInfoVol2, err := s.vcHelper.Fs.Stat(vol2.Path)
	assert.NoError(s.T(), err)
	assert.True(s.T(), fileInfoVol2.IsDir())
}

func (s *VolumeControllerTestSuite) TestLockVolume() {
	testCases := []struct {
		name        string
		volumePath  string
		cid         string
		private     bool
		expectError bool
	}{
		{
			name:        "Lock volume with CID",
			volumePath:  s.vcHelper.Volumes["volume1"].Path,
			cid:         "abcdef",
			private:     false,
			expectError: false,
		},
		{
			name:        "Lock volume with private option",
			volumePath:  s.vcHelper.Volumes["volume2"].Path,
			cid:         s.vcHelper.Volumes["volume2"].CID,
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

			err := s.vcHelper.BasicVolController.LockVolume(tc.volumePath, opts...)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// verify database fields (readOnly must be true)
				query := s.vcHelper.BasicVolController.repo.GetQuery()
				query.Conditions = append(query.Conditions, repositories.EQ("Path", tc.volumePath))
				vol, err := s.vcHelper.BasicVolController.repo.Find(context.Background(), query)
				assert.NoError(t, err)
				assert.True(t, vol.ReadOnly)
				// checking CID and Private as some test cases inputed them
				assert.Equal(t, tc.cid, vol.CID)
				assert.Equal(t, tc.private, vol.Private)

				// verifying volume dir is read-only
				fileInfo, err := s.vcHelper.Fs.Stat(tc.volumePath)
				assert.NoError(t, err)
				assert.Equal(t, os.FileMode(0400), fileInfo.Mode().Perm())
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
	}{
		{
			name:           "Delete volume by path",
			identifier:     s.vcHelper.Volumes["volume1"].Path,
			identifierType: storage.IDTypePath,
			expectError:    false,
		},
		{
			name:           "Delete volume by CID",
			identifier:     s.vcHelper.Volumes["volume2"].CID,
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
	}

	for _, tc := range testCases {
		err := s.vcHelper.BasicVolController.DeleteVolume(tc.identifier, tc.identifierType)
		if tc.expectError {
			assert.Error(s.T(), err)
		} else {
			assert.NoError(s.T(), err)
		}
	}

	// Verify that the volumes are deleted from the database
	volumes, err := s.vcHelper.BasicVolController.repo.FindAll(context.Background(), s.vcHelper.BasicVolController.repo.GetQuery())
	assert.NoError(s.T(), err)
	assert.Len(s.T(), volumes, 0)
}

func (s *VolumeControllerTestSuite) TestListVolumes() {
	volumes, err := s.vcHelper.BasicVolController.ListVolumes()
	assert.NoError(s.T(), err)

	assert.Len(s.T(), volumes, len(s.vcHelper.Volumes))

	// assert details of returned volumes
	for _, retVol := range volumes {
		// Find the corresponding volume in the test suite's volumes map
		expectedVol, ok := s.vcHelper.Volumes[filepath.Base(retVol.Path)]
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
	size, err := s.vcHelper.BasicVolController.GetSize(s.vcHelper.Volumes["volume1"].Path, storage.IDTypePath)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), int64(0), size)
}
