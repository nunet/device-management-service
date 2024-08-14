package types

type EncryptionType int

const (
	EncryptionTypeNull EncryptionType = iota
)

// Encryptor (DRAFT)
//
// Warning: this is just a draft. And it might be moved to an Encryption package
//
// TODO: it must support encryption of files/directories, otherwise we have to
// create another interface for the usecase
type Encryptor interface {
	Encrypt([]byte) ([]byte, error)
}

// Decryptor (DRAFT)
//
// Warning: this is just a draft. And it might be moved to an Encryption package
//
// TODO: it must support decryption of files/directories, otherwise we have to
// create another interface for the usecase
type Decryptor interface {
	Decrypt([]byte) ([]byte, error)
}
