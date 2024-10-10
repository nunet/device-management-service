package types

// Account represents a blockchain account with an address, private key, and mnemonic.
type Account struct {
	Address    string `json:"address,omitempty"`
	PrivateKey string `json:"private_key,omitempty"`
	Mnemonic   string `json:"mnemonic,omitempty"`
}
