package types

const (
	StorageVolumeTypeBind = "bind"
)

// StorageVolumeExecutor represents a prepared storage volume that can be mounted to an execution
type StorageVolumeExecutor struct {
	// Type of the volume (e.g. bind)
	Type string `json:"type"`
	// Source path of the volume on the host
	Source string `json:"source"`
	// Target path of the volume in the execution
	Target string `json:"target"`
	// ReadOnly flag to mount the volume as read-only
	ReadOnly bool `json:"readonly"`
}

// StorageVolume contains the location (FS path) of a directory where certain data may be stored
// and metadata about the volume itself + data (if any).
//
// Be aware that StorageVolume might not contain any data within it, specially when it's just
// created using the VolumeController, there will be no data in it by default.
//
// TODO-maybe [job]: should volumes be related to a job or []job?
//
// TODO-maybe: most of the fields should be private, callers shouldn't be able to altere
// the state of the volume except when using certain methods
type StorageVolume struct {
	BaseDBModel
	// CID is the content identifier of the storage volume.
	//
	// Warning: CID must be updated ONLY when locking volume (aka when volume was
	// is set to read-only)
	//
	// Be aware: Before relying on data's CID, be aware that it might be encrypted (
	// EncryptionType might be checked first if needed)
	//
	// TODO-maybe [type]: maybe it should be of type cid.CID
	CID string

	// Path points to the root of a DIRECTORY where data may be stored.
	Path string

	// ReadOnly indicates whether the storage volume is read-only or not.
	ReadOnly bool

	// Size is the size of the storage volume
	//
	// TODO-maybe: if relying on a size field, we would have to be more careful
	// how state is being mutated (and they shouldn't be by the caller)
	//  - Size might be calculated only when locking the volume also
	//
	// Size int64

	// Private indicates whether the storage volume is private or not.
	// If it's private, it shouldn't be shared with other nodes and it shouldn't
	// be persisted after the job is finished.
	// Practical application: if private, peer maintaining it shouldn't publish
	// its CID as if it was available to be worked on by other jobs.
	Private bool

	// EncryptionType indicates the type of encryption used for the storage volume.
	// In case no encryption is used, the value will be EncryptionTypeNull
	EncryptionType EncryptionType
}
