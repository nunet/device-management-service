package did

import (
	"fmt"
	"strings"

	libp2p_crypto "github.com/libp2p/go-libp2p/core/crypto"
	mb "github.com/multiformats/go-multibase"
	varint "github.com/multiformats/go-varint"

	"gitlab.com/nunet/device-management-service/lib/crypto"
)

type PublicKeyAnchor struct {
	did  DID
	pubk crypto.PubKey
}

var _ Anchor = (*PublicKeyAnchor)(nil)

type PrivateKeyProvider struct {
	did   DID
	privk crypto.PrivKey
}

var _ Provider = (*PrivateKeyProvider)(nil)

func NewAnchor(did DID, pubk crypto.PubKey) Anchor {
	return &PublicKeyAnchor{
		did:  did,
		pubk: pubk,
	}
}

func NewProvider(did DID, privk crypto.PrivKey) Provider {
	return &PrivateKeyProvider{
		did:   did,
		privk: privk,
	}
}

func (a *PublicKeyAnchor) DID() DID {
	return a.did
}

func (a *PublicKeyAnchor) Verify(data []byte, sig []byte) error {
	ok, err := a.pubk.Verify(data, sig)
	if err != nil {
		return err
	}

	if !ok {
		return ErrInvalidSignature
	}

	return nil
}

func (a *PublicKeyAnchor) PublicKey() crypto.PubKey {
	return a.pubk
}

func (p *PrivateKeyProvider) DID() DID {
	return p.did
}

func (p *PrivateKeyProvider) Sign(data []byte) ([]byte, error) {
	return p.privk.Sign(data)
}

func (p *PrivateKeyProvider) PrivateKey() crypto.PrivKey {
	return p.privk
}

func (p *PrivateKeyProvider) Anchor() Anchor {
	return NewAnchor(p.did, p.privk.GetPublic())
}

func FromID(id crypto.ID) (DID, error) {
	pubk, err := crypto.PublicKeyFromID(id)
	if err != nil {
		return DID{}, fmt.Errorf("public key from id: %w", err)
	}

	return FromPublicKey(pubk), nil
}

func FromPublicKey(pubk crypto.PubKey) DID {
	uri := FormatKeyURI(pubk)
	return DID{URI: uri}
}

func PublicKeyFromDID(did DID) (crypto.PubKey, error) {
	if did.Method() != "key" {
		return nil, ErrInvalidDID
	}

	pubk, err := ParseKeyURI(did.URI)
	if err != nil {
		return nil, fmt.Errorf("parsing did key identifier: %w", err)
	}

	return pubk, nil
}

func AnchorFromPublicKey(pubk crypto.PubKey) (Anchor, error) {
	did := FromPublicKey(pubk)
	return NewAnchor(did, pubk), nil
}

func ProviderFromPrivateKey(privk crypto.PrivKey) (Provider, error) {
	did := FromPublicKey(privk.GetPublic())
	return NewProvider(did, privk), nil
}

// Note: this code originated in https://github.com/ucan-wg/go-ucan/blob/main/didkey/key.go
// Copyright applies; some superficial modifications by vyzo.

const (
	multicodecKindEd25519PubKey uint64 = 0xed

	keyPrefix = "did:key"
)

func FormatKeyURI(pubk crypto.PubKey) string {
	raw, err := pubk.Raw()
	if err != nil {
		return ""
	}

	// TODO other supported key types (secp?)
	if t := pubk.Type(); t != crypto.Ed25519 {
		// we don't support those yet
		log.Errorf("unsupported key type: %d", t)
		return ""
	}
	t := multicodecKindEd25519PubKey
	size := varint.UvarintSize(t)
	data := make([]byte, size+len(raw))
	n := varint.PutUvarint(data, t)
	copy(data[n:], raw)

	b58BKeyStr, err := mb.Encode(mb.Base58BTC, data)
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%s:%s", keyPrefix, b58BKeyStr)
}

func ParseKeyURI(uri string) (crypto.PubKey, error) {
	if !strings.HasPrefix(uri, keyPrefix) {
		return nil, fmt.Errorf("decentralized identifier is not a 'key' type")
	}

	uri = strings.TrimPrefix(uri, keyPrefix+":")

	enc, data, err := mb.Decode(uri)
	if err != nil {
		return nil, fmt.Errorf("decoding multibase: %w", err)
	}

	if enc != mb.Base58BTC {
		return nil, fmt.Errorf("unexpected multibase encoding: %s", mb.EncodingToStr[enc])
	}

	keyType, n, err := varint.FromUvarint(data)
	if err != nil {
		return nil, err
	}

	switch keyType {
	case multicodecKindEd25519PubKey:
		pubk, err := libp2p_crypto.UnmarshalEd25519PublicKey(data[n:])
		if err != nil {
			return nil, err
		}
		return pubk, nil

	default:
		return nil, ErrInvalidKeyType
	}
}
