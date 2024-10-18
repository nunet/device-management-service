package crypto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCardanoAddressAndMnemonic(t *testing.T) {
	addrAndMnemonic, _ := GetCardanoAddressAndMnemonic()
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
