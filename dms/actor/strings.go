package actor

import (
	"fmt"
)

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
