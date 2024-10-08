package utils

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/fivebinaries/go-cardano-serialization/address"
)

// ValidateAddress checks if the wallet address is a valid ethereum/cardano address
func ValidateAddress(addr string) error {
	if common.IsHexAddress(addr) {
		return errors.New("ethereum wallet address not allowed")
	}

	validCardano := false
	isValidCardano(addr, &validCardano)
	if validCardano {
		return nil
	}

	return errors.New("invalid cardano wallet address")
}

// isValidCardano checks if the cardano address is valid
func isValidCardano(addr string, valid *bool) {
	defer func() {
		if r := recover(); r != nil {
			*valid = false
		}
	}()
	if _, err := address.NewAddress(addr); err == nil {
		*valid = true
	}
}
