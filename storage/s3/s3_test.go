//go:build integration || !unit

package s3

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3_types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"

	"gitlab.com/nunet/device-management-service/storage/basic_controller"
	"gitlab.com/nunet/device-management-service/types"
)

/*
IMPORTANT: the following are functional tests which communicate with AWS S3 test bucket.
I considered it was not worth to mock it.

Therefore, the tests here considers the use of environment variables or shared
credentials files (~/.aws/config and etc). These should be set up on our pipeline.

If it's not preferable to run them in any scenario, use `-tag=unit` when running `go test`.

S3Storage implementation relies on a VolumeController, see more on its instantiation func.
*/

const (
	bucketTest         = "test-user-s3-dms-bucket"
	helloObjectKey     = "hello_s3_test.txt"
	helloObjectContent = "hello s3 test"
	basePath           = "/home/.nunet/volumes/"

	object002Key     = "test/test_integration_002.txt"
	object002Content = "testintegration002"

	object003Key     = "test/test_integration_003.txt"
	object003Content = "testintegration003"

	vol1File = "file.txt"
)

type S3ProviderTestSuite struct {
	suite.Suite
	ctx       context.Context
	s3Storage *S3Storage
	vcHelper  *basic_controller.VolControllerTestSuiteHelper

	// map of pathWithinVolume(key):content
	testBuckets map[string]string
}

// SetupTest is mainly setting up a volume controller based on its test suite and a S3 client.
func (s *S3ProviderTestSuite) SetupTest() {

	volumes := map[string]*types.StorageVolume{
		"volume1": {
			Path:           filepath.Join(basePath, "volume1"),
			ReadOnly:       false,
			Private:        false,
			EncryptionType: types.EncryptionTypeNull,
		},
	}

	vcHelper, err := basic_controller.SetupVolControllerTestSuite(s.T(), basePath, volumes)
	s.NoError(err)

	// Write a file in volume1 to be later used to upload
	err = afero.WriteFile(vcHelper.Fs, filepath.Join(vcHelper.Volumes["volume1"].Path, vol1File), []byte("hello world"), 0644)
	s.NoError(err)

	config, err := GetAWSDefaultConfig()
	s.NoError(err)

	s3Client, err := NewClient(config, vcHelper.BasicVolController)
	s.NoError(err)

	s.ctx = context.Background()
	s.s3Storage = s3Client
	s.vcHelper = vcHelper
}

// TestDownload is checking both if the file exists and if the content is equal to the expected one.
// As it's an functional test based on real S3 buckets, if any error occurs, check first if the buckets
// contains the expected files.
func (s *S3ProviderTestSuite) TestDownload() {
	testCases := []struct {
		bucket         string
		key            string
		expectedOutput map[string]string
	}{
		{
			bucket: bucketTest,
			key:    helloObjectKey,
			expectedOutput: map[string]string{
				helloObjectKey: helloObjectContent,
			},
		},
		{
			bucket: bucketTest,
			key:    "*",
			expectedOutput: map[string]string{
				helloObjectKey: helloObjectContent,
				object002Key:   object002Content,
				object003Key:   object003Content,
			},
		},
		{
			bucket: bucketTest,
			key:    "test/*",
			expectedOutput: map[string]string{
				object002Key: object002Content,
				object003Key: object003Content,
			},
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Bucket=%s,Key=%s", tc.bucket, tc.key), func() {
			source := &types.SpecConfig{
				Type: types.StorageProviderS3,
				Params: map[string]interface{}{
					"Bucket": tc.bucket,
					"Key":    tc.key,
				},
			}

			vol, err := s.s3Storage.Download(s.ctx, source)
			s.NoError(err)

			for filePathWithinVolume, expectedContent := range tc.expectedOutput {
				filePath := filepath.Join(vol.Path, filePathWithinVolume)

				content, err := afero.ReadFile(s.vcHelper.Fs, filePath)
				s.NoError(err)

				contentWithoutNewline := strings.ReplaceAll(string(content), "\n", "")
				s.Equal(expectedContent, contentWithoutNewline)
			}
		})
	}
}

// TestUpload checks only if the file exists, it does not check if the content is equal to the expected one.
// Also, we try to delete files before testing in case they already exist within the bucket.
//
// # Tested based on Volume1 set up in SetupTest() with a file written within
//
// TODO: test different scenarios, mainly one with nested directories being uploaded
func (s *S3ProviderTestSuite) TestUpload() {

	// inputKey is the key received by the S3Provider.Upload() method
	inputKey := "upload/"
	// keyUploadedAs is the key resolved based on file path relative to its vol
	keyUploadedAs := filepath.Join(inputKey, vol1File)

	_, err := s.s3Storage.Client.DeleteObject(s.ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucketTest),
		Key:    aws.String(keyUploadedAs),
	})
	if err != nil {
		if s3Err, ok := err.(*s3_types.NoSuchKey); ok && s3Err.ErrorCode() == "NoSuchKey" {
			// File does not exist, ignore the error
		} else {
			s.NoError(err)
		}
	}

	destination := &types.SpecConfig{
		Type: types.StorageProviderS3,
		Params: map[string]interface{}{
			"Bucket": bucketTest,
			"Key":    inputKey,
		},
	}

	err = s.s3Storage.Upload(s.ctx, *s.vcHelper.Volumes["volume1"], destination)
	s.NoError(err)

	// Check if the uploaded file exists in the S3 bucket
	_, err = s.s3Storage.Client.HeadObject(s.ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucketTest),
		Key:    aws.String(keyUploadedAs),
	})
	s.NoError(err)
}

func (s *S3ProviderTestSuite) TestSize() {
	source := &types.SpecConfig{
		Type: types.StorageProviderS3,
		Params: map[string]interface{}{
			"Bucket": bucketTest,
			"Key":    helloObjectKey,
		},
	}

	expectedSize := uint64(14) // 14 Bytes

	size, err := s.s3Storage.Size(s.ctx, source)
	s.NoError(err)
	s.Equal(expectedSize, size)
}

func TestS3ProviderTestSuite(t *testing.T) {
	suite.Run(t, new(S3ProviderTestSuite))
}
