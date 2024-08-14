package backend

import (
	"gitlab.com/nunet/device-management-service/dms/onboarding"
	"gitlab.com/nunet/device-management-service/types"
)

type Wallet struct{}

func (w *Wallet) GetCardanoAddressAndMnemonic() (*types.BlockchainAddressPrivKey, error) {
	return onboarding.GetCardanoAddressAndMnemonic()
}

func (w *Wallet) GetEthereumAddressAndPrivateKey() (*types.BlockchainAddressPrivKey, error) {
	return onboarding.GetEthereumAddressAndPrivateKey()
}
