package did

import (
	"strings"
)

type DID struct {
	URI string `json:"uri,omitempty"`
}

func (did DID) Equal(other DID) bool {
	return did.URI == other.URI
}

func (did DID) Empty() bool {
	return did.URI == ""
}

func (did DID) String() string {
	return did.URI
}

func (did DID) Method() string {
	parts := strings.Split(did.URI, ":")
	if len(parts) == 3 {
		return parts[1]
	}

	return ""
}

func (did DID) Identifier() string {
	parts := strings.Split(did.URI, ":")
	if len(parts) == 3 {
		return parts[2]
	}

	return ""
}

func FromString(s string) (DID, error) {
	if s != "" {
		parts := strings.Split(s, ":")
		if len(parts) == 2 {
			return DID{}, ErrInvalidDID
		}

		for _, part := range parts {
			if part == "" {
				return DID{}, ErrInvalidDID
			}
		}

		// TODO validate parts according to spec: https://www.w3.org/TR/did-core/
	}

	return DID{URI: s}, nil
}
