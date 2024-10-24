// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package s3

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3Manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"gitlab.com/nunet/device-management-service/storage"
	"gitlab.com/nunet/device-management-service/types"
)

type Storage struct {
	*s3.Client
	volController storage.VolumeController
	downloader    *s3Manager.Downloader
	uploader      *s3Manager.Uploader
}

type s3Object struct {
	key   *string
	eTag  *string
	size  int64
	isDir bool
}

// NewClient creates a new S3Storage which includes a S3-SDK client.
// It depends on a VolumeController to manage the volumes being acted upon.
func NewClient(config aws.Config, volController storage.VolumeController) (*Storage, error) {
	ctx, cancel := st.SpanContext(context.Background(), "s3", "new_client_duration", "opentelemetry", "log")
	defer cancel()

	if !hasValidCredentials(config) {
		err := fmt.Errorf("invalid credentials")
		ctx = context.WithValue(ctx, errorKey, err.Error())
		st.Error(ctx, "new_client_invalid_credentials", nil)
		return nil, err
	}

	s3Client := s3.NewFromConfig(config)
	storage := &Storage{
		s3Client,
		volController,
		s3Manager.NewDownloader(s3Client),
		s3Manager.NewUploader(s3Client),
	}

	st.Info(ctx, "new_client_success", nil)
	return storage, nil
}

func (s *Storage) Size(ctx context.Context, source *types.SpecConfig) (uint64, error) {
	ctx, cancel := st.SpanContext(ctx, "s3", "s3_size_duration", "opentelemetry", "log")
	defer cancel()

	inputSource, err := DecodeInputSpec(source)
	if err != nil {
		ctx = context.WithValue(ctx, errorKey, err.Error())
		st.Error(ctx, "s3_size_decode_input_spec_failure", nil)
		return 0, fmt.Errorf("failed to decode input spec: %v", err)
	}

	input := &s3.HeadObjectInput{
		Bucket: aws.String(inputSource.Bucket),
		Key:    aws.String(inputSource.Key),
	}

	output, err := s.HeadObject(ctx, input)
	if err != nil {
		ctx = context.WithValue(ctx, errorKey, err.Error())
		st.Error(ctx, "s3_size_head_object_failure", nil)
		return 0, fmt.Errorf("failed to get object size: %v", err)
	}

	st.Info(ctx, "s3_size_success", nil)
	return uint64(*output.ContentLength), nil
}

// Compile time interface check
// var _ storage.StorageProvider = (*S3Storage)(nil)
