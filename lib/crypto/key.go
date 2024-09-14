package crypto

import (
	"crypto/rand"
	"fmt"

	"github.com/libp2p/go-libp2p/core/crypto"
)

const (
	Ed25519   = crypto.Ed25519
	Secp256k1 = crypto.Secp256k1
)

type (
	Key     = crypto.Key
	PrivKey = crypto.PrivKey
	PubKey  = crypto.PubKey
)

func AllowedKey(t int) bool {
	switch t {
	case Ed25519:
		return true
	case Secp256k1:
		return true
	default:
		return false
	}
}

func GenerateKeyPair(t int) (PrivKey, PubKey, error) {
	switch t {
	case Ed25519:
		return crypto.GenerateEd25519Key(rand.Reader)
	case Secp256k1:
		return crypto.GenerateSecp256k1Key(rand.Reader)
	default:
		return nil, nil, fmt.Errorf("unsupported key type %d: %w", t, ErrUnsupportedKeyType)
	}
}

func PublicKeyToBytes(k PubKey) ([]byte, error) {
	return crypto.MarshalPublicKey(k)
}

func BytesToPublicKey(data []byte) (PubKey, error) {
	return crypto.UnmarshalPublicKey(data)
}

func PrivateKeyToBytes(k PrivKey) ([]byte, error) {
	return crypto.MarshalPrivateKey(k)
}

func BytesToPrivateKey(data []byte) (PrivKey, error) {
	return crypto.UnmarshalPrivateKey(data)
}

func IDFromPublicKey(k PubKey) (ID, error) {
	data, err := PublicKeyToBytes(k)
	if err != nil {
		return ID{}, fmt.Errorf("id from public key: %w", err)
	}

	return ID{PublicKey: data}, nil
}

func PublicKeyFromID(id ID) (PubKey, error) {
	return BytesToPublicKey(id.PublicKey)
}
