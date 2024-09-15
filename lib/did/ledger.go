package did

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"

	"gitlab.com/nunet/device-management-service/lib/crypto"
)

const ledgerCLI = "ledger-cli"

type LedgerWalletProvider struct {
	did  DID
	pubk crypto.PubKey
	acct int
}

var _ Provider = (*LedgerWalletProvider)(nil)

type LedgerKeyOutput struct {
	Key     string `json:"key"`
	Address string `json:"address"`
}

type LedgerSignOutput struct {
	ECDSA LedgerSignECDSAOutput `json:"ecdsa"`
}

type LedgerSignECDSAOutput struct {
	V uint   `json:"v"`
	R string `json:"r"`
	S string `json:"s"`
}

func NewLedgerWalletProvider(acct int) (Provider, error) {
	tmp, err := getLedgerTmpFile()
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp)

	var output LedgerKeyOutput
	if err := ledgerExec(
		tmp,
		&output,
		"key",
		"-o", tmp,
		"-a", fmt.Sprintf("%d", acct),
	); err != nil {
		return nil, fmt.Errorf("error executing ledger cli: %w", err)
	}

	// decode the hex key
	raw, err := hex.DecodeString(output.Key)
	if err != nil {
		return nil, fmt.Errorf("decode ledger key: %w", err)
	}

	pubk, err := crypto.UnmarshalEthPublicKey(raw)
	if err != nil {
		return nil, fmt.Errorf("unmarshal ledger raw key: %w", err)
	}

	did := FromPublicKey(pubk)

	return &LedgerWalletProvider{
		did:  did,
		pubk: pubk,
		acct: acct,
	}, nil
}

func ledgerExec(tmp string, output interface{}, args ...string) error {
	ledger, err := exec.LookPath(ledgerCLI)
	if err != nil {
		return fmt.Errorf("can't find %s in PATH: %w", ledgerCLI, err)
	}

	cmd := exec.Command(ledger, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("get ledger key: %w", err)
	}

	f, err := os.Open(tmp)
	if err != nil {
		return fmt.Errorf("open ledger output: %w", err)
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	if err := decoder.Decode(&output); err != nil {
		return fmt.Errorf("parse ledger output: %w", err)
	}

	return nil
}

func getLedgerTmpFile() (string, error) {
	tmp, err := os.CreateTemp("", "ledger.out")
	if err != nil {
		return "", fmt.Errorf("creating temporary file: %w", err)
	}
	tmpPath := tmp.Name()
	tmp.Close()

	return tmpPath, nil
}

func (p *LedgerWalletProvider) DID() DID {
	return p.did
}

func (p *LedgerWalletProvider) Sign(data []byte) ([]byte, error) {
	tmp, err := getLedgerTmpFile()
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp)

	dataHex := hex.EncodeToString(data)

	var output LedgerSignOutput
	if err := ledgerExec(
		tmp,
		&output,
		"sign",
		"-o", tmp,
		"-a", fmt.Sprintf("%d", p.acct),
		dataHex,
	); err != nil {
		return nil, fmt.Errorf("error executing ledger cli: %w", err)
	}

	rBytes, err := hex.DecodeString(output.ECDSA.R)
	if err != nil {
		return nil, fmt.Errorf("error decoding signature r: %w", err)
	}
	sBytes, err := hex.DecodeString(output.ECDSA.S)
	if err != nil {
		return nil, fmt.Errorf("error decoding signature s: %w", err)
	}

	r := secp256k1.ModNScalar{}
	s := secp256k1.ModNScalar{}
	if overflow := r.SetByteSlice(rBytes); overflow {
		return nil, fmt.Errorf("signature r overflowed")
	}
	if overflow := s.SetByteSlice(sBytes); overflow {
		return nil, fmt.Errorf("signature s overflowed")
	}

	sig := ecdsa.NewSignature(&r, &s)
	return sig.Serialize(), nil
}

func (p *LedgerWalletProvider) Anchor() Anchor {
	return NewAnchor(p.did, p.pubk)
}

func (p *LedgerWalletProvider) PrivateKey() (crypto.PrivKey, error) {
	return nil, fmt.Errorf("ledger private key cannot be exported: %w", ErrHardwareKey)
}
