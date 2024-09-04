package actor

import (
	"errors"
)

var (
	ErrInvalidMessage         = errors.New("invalid message")
	ErrMissingOption          = errors.New("missing option")
	ErrSignatureVerification  = errors.New("signature verification failed")
	ErrInvalidSecurityContext = errors.New("invalid security context")
	ErrMessageExpired         = errors.New("message expired")
	ErrBadSender              = errors.New("bad sender")

	ErrTODO = errors.New("TODO")
)
