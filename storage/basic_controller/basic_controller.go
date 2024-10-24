// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package basiccontroller

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/storage"
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"
)

// BasicVolumeController is the default implementation of the VolumeController.
// It persists storage volumes information using the StorageVolume.
type BasicVolumeController struct {
	// repo is the repository for storage volume operations
	repo repositories.StorageVolume

	// basePath is the base path where volumes are stored under
	basePath string

	// file system to act upon
	FS afero.Fs
}

// NewDefaultVolumeController returns a new instance of BasicVolumeController
//
// TODO-BugFix [path]: volBasePath might not end with `/`, causing errors when calling methods.
// We need to validate it using the `path` library or just verifying the string.
func NewDefaultVolumeController(repo repositories.StorageVolume, volBasePath string, fs afero.Fs) (*BasicVolumeController, error) {
	ctx, cancel := st.SpanContext(context.Background(), "controller", "volume_controller_init_duration", "opentelemetry", "log")
	defer cancel()

	vc := &BasicVolumeController{
		repo:     repo,
		basePath: volBasePath,
		FS:       fs,
	}

	st.Info(ctx, "volume_controller_init_success", nil)
	return vc, nil
}

// CreateVolume creates a new storage volume given a source (S3, IPFS, job, etc). The
// creation of a storage volume effectively creates an empty directory in the local filesystem
// and writes a record in the database.
//
// The directory name follows the format: `<volSource> + "-" + <name>
// where `name` is random.
//
// TODO-maybe [withName]: allow callers to specify custom name for path
func (vc *BasicVolumeController) CreateVolume(volSource storage.VolumeSource, opts ...storage.CreateVolOpt) (types.StorageVolume, error) {
	ctx, cancel := st.SpanContext(context.Background(), "controller", "volume_create_duration", "opentelemetry", "log")
	defer cancel()

	vol := types.StorageVolume{
		Private:        false,
		ReadOnly:       false,
		EncryptionType: types.EncryptionTypeNull,
	}

	for _, opt := range opts {
		opt(&vol)
	}

	randomStr, err := utils.RandomString(16)
	if err != nil {
		return types.StorageVolume{}, fmt.Errorf("failed to create random string: %w", err)
	}

	vol.Path = vc.basePath + string(volSource) + "-" + randomStr
	ctx = context.WithValue(ctx, pathKey, vol.Path)

	if err := vc.FS.Mkdir(vol.Path, os.ModePerm); err != nil {
		ctx = context.WithValue(ctx, errorKey, err.Error())
		st.Error(ctx, "volume_create_failure", nil)
		return types.StorageVolume{}, fmt.Errorf("failed to create storage volume: %w", err)
	}

	createdVol, err := vc.repo.Create(ctx, vol)
	if err != nil {
		ctx = context.WithValue(ctx, errorKey, err.Error())
		st.Error(ctx, "volume_create_failure", nil)
		return types.StorageVolume{}, fmt.Errorf("failed to create storage volume in repository: %w", err)
	}

	ctx = context.WithValue(ctx, volumeIDKey, createdVol.ID)
	st.Info(ctx, "volume_create_success", nil)
	return createdVol, nil
}

// LockVolume makes the volume read-only, not only changing the field value but also changing file permissions.
// It should be used after all necessary data has been written.
// It optionally can also set the CID and mark the volume as private.
//
// TODO-maybe [CID]: maybe calculate CID of every volume in case WithCID opt is not provided
func (vc *BasicVolumeController) LockVolume(pathToVol string, opts ...storage.LockVolOpt) error {
	ctx, cancel := st.SpanContext(context.Background(), "controller", "volume_lock_duration", "opentelemetry", "log")
	defer cancel()

	ctx = context.WithValue(ctx, pathKey, pathToVol)
	query := vc.repo.GetQuery()
	query.Conditions = append(query.Conditions, repositories.EQ("Path", pathToVol))
	vol, err := vc.repo.Find(ctx, query)
	if err != nil {
		ctx = context.WithValue(ctx, errorKey, err.Error())
		st.Error(ctx, "volume_lock_failure", nil)
		return fmt.Errorf("failed to find storage volume with path %s - Error: %w", pathToVol, err)
	}

	for _, opt := range opts {
		opt(&vol)
	}

	vol.ReadOnly = true
	updatedVol, err := vc.repo.Update(ctx, vol.ID, vol)
	if err != nil {
		ctx = context.WithValue(ctx, errorKey, err.Error())
		st.Error(ctx, "volume_lock_failure", nil)
		return fmt.Errorf("failed to update storage volume with path %s - Error: %w", pathToVol, err)
	}

	// change file permissions
	if err := vc.FS.Chmod(updatedVol.Path, 0o400); err != nil {
		ctx = context.WithValue(ctx, errorKey, err.Error())
		st.Error(ctx, "volume_lock_failure", nil)
		return fmt.Errorf("failed to make storage volume read-only (path: %s): %w", updatedVol.Path, err)
	}

	st.Info(ctx, "volume_lock_success", nil)
	return nil
}

// WithPrivate designates a given volume as private. It can be used both
// when creating or locking a volume.
func WithPrivate[T storage.CreateVolOpt | storage.LockVolOpt]() T {
	return func(v *types.StorageVolume) {
		v.Private = true
	}
}

// WithCID sets the CID of a given volume if already calculated
//
// TODO [validate]: check if CID provided is valid
func WithCID(cid string) storage.LockVolOpt {
	return func(v *types.StorageVolume) {
		v.CID = cid
	}
}

