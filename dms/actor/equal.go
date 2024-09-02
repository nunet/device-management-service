package actor

import (
	"bytes"
)

func (id ID) Equals(other ID) bool {
	return bytes.Equal(id.PublicKey, other.PublicKey)
}

func (did DID) Equals(other DID) bool {
	return bytes.Equal(did.PublicKey, other.PublicKey)
}
