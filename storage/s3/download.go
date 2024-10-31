package s3

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/storage"
	basicController "gitlab.com/nunet/device-management-service/storage/basic_controller"
	"gitlab.com/nunet/device-management-service/types"
)

// Download fetches files from a given S3 bucket. The key may be a directory ending
// with `/` or have a wildcard (`*`) so it handles normal S3 folders but it does
// not handle x-directory.
//
// Warning: the implementation should rely on the FS provided by the volume controller,
// be careful if managing files with `os` (the volume controller might be
// using an in-memory one)
func (s *Storage) Download(ctx context.Context, sourceSpecs *types.SpecConfig) (types.StorageVolume, error) {
	endTrace := observability.StartTrace("s3_download_duration")
	defer endTrace()

	var storageVol types.StorageVolume

	source, err := DecodeInputSpec(sourceSpecs)
	if err != nil {
		log.Errorw("s3_download_failure", "error", err)
		return types.StorageVolume{}, err
	}

	storageVol, err = s.volController.CreateVolume(storage.VolumeSourceS3)
	if err != nil {
		log.Errorw("s3_volume_create_failure", "error", fmt.Errorf("failed to create storage volume: %w", err))
		return types.StorageVolume{}, fmt.Errorf("failed to create storage volume: %v", err)
	}

	resolvedObjects, err := resolveStorageKey(ctx, s.Client, &source)
	if err != nil {
		log.Errorw("s3_resolve_key_failure", "error", fmt.Errorf("failed to resolve storage key: %v", err))
		return types.StorageVolume{}, fmt.Errorf("failed to resolve storage key: %v", err)
	}

	for _, resolvedObject := range resolvedObjects {
		err = s.downloadObject(ctx, &source, resolvedObject, storageVol.Path)
		if err != nil {
			log.Errorw("s3_download_object_failure", "error", fmt.Errorf("failed to download s3 object: %v", err))
			return types.StorageVolume{}, fmt.Errorf("failed to download s3 object: %v", err)
		}
	}

	// after data is filled within the volume, we have to lock it
	err = s.volController.LockVolume(storageVol.Path)
	if err != nil {
		log.Errorw("s3_volume_lock_failure", "error", fmt.Errorf("failed to lock storage volume: %v", err))
		return types.StorageVolume{}, fmt.Errorf("failed to lock storage volume: %v", err)
	}

	log.Infow("s3_download_success", "volumeID", storageVol.ID, "path", storageVol.Path)
	return storageVol, nil
}

func (s *Storage) downloadObject(ctx context.Context, source *InputSource, object s3Object, volPath string) error {
	endTrace := observability.StartTrace("s3_download_object_duration")
	defer endTrace()

	outputPath := filepath.Join(volPath, *object.key)

	// use the same file system instance used by the Volume Controller
	var fs afero.Fs
	if basicVolController, ok := s.volController.(*basicController.BasicVolumeController); ok {
		fs = basicVolController.FS
	}

	err := fs.MkdirAll(outputPath, 0o755)
	if err != nil {
		log.Errorw("s3_create_directory_failure", "path", outputPath, "error", fmt.Errorf("failed to create directory: %v", err))
		return fmt.Errorf("failed to create directory: %v", err)
	}

	if object.isDir {
		// if object is a directory, we don't need to download it (just create the dir)
		return nil
	}

	outputFile, err := fs.OpenFile(outputPath, os.O_RDWR|os.O_CREATE, 0o755)
	if err != nil {
		log.Errorw("s3_open_file_failure", "path", outputPath, "error", err)
		return err
	}
	defer outputFile.Close()

	log.Debugw("Downloading s3 object", "objectKey", *object.key, "outputPath", outputPath)
	_, err = s.downloader.Download(ctx, outputFile, &s3.GetObjectInput{
		Bucket:  aws.String(source.Bucket),
		Key:     object.key,
		IfMatch: object.eTag,
	})
	if err != nil {
		log.Errorw("s3_download_failure", "objectKey", *object.key, "error", fmt.Errorf("failed to download file: %w", err))
		return fmt.Errorf("failed to download file: %w", err)
	}

	log.Infow("s3_download_object_success", "objectKey", *object.key)
	return nil
}

// resolveStorageKey returns a list of s3 objects within a bucket according to the key provided.
func resolveStorageKey(ctx context.Context, client *s3.Client, source *InputSource) ([]s3Object, error) {
	key := source.Key
	if key == "" {
		err := fmt.Errorf("key is required")
		log.Errorw("s3_resolve_key_failure", "error", err)
		return nil, err
	}

	// Check if the key represents a single object
	if !strings.HasSuffix(key, "/") && !strings.Contains(key, "*") {
		return resolveSingleObject(ctx, client, source)
	}

	// key represents multiple objects
	return resolveObjectsWithPrefix(ctx, client, source)
}

func resolveSingleObject(ctx context.Context, client *s3.Client, source *InputSource) ([]s3Object, error) {
	key := sanitizeKey(source.Key)

	headObjectInput := &s3.HeadObjectInput{
		Bucket: aws.String(source.Bucket),
		Key:    aws.String(key),
	}

	headObjectOut, err := client.HeadObject(ctx, headObjectInput)
	if err != nil {
		log.Errorw("s3_head_object_failure", "key", key, "error", err)
		return []s3Object{}, fmt.Errorf("failed to retrieve object metadata: %v", err)
	}

	if strings.HasPrefix(*headObjectOut.ContentType, "application/x-directory") {
		err := fmt.Errorf("x-directory is not yet handled")
		log.Errorw("s3_directory_handling_failure", "key", key, "error", err)
		return []s3Object{}, err
	}

	return []s3Object{
		{
			key:  aws.String(source.Key),
			eTag: headObjectOut.ETag,
			size: *headObjectOut.ContentLength,
		},
	}, nil
}

func resolveObjectsWithPrefix(ctx context.Context, client *s3.Client, source *InputSource) ([]s3Object, error) {
	key := sanitizeKey(source.Key)

	// List objects with the given prefix
	listObjectsInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(source.Bucket),
		Prefix: aws.String(key),
	}
	var objects []s3Object
	paginator := s3.NewListObjectsV2Paginator(client, listObjectsInput)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			log.Errorw("s3_list_objects_failure", "error", fmt.Errorf("failed to list objects: %v", err))
			return nil, fmt.Errorf("failed to list objects: %v", err)
		}

		for _, obj := range page.Contents {
			objects = append(objects, s3Object{
				key:   aws.String(*obj.Key),
				size:  *obj.Size,
				isDir: strings.HasSuffix(*obj.Key, "/"),
			})
		}
	}

	log.Infow("s3_resolve_objects_with_prefix_success", "objectCount", len(objects))
	return objects, nil
}
