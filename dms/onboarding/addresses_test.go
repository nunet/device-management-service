package onboarding_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/dms/onboarding"
)

func TestGetEthereumAddressAndPrivateKey(t *testing.T) {
	addrAndPrivKey, _ := onboarding.GetEthereumAddressAndPrivateKey()
	addr := addrAndPrivKey.Address
	privKey := addrAndPrivKey.PrivateKey

	t.Run("ethereum address is 42 characters long", func(t *testing.T) {
		want := 42
		assert.Equal(t, want, len(addr))
	})

	t.Run("ethereum private key is 66 characters long", func(t *testing.T) {
		want := 66
		assert.Equal(t, want, len(privKey))
	})
}

func TestGetCardanoAddressAndMnemonic(t *testing.T) {
	addrAndMnemonic, _ := onboarding.GetCardanoAddressAndMnemonic()
	addr := addrAndMnemonic.Address
	mnemonic := addrAndMnemonic.Mnemonic

	t.Run("cardano address is 103 characters long", func(t *testing.T) {
		want := 103
		assert.Equal(t, len(addr), want)
	})

	t.Run("cardano mnemonic is 24 words long", func(t *testing.T) {
		want := 24
		got := len(strings.Split(mnemonic, " "))
		assert.Equal(t, want, got)
	})
}
