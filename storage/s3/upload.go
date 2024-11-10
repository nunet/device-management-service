// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

// s3/upload.go
package s3

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/observability"
	basicController "gitlab.com/nunet/device-management-service/storage/basic_controller"
	"gitlab.com/nunet/device-management-service/types"
)

// Upload uploads all files (recursively) from a local volume to an S3 bucket.
// It handles directories.
//
// Warning: the implementation should rely on the FS provided by the volume controller,
// be careful if managing files with `os` (the volume controller might be
// using an in-memory one)
func (s *Storage) Upload(ctx context.Context, vol types.StorageVolume, destinationSpecs *types.SpecConfig) error {
	endTrace := observability.StartTrace("s3_upload_duration")
	defer endTrace()

	target, err := DecodeInputSpec(destinationSpecs)
	if err != nil {
		log.Errorw("s3_upload_decode_spec_failure", "error", err)
		return fmt.Errorf("failed to decode input spec: %v", err)
	}

	sanitizedKey := sanitizeKey(target.Key)

	// set file system to act upon based on the volume controller implementation
	var fs afero.Fs
	if basicVolController, ok := s.volController.(*basicController.BasicVolumeController); ok {
		fs = basicVolController.FS
	}

	log.Debugw("Uploading files", "sourcePath", vol.Path, "bucket", target.Bucket, "key", sanitizedKey)
	err = afero.Walk(fs, vol.Path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			log.Errorw("s3_upload_walk_failure", "error", err)
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(vol.Path, filePath)
		if err != nil {
			log.Errorw("s3_upload_relative_path_failure", "error", err)
			return fmt.Errorf("failed to get relative path: %v", err)
		}

		// Construct the S3 key by joining the sanitized key and the relative path
		s3Key := filepath.Join(sanitizedKey, relPath)

		file, err := fs.Open(filePath)
		if err != nil {
			log.Errorw("s3_upload_open_file_failure", "filePath", filePath, "error", err)
			return fmt.Errorf("failed to open file: %v", err)
		}
		defer file.Close()

		log.Debugw("Uploading file to S3", "filePath", filePath, "bucket", target.Bucket, "key", s3Key)
		_, err = s.uploader.Upload(ctx, &s3.PutObjectInput{
			Bucket: aws.String(target.Bucket),
			Key:    aws.String(s3Key),
			Body:   file,
		})
		if err != nil {
			log.Errorw("s3_upload_put_object_failure", "filePath", filePath, "error", err)
			return fmt.Errorf("failed to upload file to S3: %v", err)
		}

		log.Infow("s3_upload_file_success", "filePath", filePath, "bucket", target.Bucket, "key", s3Key)
		return nil
	})
	if err != nil {
		log.Errorw("s3_upload_failure", "error", err)
		return fmt.Errorf("upload failed. It's possible that some files were uploaded; Error: %v", err)
	}

	log.Infow("s3_upload_success", "sourcePath", vol.Path, "bucket", target.Bucket)
	return nil
}