// DeleteVolume deletes a given storage volume record from the database and removes the corresponding directory.
// Identifier is either a CID or a path of a volume.
//
// Note [CID]: if we start to type CID as cid.CID, we may have to use generics here
// as in `[T string | cid.CID]`
func (vc *BasicVolumeController) DeleteVolume(identifier string, idType storage.IDType) error {
	ctx, cancel := st.SpanContext(context.Background(), "controller", "volume_delete_duration", "opentelemetry", "log")
	defer cancel()

	ctx = context.WithValue(ctx, identifierKey, identifier)
	ctx = context.WithValue(ctx, idTypeKey, idType)
	query := vc.repo.GetQuery()

	switch idType {
	case storage.IDTypePath:
		query.Conditions = append(query.Conditions, repositories.EQ("Path", identifier))
	case storage.IDTypeCID:
		query.Conditions = append(query.Conditions, repositories.EQ("CID", identifier))
	default:
		ctx = context.WithValue(ctx, errorKey, "identifier type not supported")
		st.Error(ctx, "volume_delete_failure", nil)
		return fmt.Errorf("identifier type not supported")
	}

	vol, err := vc.repo.Find(ctx, query)
	if err != nil {
		if err == repositories.ErrNotFound {
			ctx = context.WithValue(ctx, errorKey, err.Error())
			st.Error(ctx, "volume_delete_failure", nil)
			return fmt.Errorf("volume not found: %w", err)
		}
		ctx = context.WithValue(ctx, errorKey, err.Error())
		st.Error(ctx, "volume_delete_failure", nil)
		return fmt.Errorf("failed to find volume: %w", err)
	}

	// Remove the directory
	if err := vc.FS.RemoveAll(vol.Path); err != nil {
		return fmt.Errorf("failed to remove volume directory: %w", err)
	}

	// Delete the record from the database
	if err := vc.repo.Delete(context.Background(), vol.ID); err != nil {
		ctx = context.WithValue(ctx, errorKey, err.Error())
		st.Error(ctx, "volume_delete_failure", nil)
		return fmt.Errorf("failed to delete volume: %w", err)
	}

	st.Info(ctx, "volume_delete_success", nil)
	return nil
}

// ListVolumes returns a list of all storage volumes stored on the database
//
// TODO [filter]: maybe add opts to filter results by certain values
func (vc *BasicVolumeController) ListVolumes() ([]types.StorageVolume, error) {
	ctx, cancel := st.SpanContext(context.Background(), "controller", "volume_list_duration", "opentelemetry", "log")
	defer cancel()

	volumes, err := vc.repo.FindAll(ctx, vc.repo.GetQuery())
	if err != nil {
		ctx = context.WithValue(ctx, errorKey, err.Error())
		st.Error(ctx, "volume_list_failure", nil)
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}

	ctx = context.WithValue(ctx, volumeCountKey, len(volumes))
	st.Info(ctx, "volume_list_success", nil)
	return volumes, nil
}

// GetSize returns the size of a volume
// TODO-minor: identify which measurement type will be used
func (vc *BasicVolumeController) GetSize(identifier string, idType storage.IDType) (int64, error) {
	ctx, cancel := st.SpanContext(context.Background(), "controller", "volume_get_size_duration", "opentelemetry", "log")
	defer cancel()

	ctx = context.WithValue(ctx, identifierKey, identifier)
	ctx = context.WithValue(ctx, idTypeKey, idType)
	query := vc.repo.GetQuery()

	switch idType {
	case storage.IDTypePath:
		query.Conditions = append(query.Conditions, repositories.EQ("Path", identifier))
	case storage.IDTypeCID:
		query.Conditions = append(query.Conditions, repositories.EQ("CID", identifier))
	default:
		ctx = context.WithValue(ctx, errorKey, fmt.Sprintf("unsupported ID type: %d", idType))
		st.Error(ctx, "volume_get_size_failure", nil)
		return 0, fmt.Errorf("unsupported ID type: %d", idType)
	}

	vol, err := vc.repo.Find(ctx, query)
	if err != nil {
		ctx = context.WithValue(ctx, errorKey, fmt.Sprintf("failed to find volume: %v", err))
		st.Error(ctx, "volume_get_size_failure", nil)
		return 0, fmt.Errorf("failed to find volume: %w", err)
	}

	size, err := utils.GetDirectorySize(vc.FS, vol.Path)
	if err != nil {
		ctx = context.WithValue(ctx, errorKey, fmt.Sprintf("failed to get directory size: %v", err))
		st.Error(ctx, "volume_get_size_failure", nil)
		return 0, fmt.Errorf("failed to get directory size: %w", err)
	}

	ctx = context.WithValue(ctx, sizeKey, size)
	st.Info(ctx, "volume_get_size_success", nil)
	ctx = context.WithValue(ctx, sizeKey, size)
	st.Info(ctx, "volume_get_size_success", nil)
	return size, nil
}

// EncryptVolume encrypts a given volume
func (vc *BasicVolumeController) EncryptVolume(path string, _ types.Encryptor, _ types.EncryptionType) error {
	ctx, cancel := st.SpanContext(context.Background(), "controller", "volume_encrypt_duration", "opentelemetry", "log")
	defer cancel()

	ctx = context.WithValue(ctx, pathKey, path)
	st.Error(ctx, "volume_encrypt_not_implemented", nil)
	return fmt.Errorf("not implemented")
}

// DecryptVolume decrypts a given volume
func (vc *BasicVolumeController) DecryptVolume(path string, _ types.Decryptor, _ types.EncryptionType) error {
	ctx, cancel := st.SpanContext(context.Background(), "controller", "volume_decrypt_duration", "opentelemetry", "log")
	defer cancel()

	ctx = context.WithValue(ctx, pathKey, path)
	st.Error(ctx, "volume_decrypt_not_implemented", nil)
	return fmt.Errorf("not implemented")
}

var _ storage.VolumeController = (*BasicVolumeController)(nil)
