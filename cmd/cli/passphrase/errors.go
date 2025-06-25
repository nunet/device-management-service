package passphrase

import "errors"

var (
	ErrPassphraseNotFound  = errors.New("passphrase not found")
	ErrNewPassphraseFailed = errors.New("failed to generate new passphrase")
)
