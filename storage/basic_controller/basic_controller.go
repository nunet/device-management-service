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
	"os"

	"github.com/spf13/afero"
	"gitlab.com/nunet/device-management-service/db/repositories"
	"gitlab.com/nunet/device-management-service/observability"
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
	endTrace := observability.StartTrace(TraceVolumeControllerInitDuration)
	defer endTrace()

	vc := &BasicVolumeController{
		repo:     repo,
		basePath: volBasePath,
		FS:       fs,
	}

	log.Infow(LogVolumeControllerInitSuccess, LogKeyBasePath, volBasePath)
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
	endTrace := observability.StartTrace(TraceVolumeCreateDuration)
	defer endTrace()

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
		log.Errorw(LogVolumeCreateFailure, LogKeyError, err)
		return types.StorageVolume{}, err
	}

	vol.Path = vc.basePath + string(volSource) + "-" + randomStr

	if err := vc.FS.Mkdir(vol.Path, os.ModePerm); err != nil {
		log.Errorw(LogVolumeCreateFailure, LogKeyPath, vol.Path, LogKeyError, err)
		return types.StorageVolume{}, err
	}

	createdVol, err := vc.repo.Create(context.TODO(), vol)
	if err != nil {
		log.Errorw(LogVolumeCreateFailure, LogKeyPath, vol.Path, LogKeyError, err)
		return types.StorageVolume{}, err
	}

	log.Infow(LogVolumeCreateSuccess, LogKeyVolumeID, createdVol.ID, LogKeyPath, vol.Path)
	return createdVol, nil
}

// LockVolume makes the volume read-only, not only changing the field value but also changing file permissions.
// It should be used after all necessary data has been written.
// It optionally can also set the CID and mark the volume as private.
//
// TODO-maybe [CID]: maybe calculate CID of every volume in case WithCID opt is not provided
func (vc *BasicVolumeController) LockVolume(pathToVol string, opts ...storage.LockVolOpt) error {
	endTrace := observability.StartTrace(TraceVolumeLockDuration)
	defer endTrace()

	query := vc.repo.GetQuery()
	query.Conditions = append(query.Conditions, repositories.EQ("Path", pathToVol))
	vol, err := vc.repo.Find(context.TODO(), query)
	if err != nil {
		log.Errorw(LogVolumeLockFailure, LogKeyPath, pathToVol, LogKeyError, err)
		return err
	}

	for _, opt := range opts {
		opt(&vol)
	}

	vol.ReadOnly = true
	updatedVol, err := vc.repo.Update(context.TODO(), vol.ID, vol)
	if err != nil {
		log.Errorw(LogVolumeLockFailure, LogKeyVolumeID, vol.ID, LogKeyError, err)
		return err
	}

	// Change file permissions to read-only
	if err := vc.FS.Chmod(updatedVol.Path, 0o400); err != nil {
		log.Errorw(LogVolumeLockFailure, LogKeyPath, updatedVol.Path, LogKeyError, err)
		return err
	}

	log.Infow(LogVolumeLockSuccess, LogKeyVolumeID, updatedVol.ID, LogKeyPath, updatedVol.Path)
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
	endTrace := observability.StartTrace(TraceVolumeDeleteDuration)
	defer endTrace()

	query := vc.repo.GetQuery()

	switch idType {
	case storage.IDTypePath:
		query.Conditions = append(query.Conditions, repositories.EQ("Path", identifier))
	case storage.IDTypeCID:
		query.Conditions = append(query.Conditions, repositories.EQ("CID", identifier))
	default:
		log.Errorw(LogVolumeDeleteFailure, LogKeyIdentifier, identifier, LogKeyError, ErrMsgInvalidIdentifier)
		return ErrInvalidIdentifier
	}

	vol, err := vc.repo.Find(context.TODO(), query)
	if err != nil {
		if err == repositories.ErrNotFound {
			log.Errorw(LogVolumeDeleteFailure, LogKeyIdentifier, identifier, LogKeyError, ErrMsgVolumeNotFound)
			return repositories.ErrNotFound
		}
		log.Errorw(LogVolumeDeleteFailure, LogKeyIdentifier, identifier, LogKeyError, err)
		return err
	}

	// Remove the directory
	if err := vc.FS.RemoveAll(vol.Path); err != nil {
		log.Errorw(LogVolumeDeleteFailure, LogKeyPath, vol.Path, LogKeyError, err)
		return err
	}

	if err := vc.repo.Delete(context.TODO(), vol.ID); err != nil {
		log.Errorw(LogVolumeDeleteFailure, LogKeyVolumeID, vol.ID, LogKeyError, err)
		return err
	}

	log.Infow(LogVolumeDeleteSuccess, LogKeyVolumeID, vol.ID, LogKeyPath, vol.Path)
	return nil
}

// ListVolumes returns a list of all storage volumes stored on the database
//
// TODO [filter]: maybe add opts to filter results by certain values
func (vc *BasicVolumeController) ListVolumes() ([]types.StorageVolume, error) {
	endTrace := observability.StartTrace(TraceVolumeListDuration)
	defer endTrace()

	volumes, err := vc.repo.FindAll(context.TODO(), vc.repo.GetQuery())
	if err != nil {
		log.Errorw(LogVolumeListFailure, LogKeyError, err)
		return nil, err
	}

	log.Infow(LogVolumeListSuccess, LogKeyVolumeCount, len(volumes))
	return volumes, nil
}

// GetSize returns the size of a volume
// TODO-minor: identify which measurement type will be used
func (vc *BasicVolumeController) GetSize(identifier string, idType storage.IDType) (int64, error) {
	endTrace := observability.StartTrace(TraceVolumeGetSizeDuration)
	defer endTrace()

	query := vc.repo.GetQuery()

	switch idType {
	case storage.IDTypePath:
		query.Conditions = append(query.Conditions, repositories.EQ("Path", identifier))
	case storage.IDTypeCID:
		query.Conditions = append(query.Conditions, repositories.EQ("CID", identifier))
	default:
		log.Errorw(LogVolumeGetSizeFailure, LogKeyIdentifier, identifier, LogKeyError, ErrMsgInvalidIdentifier)
		return 0, ErrInvalidIdentifier
	}

	vol, err := vc.repo.Find(context.TODO(), query)
	if err != nil {
		log.Errorw(LogVolumeGetSizeFailure, LogKeyIdentifier, identifier, LogKeyError, err)
		return 0, err
	}

	size, err := utils.GetDirectorySize(vc.FS, vol.Path)
	if err != nil {
		log.Errorw(LogVolumeGetSizeFailure, LogKeyPath, vol.Path, LogKeyError, err)
		return 0, err
	}

	log.Infow(LogVolumeGetSizeSuccess, LogKeyVolumeID, vol.ID, LogKeySize, size)
	return size, nil
}

// EncryptVolume encrypts a given volume
func (vc *BasicVolumeController) EncryptVolume(path string, _ types.Encryptor, _ types.EncryptionType) error {
	endTrace := observability.StartTrace(TraceVolumeEncryptDuration)
	defer endTrace()

	log.Errorw(LogVolumeEncryptNotImplemented, LogKeyPath, path)
	return ErrNotImplemented
}

// DecryptVolume decrypts a given volume
func (vc *BasicVolumeController) DecryptVolume(path string, _ types.Decryptor, _ types.EncryptionType) error {
	endTrace := observability.StartTrace(TraceVolumeDecryptDuration)
	defer endTrace()

	log.Errorw(LogVolumeDecryptNotImplemented, LogKeyPath, path)
	return ErrNotImplemented
}

var _ storage.VolumeController = (*BasicVolumeController)(nil)
