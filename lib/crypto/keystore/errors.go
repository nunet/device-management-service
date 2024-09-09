package keystore

import "errors"

var (
	ErrEmptyPassphrase   = errors.New("passphrase is empty")
	ErrVersionMismatch   = errors.New("version mismatch")
	ErrCipherMismatch    = errors.New("cipher mismatch")
	ErrMACMismatch       = errors.New("mac mismatch")
	ErrEmptyKeysDir      = errors.New("keysDir is empty")
	ErrEmptyNodeIdentity = errors.New("nodeIdentityData is empty")
	ErrKeyNotFound       = errors.New("key not found on this node")
	ErrTokenInvalid      = errors.New("token is invalid")
	ErrNotUnlockedKey    = errors.New("key is not unlocked")
)
