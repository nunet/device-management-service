package s3

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3Manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/storage"
)

type S3Storage struct {
	*s3.Client
	volController storage.VolumeController
	downloader    *s3Manager.Downloader
	uploader      *s3Manager.Uploader
}

type s3Object struct {
	key       *string
	eTag      *string
	versionID *string
	size      int64
	isDir     bool
}

// NewClient creates a new S3Storage which includes a S3-SDK client.
// It depends on a VolumeController to manage the volumes being acted upon.
func NewClient(config aws.Config, volController storage.VolumeController) (*S3Storage, error) {
	if !hasValidCredentials(config) {
		return nil, fmt.Errorf("invalid credentials")
	}

	s3Client := s3.NewFromConfig(config)
	return &S3Storage{
		s3Client,
		volController,
		s3Manager.NewDownloader(s3Client),
		s3Manager.NewUploader(s3Client),
	}, nil
}

func (s *S3Storage) Size(ctx context.Context, source *types.SpecConfig) (uint64, error) {
	inputSource, err := DecodeInputSpec(source)
	if err != nil {
		return 0, fmt.Errorf("failed to decode input spec: %v", err)
	}

	input := &s3.HeadObjectInput{
		Bucket: aws.String(inputSource.Bucket),
		Key:    aws.String(inputSource.Key),
	}

	output, err := s.HeadObject(ctx, input)
	if err != nil {
		return 0, fmt.Errorf("failed to get object size: %v", err)
	}

	return uint64(*output.ContentLength), nil
}

// Compile time interface check
// var _ storage.StorageProvider = (*S3Storage)(nil)
