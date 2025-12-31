// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package did

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os/exec"
	"regexp"

	"gitlab.com/nunet/device-management-service/lib/crypto"
)

const eternlCLI = "eternl-cli"

type EternlWalletProvider struct {
	did  DID
	pubk crypto.PubKey
}

var _ Provider = (*EternlWalletProvider)(nil)

func signDataWithBinary(binaryPath string, data string) (string, string, error) {
	// add simple data to get a signature/public key
	if data == "" {
		data = "6765745F7075626B6579"
	}

	cmd := exec.Command(binaryPath, data)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("failed to run binary: %w\nOutput: %s", err, out.String())
	}

	re := regexp.MustCompile(`PubKeyRaw:\[([a-fA-F0-9]+)\]`)
	matches := re.FindStringSubmatch(out.String())
	if len(matches) < 2 {
		return "", "", fmt.Errorf("pub value not found in output")
	}
	re2 := regexp.MustCompile(`Signature:\[([a-fA-F0-9]+)\]`)
	matches2 := re2.FindStringSubmatch(out.String())
	if len(matches2) < 2 {
		return "", "", fmt.Errorf("sig value not found in output")
	}

	pubkey := matches[1]
	sig := matches2[1]

	return pubkey, sig, nil
}

func NewEternlWalletProvider() (Provider, error) {
	eCli, err := exec.LookPath(eternlCLI)
	if err != nil {
		return nil, fmt.Errorf("can't find %s in PATH: %w", eternlCLI, err)
	}

	pub, _, err := signDataWithBinary(eCli, "")
	if err != nil {
		return nil, err
	}
	pubKeyBytes, err := hex.DecodeString(pub)
	if err != nil {
		return nil, err
	}

	pubKey, err := crypto.UnmarshalCardanoPublicKey(pubKeyBytes)
	if err != nil {
		return nil, err
	}

	did := FromPublicKey(pubKey)

	return &EternlWalletProvider{
		did:  did,
		pubk: pubKey,
	}, nil
}

func (p *EternlWalletProvider) Signer() Signer {
	return EternlSigner
}

func (p *EternlWalletProvider) DID() DID {
	return p.did
}

func (p *EternlWalletProvider) Sign(data []byte) ([]byte, error) {
	eCli, err := exec.LookPath(eternlCLI)
	if err != nil {
		return nil, fmt.Errorf("can't find %s in PATH: %w", eternlCLI, err)
	}
	dataHex := hex.EncodeToString(data)
	_, sig, err := signDataWithBinary(eCli, dataHex)
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(sig)
}

func (p *EternlWalletProvider) Anchor() Anchor {
	return NewAnchor(p.did, p.pubk)
}

func (p *EternlWalletProvider) PrivateKey() (crypto.PrivKey, error) {
	return nil, fmt.Errorf("eternl private key cannot be exported: %w", ErrHardwareKey)
}
