package crypto

import (
	"bytes"
	"encoding/base32"
	"encoding/json"
	"fmt"
)

// ID is the encoding of the actor's public key
type ID struct{ PublicKey []byte }

func (id ID) Equal(other ID) bool {
	return bytes.Equal(id.PublicKey, other.PublicKey)
}

// IDJsonView is the on the wire json reprsentation of an ID
type IDJSONView struct {
	Pub string `json:"pub"`
}

func (id ID) String() string {
	return base32.StdEncoding.EncodeToString(id.PublicKey)
}

func IDFromString(s string) (ID, error) {
	data, err := base32.StdEncoding.DecodeString(s)
	if err != nil {
		return ID{}, fmt.Errorf("decode ID: %w", err)
	}

	return ID{PublicKey: data}, nil
}

func (id ID) MarshalJSON() ([]byte, error) {
	return json.Marshal(IDJSONView{Pub: id.String()})
}

var _ json.Marshaler = ID{}

func (id *ID) UnmarshalJSON(data []byte) error {
	var input IDJSONView
	err := json.Unmarshal(data, &input)
	if err != nil {
		return fmt.Errorf("unmarshaling ID: %w", err)
	}

	val, err := IDFromString(input.Pub)
	if err != nil {
		return fmt.Errorf("unmarshaling ID: %w", err)
	}

	*id = val
	return nil
}

var _ json.Unmarshaler = (*ID)(nil)
