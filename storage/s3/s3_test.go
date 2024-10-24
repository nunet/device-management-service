// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build integration || !unit

package s3

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"

	basicController "gitlab.com/nunet/device-management-service/storage/basic_controller"
	"gitlab.com/nunet/device-management-service/telemetry"
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"
)

/*
IMPORTANT: the following are functional tests which communicate with AWS S3 test bucket.
I considered it was not worth to mock it.

Therefore, the tests here considers the use of environment variables or shared
credentials files (~/.aws/config and etc). These should be set up on our pipeline.

(If failed to get credentials, the tests will be skipped)

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
	vol1Data = "hello world"
)

type S3ProviderTestSuite struct {
	suite.Suite
	ctx       context.Context
	s3Storage *Storage
	vcHelper  *basicController.VolumeControllerTestKit
}

// SetupTest is mainly setting up a volume controller based on its test suite and a S3 client.
func (s *S3ProviderTestSuite) SetupTest() {
	s.ctx = context.Background()
	// Initialize telemetry in test mode, replacing the global st
	st = telemetry.NewTelemetry(nil, nil, true)

	volumes := map[string]*types.StorageVolume{
		"volume1": {
			Path:           filepath.Join(basePath, "volume1"),
			ReadOnly:       false,
			Private:        false,
			EncryptionType: types.EncryptionTypeNull,
		},
	}

	vcHelper, err := basicController.SetupVolumeControllerTestKit(basePath, volumes)
	s.NoError(err)

	// Write a file in volume1 to be later used to upload
	err = afero.WriteFile(vcHelper.Fs, filepath.Join(vcHelper.Volumes["volume1"].Path, vol1File), []byte(vol1Data), 0o644)
	s.NoError(err)

	config, err := GetAWSDefaultConfig()
	s.NoError(err)

	s3Client, err := NewClient(config, vcHelper.BasicVolController)
	s.NoError(err)

	// Make sure all uploaded files by the tests are deleted after one day.
	_, err = s3Client.PutBucketLifecycleConfiguration(
		s.ctx,
		&s3.PutBucketLifecycleConfigurationInput{
			Bucket: aws.String(bucketTest),
			LifecycleConfiguration: &s3Types.BucketLifecycleConfiguration{
				Rules: []s3Types.LifecycleRule{
					{
						ID:     aws.String("Delete old test objects"),
						Status: s3Types.ExpirationStatusEnabled,
						Filter: &s3Types.LifecycleRuleFilterMemberPrefix{
							Value: "upload-test-",
						},
						Expiration: &s3Types.LifecycleExpiration{
							Days: aws.Int32(1),
						},
					},
				},
			},
		},
	)
	s.NoError(err)

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

// TestUpload checks if the file exists and its content is equal to the expected one.
// It uses a random key for upload and deletes the object at the end of the test.
//
// Important: Tested based on Volume1 set up in SetupTest() with a file written within
func (s *S3ProviderTestSuite) TestUpload() {
	key, err := utils.RandomString(8)
	s.NoError(err)

	randomKey := fmt.Sprintf("upload-test-%s/", key)
	keyUploadedAs := filepath.Join(randomKey, vol1File)

	destination := &types.SpecConfig{
		Type: types.StorageProviderS3,
		Params: map[string]interface{}{
			"Bucket": bucketTest,
			"Key":    randomKey,
		},
	}

	err = s.s3Storage.Upload(s.ctx, *s.vcHelper.Volumes["volume1"], destination)
	s.NoError(err)

	defer func() {
		_, err := s.s3Storage.Client.DeleteObject(s.ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(bucketTest),
			Key:    aws.String(keyUploadedAs),
		})
		s.NoError(err)
	}()

	// Check if the uploaded file exists and has the correct content
	getObjectOutput, err := s.s3Storage.Client.GetObject(s.ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketTest),
		Key:    aws.String(keyUploadedAs),
	})
	s.NoError(err)
	defer getObjectOutput.Body.Close()

	content, err := io.ReadAll(getObjectOutput.Body)
	s.NoError(err)
	s.Equal(vol1Data, string(content))
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
	isPipeline, _ := strconv.ParseBool(os.Getenv("GITLAB_CI"))
	errMsg := "Error getting S3 credentials for S3 integration tests"

	config, err := GetAWSDefaultConfig()
	if err != nil {
		if isPipeline {
			t.Fatal(errMsg)
		} else {
			t.Skip(errMsg)
		}
	}

	if valid := hasValidCredentials(config); !valid {
		if isPipeline {
			t.Fatal(errMsg)
		} else {
			t.Skip(errMsg)
		}
	}

	suite.Run(t, new(S3ProviderTestSuite))
}
