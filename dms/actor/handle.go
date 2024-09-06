package actor

import (
	"fmt"
)

func (h *Handle) Empty() bool {
	return h.ID.Empty() &&
		h.DID.Empty() &&
		h.Address.Empty()
}

func (h *Handle) String() string {
	return fmt.Sprintf("%s[%s]@%s", h.ID, h.DID, h.Address)
}

func HandleFromString(_ string) (Handle, error) {
	// TODO
	return Handle{}, ErrTODO
}

func (a *Address) Empty() bool {
	return a.HostID == "" && a.InboxAddress == ""
}

func (a *Address) String() string {
	return a.HostID + ":" + a.InboxAddress
}

func AddressFromString(_ string) (Address, error) {
	// TODO
	return Address{}, ErrTODO
}
