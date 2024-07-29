// Package storage handles mainly storage access to remote storage providers such as
// AWS S3 and IPFS. Also, it handles the control of storage volumes.
package storage

import (
	"context"
	"gitlab.com/nunet/device-management-service/models"
)

// StorageProvider handles I/O operations of files with remote storage providers
// such as AWS S3 and IPFS.
//
// Its functionality is coupled with local mounted volumes, meaning that implementations
// will rely on mounted files to upload data and downloading data will result in a
// local mounted volume.
//
// - When needed, the availability-checking (e.g.: check if IPFS node is running) of some
// storage provider is handled when instantiating the implementation
//
// - Any necessary authentication data is also provided within the `*models.SpecConfig`
// parameters
//
// - Although it may be feasible to implement, the interface was not built with
// the idea of supporting streaming of data and non-file storage operations (e.g.:
// some databases)
type StorageProvider interface {
	// Upload uploads a storage volume data to a given remote storage provider.
	// The operation results in a return value that might vary from provider to provider
	// (and it may not exist in some cases).
	Upload(ctx context.Context, vol models.StorageVolume, target *models.SpecConfig) (*models.SpecConfig, error)

	// Download downloads data from a given source, mounting it to a certain local path
	// which is defined by the VolumeController being used.
	Download(ctx context.Context, source *models.SpecConfig) (models.StorageVolume, error)

	// Size returns the size in bytes of a given source. The method may also be useful to check
	// if a given source is available.
	Size(ctx context.Context, source *models.SpecConfig) (uint64, error)
}
