package s3

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/afero"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	basiccontroller "gitlab.com/nunet/device-management-service/storage/basic_controller"
	"gitlab.com/nunet/device-management-service/types"
)

// Upload uploads all files (recursively) from a local volume to an S3 bucket.
// It handles directories.
//
// Warning: the implementation should rely on the FS provided by the volume controller,
// be careful if managing files with `os` (the volume controller might be
// using an in-memory one)
func (s *Storage) Upload(ctx context.Context, vol types.StorageVolume, destinationSpecs *types.SpecConfig) error {
	target, err := DecodeInputSpec(destinationSpecs)
	if err != nil {
		return fmt.Errorf("failed to decode input spec: %v", err)
	}

	sanitizedKey := sanitizeKey(target.Key)

	// set file system to act upon based on the volume controller implementation
	var fs afero.Fs
	if basicVolController, ok := s.volController.(*basiccontroller.BasicVolumeController); ok {
		fs = basicVolController.FS
	}

	zlog.Sugar().Debugf("Uploading files from %s to s3://%s/%s", vol.Path, target.Bucket, sanitizedKey)
	err = afero.Walk(fs, vol.Path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(vol.Path, filePath)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %v", err)
		}

		// Construct the S3 key by joining the sanitized key and the relative path
		s3Key := filepath.Join(sanitizedKey, relPath)

		file, err := fs.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open file: %v", err)
		}
		defer file.Close()

		zlog.Sugar().Debugf("Uploading %s to s3://%s/%s", filePath, target.Bucket, s3Key)
		_, err = s.uploader.Upload(ctx, &s3.PutObjectInput{
			Bucket: aws.String(target.Bucket),
			Key:    aws.String(s3Key),
			Body:   file,
		})
		if err != nil {
			return fmt.Errorf("failed to upload file to S3: %v", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("upload failed. It's possible that some files were uploaded; Error: %v", err)
	}

	return nil
}
