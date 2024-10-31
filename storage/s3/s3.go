// s3/client.go
package s3

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3Manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"gitlab.com/nunet/device-management-service/observability"
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
	endTrace := observability.StartTrace("new_client_duration")
	defer endTrace()

	if !hasValidCredentials(config) {
		err := fmt.Errorf("invalid credentials")
		log.Errorw("new_client_invalid_credentials", "error", err)
		return nil, err
	}

	s3Client := s3.NewFromConfig(config)
	storage := &Storage{
		s3Client,
		volController,
		s3Manager.NewDownloader(s3Client),
		s3Manager.NewUploader(s3Client),
	}

	log.Infow("new_client_success")
	return storage, nil
}

// Size calculates the size of a given object in S3.
func (s *Storage) Size(ctx context.Context, source *types.SpecConfig) (uint64, error) {
	endTrace := observability.StartTrace("s3_size_duration")
	defer endTrace()

	inputSource, err := DecodeInputSpec(source)
	if err != nil {
		log.Errorw("s3_size_decode_input_spec_failure", "error", err)
		return 0, fmt.Errorf("failed to decode input spec: %v", err)
	}

	input := &s3.HeadObjectInput{
		Bucket: aws.String(inputSource.Bucket),
		Key:    aws.String(inputSource.Key),
	}

	output, err := s.HeadObject(ctx, input)
	if err != nil {
		log.Errorw("s3_size_head_object_failure", "error", err)
		return 0, fmt.Errorf("failed to get object size: %v", err)
	}

	log.Infow("s3_size_success", "bucket", inputSource.Bucket, "key", inputSource.Key, "size", *output.ContentLength)
	return uint64(*output.ContentLength), nil
}

// Compile time interface check
// var _ storage.StorageProvider = (*S3Storage)(nil)
