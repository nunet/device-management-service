package actor

import (
	"encoding/base32"
	"encoding/json"
	"fmt"
)

type IDJSONView struct {
	ID string
}

type DIDJSONView struct {
	DID string
}

func (id ID) String() string {
	return base32.StdEncoding.EncodeToString(id.PublicKey)
}

func (id ID) MarshalJSON() ([]byte, error) {
	return json.Marshal(IDJSONView{ID: id.String()})
}

var _ json.Marshaler = ID{}

func IDFromString(s string) (ID, error) {
	data, err := base32.StdEncoding.DecodeString(s)
	if err != nil {
		return ID{}, fmt.Errorf("decode ID: %w", err)
	}

	return ID{PublicKey: data}, nil
}

func (id *ID) UnmarshalJSON(data []byte) error {
	var input IDJSONView
	err := json.Unmarshal(data, &input)
	if err != nil {
		return fmt.Errorf("unmarshaling ID: %w", err)
	}

	val, err := IDFromString(input.ID)
	if err != nil {
		return fmt.Errorf("unmarshaling ID: %w", err)
	}

	*id = val
	return nil
}

var _ json.Unmarshaler = (*ID)(nil)

func (did DID) String() string {
	return base32.StdEncoding.EncodeToString(did.PublicKey)
}

func (did DID) MarshalJSON() ([]byte, error) {
	return json.Marshal(DIDJSONView{DID: did.String()})
}

var _ json.Marshaler = DID{}

func DIDFromString(s string) (DID, error) {
	data, err := base32.StdEncoding.DecodeString(s)
	if err != nil {
		return DID{}, fmt.Errorf("decode DID: %w", err)
	}

	return DID{PublicKey: data}, nil
}

func (did *DID) UnmarshalJSON(data []byte) error {
	var input DIDJSONView
	err := json.Unmarshal(data, &input)
	if err != nil {
		return fmt.Errorf("unmarshaling DID: %w", err)
	}

	val, err := DIDFromString(input.DID)
	if err != nil {
		return fmt.Errorf("unmarshaling DID: %w", err)
	}

	*did = val
	return nil
}

var _ json.Unmarshaler = (*DID)(nil)

func (a Address) String() string {
	return a.HostID + ":" + a.InboxAddress
}

func AddressFromString(_ string) (Address, error) {
	// TODO
	return Address{}, ErrTODO
}

func (a Handle) String() string {
	return fmt.Sprintf("%s[%s]@%s", a.ID, a.DID, a.Address)
}

func HandleFromString(_ string) (Handle, error) {
	// TODO
	return Handle{}, ErrTODO
}
