package basiccontroller

import "errors"

const (
	// Log messages
	LogVolumeControllerInitSuccess = "volume_controller_init_success"
	LogVolumeCreateSuccess         = "volume_create_success"
	LogVolumeCreateFailure         = "volume_create_failure"
	LogVolumeLockSuccess           = "volume_lock_success"
	LogVolumeLockFailure           = "volume_lock_failure"
	LogVolumeDeleteSuccess         = "volume_delete_success"
	LogVolumeDeleteFailure         = "volume_delete_failure"
	LogVolumeListSuccess           = "volume_list_success"
	LogVolumeListFailure           = "volume_list_failure"
	LogVolumeGetSizeSuccess        = "volume_get_size_success"
	LogVolumeGetSizeFailure        = "volume_get_size_failure"
	LogVolumeEncryptNotImplemented = "volume_encrypt_not_implemented"
	LogVolumeDecryptNotImplemented = "volume_decrypt_not_implemented"

	// Trace names
	TraceVolumeControllerInitDuration = "volume_controller_init_duration"
	TraceVolumeCreateDuration         = "volume_create_duration"
	TraceVolumeLockDuration           = "volume_lock_duration"
	TraceVolumeDeleteDuration         = "volume_delete_duration"
	TraceVolumeListDuration           = "volume_list_duration"
	TraceVolumeGetSizeDuration        = "volume_get_size_duration"
	TraceVolumeEncryptDuration        = "volume_encrypt_duration"
	TraceVolumeDecryptDuration        = "volume_decrypt_duration"

	// Error messages
	ErrMsgVolumeCreateFailure         = "failed to create storage volume"
	ErrMsgVolumeCreateRepoFailure     = "failed to create storage volume in repository"
	ErrMsgVolumeLockRepoFailure       = "failed to update storage volume"
	ErrMsgVolumeDeleteFailure         = "failed to delete volume"
	ErrMsgVolumeFindFailure           = "failed to find storage volume"
	ErrMsgVolumeNotFound              = "volume not found"
	ErrMsgVolumePathNotSupported      = "identifier type not supported"
	ErrMsgVolumeSizeFailure           = "failed to get directory size"
	ErrMsgVolumeEncryptNotImplemented = "volume encryption not implemented"
	ErrMsgVolumeDecryptNotImplemented = "volume decryption not implemented"
	ErrMsgInvalidIdentifier           = "invalid identifier"
	ErrMsgNotImplemented              = "not implemented"

	// Other keys
	LogKeyBasePath    = "basePath"
	LogKeyVolumeID    = "volumeID"
	LogKeyPath        = "path"
	LogKeyError       = "error"
	LogKeyVolumeCount = "volumeCount"
	LogKeyIdentifier  = "identifier"
	LogKeySize        = "size"
)

var (
	// Error messages
	ErrNotImplemented    = errors.New("not implemented")
	ErrInvalidIdentifier = errors.New("invalid identifier")
)
