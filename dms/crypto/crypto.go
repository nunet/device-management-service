package crypto

import (
	"fmt"

	"gitlab.com/nunet/device-management-service/types"
)

// CreatePaymentAddress generates a keypair based on the wallet type. Currently supported types: cardano.
func CreatePaymentAddress(wallet string) (*types.Account, error) {
	if wallet != "cardano" {
		return nil, fmt.Errorf("invalid wallet")
	}
	pair, err := GetCardanoAddressAndMnemonic()
	if err != nil {
		return nil, fmt.Errorf("could not generate %s address: %w", wallet, err)
	}
	return pair, nil
}
