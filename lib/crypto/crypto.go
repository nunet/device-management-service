package crypto

import (
	"crypto/rand"
	"errors"
	"io"

	"golang.org/x/crypto/sha3"
)

// RandomEntropy bytes from rand.Reader
func RandomEntropy(length int) ([]byte, error) {
	buf := make([]byte, length)
	n, err := io.ReadFull(rand.Reader, buf)
	if err != nil || n != length {
		return nil, errors.New("failed to read random bytes")
	}
	return buf, nil
}

// Sha3 return sha3 of a given byte array
func Sha3(data ...[]byte) ([]byte, error) {
	d := sha3.New256()
	for _, b := range data {
		_, err := d.Write(b)
		if err != nil {
			return nil, err
		}
	}
	return d.Sum(nil), nil
}
