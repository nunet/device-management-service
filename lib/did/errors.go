package did

import (
	"errors"
)

var (
	ErrInvalidDID       = errors.New("invalid DID")
	ErrInvalidKeyType   = errors.New("invalid key type")
	ErrInvalidSignature = errors.New("signature verification failed")
	ErrNoProvider       = errors.New("no provider")
	ErrNoAnchorMethod   = errors.New("no anchor method")
	ErrHardwareKey      = errors.New("hardware key")

	ErrTODO = errors.New("TODO")
)
