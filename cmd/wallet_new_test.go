package cmd

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/types"
)

type MockWalletService struct {
	address    string
	mnemonic   string
	privateKey string
}

func (mw *MockWalletService) GetCardanoAddressAndMnemonic() (*types.BlockchainAddressPrivKey, error) {
	addr := &types.BlockchainAddressPrivKey{
		Address:  mw.address,
		Mnemonic: mw.mnemonic,
	}

	return addr, nil
}

func (mw *MockWalletService) GetEthereumAddressAndPrivateKey() (*types.BlockchainAddressPrivKey, error) {
	addr := &types.BlockchainAddressPrivKey{
		Address:    mw.address,
		PrivateKey: mw.privateKey,
	}

	return addr, nil
}

func TestWalletNewCmdCardano(t *testing.T) {
	buf := new(bytes.Buffer)

	address := "addr1qabcdef123"
	mnemonic := "foo bar"

	mockWallet := &MockWalletService{
		address:  address,
		mnemonic: mnemonic,
	}
	cmd := NewWalletNewCmd(mockWallet)

	cmd.SetOut(buf)
	cmd.SetErr(buf)
	// testing with --cardano flag
	cmd.SetArgs([]string{"--cardano"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("error executing command: %v", err)
	}

	if !flagAda {
		t.Fatalf("Expected true for cardano flag, but got %t", flagAda)
	}

	expected := fmt.Sprintf("address: %s\nmnemonic: %s\n", address, mnemonic)

	assert.Equal(t, buf.String(), expected)

	buf.Reset()
}

func TestWalletNewCmdEthereum(t *testing.T) {
	buf := new(bytes.Buffer)

	address := "0x123abcdef"
	privateKey := "0x456ghijkl"

	mockWallet := &MockWalletService{
		address:    address,
		privateKey: privateKey,
	}
	cmd := NewWalletNewCmd(mockWallet)

	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--ethereum"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("error executing command: %v", err)
	}

	if !flagEth {
		t.Fatalf("Expected true for ethereum flag, but got %t", flagEth)
	}

	expected := fmt.Sprintf("address: %s\nprivate_key: %s\n", address, privateKey)

	assert.Equal(t, buf.String(), expected)

	buf.Reset()
}
