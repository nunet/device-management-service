package crypto

import (
	"fmt"

	"gitlab.com/nunet/device-management-service/types"
)

// CreatePaymentAddress generates a keypair based on the wallet type. Currently supported types: ethereum, cardano.
func CreatePaymentAddress(wallet string) (*types.Account, error) {
	var (
		pair *types.Account
		err  error
	)
	switch wallet {
	case "ethereum":
		pair, err = GetEthereumAddressAndPrivateKey()
	case "cardano":
		pair, err = GetCardanoAddressAndMnemonic()
	default:
		return nil, fmt.Errorf("invalid wallet")
	}
	if err != nil {
		return nil, fmt.Errorf("could not generate %s address: %w", wallet, err)
	}
	return pair, nil
}
