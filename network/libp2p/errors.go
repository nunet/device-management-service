package libp2p

import (
	"errors"
)

var (
	ErrNotSubscribed      = errors.New("not subscribed to topic")
	ErrStreamRegistered   = errors.New("stream with this protocol is already registered")
	ErrHostNotInitialized = errors.New("host is not initialized")
)
