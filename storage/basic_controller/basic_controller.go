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
	return &BasicVolumeController{
		repo:     repo,
		basePath: volBasePath,
		FS:       fs,
	}, nil
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
	vol := types.StorageVolume{
		Private:        false,
		ReadOnly:       false,
		EncryptionType: types.EncryptionTypeNull,
	}

	for _, opt := range opts {
		opt(&vol)
	}

	vol.Path = vc.basePath + string(volSource) + "-" + utils.RandomString(16)
	if err := vc.FS.Mkdir(vol.Path, os.ModePerm); err != nil {
		return types.StorageVolume{}, fmt.Errorf("failed to create storage volume: %w", err)
	}

	createdVol, err := vc.repo.Create(context.Background(), vol)
	if err != nil {
		return types.StorageVolume{}, fmt.Errorf("failed to create storage volume in repository: %w", err)
	}

	return createdVol, nil
}

// LockVolume makes the volume read-only, not only changing the field value but also changing file permissions.
// It should be used after all necessary data has been written.
// It optionally can also set the CID and mark the volume as private.
//
// TODO-maybe [CID]: maybe calculate CID of every volume in case WithCID opt is not provided
func (vc *BasicVolumeController) LockVolume(pathToVol string, opts ...storage.LockVolOpt) error {
	query := vc.repo.GetQuery()
	query.Conditions = append(query.Conditions, repositories.EQ("Path", pathToVol))
	vol, err := vc.repo.Find(context.Background(), query)
	if err != nil {
		return fmt.Errorf("failed to find storage volume with path %s - Error: %w", pathToVol, err)
	}

	for _, opt := range opts {
		opt(&vol)
	}

	// update records
	vol.ReadOnly = true
	updatedVol, err := vc.repo.Update(context.Background(), vol.ID, vol)
	if err != nil {
		return fmt.Errorf("failed to update storage volume with path %s - Error: %w", pathToVol, err)
	}

	// change file permissions
	// nolint:gofumpt
	if err := vc.FS.Chmod(updatedVol.Path, 0o400); err != nil {
		return fmt.Errorf("failed to make storage volume read-only (path: %s): %w", updatedVol.Path, err)
	}

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

// DeleteVolume deletes a given storage volume record from the database.
// Identifier is either a CID or a path of a volume. Therefore, records for both
// will be deleted.
//
// Note [CID]: if we start to type CID as cid.CID, we may have to use generics here
// as in `[T string | cid.CID]`
func (vc *BasicVolumeController) DeleteVolume(identifier string, idType storage.IDType) error {
	query := vc.repo.GetQuery()

	switch idType {
	case storage.IDTypePath:
		query.Conditions = append(query.Conditions, repositories.EQ("Path", identifier))
	case storage.IDTypeCID:
		query.Conditions = append(query.Conditions, repositories.EQ("CID", identifier))
	default:
		return fmt.Errorf("identifier type not supported")
	}

	vol, err := vc.repo.Find(context.Background(), query)
	if err != nil {
		if err == repositories.ErrNotFound {
			return fmt.Errorf("volume not found: %w", err)
		}
		return fmt.Errorf("failed to find volume: %w", err)
	}

	err = vc.repo.Delete(context.Background(), vol.ID)
	if err != nil {
		return fmt.Errorf("failed to delete volume: %w", err)
	}

	return nil
}

// ListVolumes returns a list of all storage volumes stored on the database
//
// TODO [filter]: maybe add opts to filter results by certain values
func (vc *BasicVolumeController) ListVolumes() ([]types.StorageVolume, error) {
	volumes, err := vc.repo.FindAll(context.Background(), vc.repo.GetQuery())
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}
	return volumes, nil
}

// GetSize returns the size of a volume
// TODO-minor: identify which measurement type will be used
func (vc *BasicVolumeController) GetSize(identifier string, idType storage.IDType) (int64, error) {
	query := vc.repo.GetQuery()

	switch idType {
	case storage.IDTypePath:
		query.Conditions = append(query.Conditions, repositories.EQ("Path", identifier))
	case storage.IDTypeCID:
		query.Conditions = append(query.Conditions, repositories.EQ("CID", identifier))
	default:
		return 0, fmt.Errorf("unsupported ID type: %d", idType)
	}

	vol, err := vc.repo.Find(context.Background(), query)
	if err != nil {
		return 0, fmt.Errorf("failed to find volume: %w", err)
	}

	size, err := utils.GetDirectorySize(vc.FS, vol.Path)
	if err != nil {
		return 0, fmt.Errorf("failed to get directory size: %w", err)
	}

	return size, nil
}

// EncryptVolume encrypts a given volume
func (vc *BasicVolumeController) EncryptVolume(_ string, _ types.Encryptor, _ types.EncryptionType) error {
	return fmt.Errorf("not implemented")
}

// DecryptVolume decrypts a given volume
func (vc *BasicVolumeController) DecryptVolume(_ string, _ types.Decryptor, _ types.EncryptionType) error {
	return fmt.Errorf("not implemented")
}

// TODO-minor: compiler-time check for interface implementation
var _ storage.VolumeController = (*BasicVolumeController)(nil)
