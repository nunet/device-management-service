package actor

import (
	"errors"
)

var (
	ErrInvalidMessage         = errors.New("invalid message")
	ErrMissingOption          = errors.New("missing option")
	ErrUnsupportedKeyType     = errors.New("unsupported key type")
	ErrSignatureVerification  = errors.New("signature verification failed")
	ErrInvalidSecurityContext = errors.New("invalid security context")
	ErrMessageExpired         = errors.New("message expired")

	ErrTODO = errors.New("TODO")
)
