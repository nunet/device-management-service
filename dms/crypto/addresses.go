package crypto

import (
	"github.com/fivebinaries/go-cardano-serialization/address"
	"github.com/fivebinaries/go-cardano-serialization/bip32"
	"github.com/fivebinaries/go-cardano-serialization/network"
	"github.com/tyler-smith/go-bip39"
	"gitlab.com/nunet/device-management-service/types"
)

func harden(num uint32) uint32 {
	return 0x80000000 + num
}

func GetCardanoAddressAndMnemonic() (*types.Account, error) {
	var pair types.Account
	entropy, _ := bip39.NewEntropy(256)
	mnemonic, _ := bip39.NewMnemonic(entropy)
	pair.Mnemonic = mnemonic
	rootKey := bip32.FromBip39Entropy(
		entropy,
		[]byte{},
	)
	accountKey := rootKey.Derive(harden(1852)).Derive(harden(1815)).Derive(harden(0))
	utxoPubKey := accountKey.Derive(0).Derive(0).Public()
	utxoPubKeyHash := utxoPubKey.PublicKey().Hash()
	stakeKey := accountKey.Derive(2).Derive(0).Public()
	stakeKeyHash := stakeKey.PublicKey().Hash()
	addr := address.NewBaseAddress(
		network.MainNet(),
		&address.StakeCredential{
			Kind:    address.KeyStakeCredentialType,
			Payload: utxoPubKeyHash[:],
		},
		&address.StakeCredential{
			Kind:    address.KeyStakeCredentialType,
			Payload: stakeKeyHash[:],
		})
	pair.Address = addr.String()
	return &pair, nil
}
