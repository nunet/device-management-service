package libp2p

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/pnet"
	"github.com/spf13/afero"
)

/*
    ** Swarm key **

    By default, the swarm key shall be stored in a file named `swarm.key`
    using the following pathbased codec:

   `/key/swarm/psk/1.0.0/<base_encoding>/<256_bits_key>`

   `<base_encoding>` is either bin, base16 or base64.
*/

// TODO-pnet-1: we shouldn't handle configuration paths here, a general configuration path
// should be provided by /internal/config.go
func getBasePath(_ afero.Fs) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("error getting home directory: %w", err)
	}

	nunetDir := filepath.Join(homeDir, ".nunet")
	return nunetDir, nil
}

// configureSwarmKey try to read the swarm key from `<config_path>/swarm.key` file.
// If a swarm key is not found, generate a new one.
//
// TODO-ask: should we continue to generate a new swarm key if one is not found?
// Or we should enforce the user to use some cmd/API rpc to generate a new one?
func configureSwarmKey(fs afero.Fs) (pnet.PSK, error) {
	var psk pnet.PSK
	var err error

	psk, err = getSwarmKey(fs)
	if err != nil {
		psk, err = generateSwarmKey(fs)
		if err != nil {
			return nil, fmt.Errorf("failed to generate new swarm key: %w", err)
		}
	}
	return psk, nil
}

// getSwarmKey reads the swarm key from a file
func getSwarmKey(fs afero.Fs) (pnet.PSK, error) {
	homeDir, err := getBasePath(fs)
	if err != nil {
		return nil, fmt.Errorf("failed to get base file path: %w", err)
	}
	swarmkey, err := afero.ReadFile(fs, filepath.Join(homeDir, "swarm.key"))
	if err != nil {
		return nil, fmt.Errorf("failed to read swarm key file: %w", err)
	}

	psk, err := pnet.DecodeV1PSK(bytes.NewReader(swarmkey))
	if err != nil {
		return nil, fmt.Errorf("failed to configure private network: %s", err)
	}

	// TODO-ask: should we return psk fingerprint?
	return psk, nil
}

// generateSwarmKey generates a new swarm key, storing it within
// `<nunet_config_dir>/swarm.key`.
func generateSwarmKey(fs afero.Fs) (pnet.PSK, error) {
	priv, _, err := crypto.GenerateKeyPair(crypto.Secp256k1, 256)
	if err != nil {
		return nil, err
	}

	privBytes, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return nil, err
	}
	encodedKey := base64.StdEncoding.EncodeToString(privBytes)

	swarmKeyWithCodec := fmt.Sprintf("/key/swarm/psk/1.0.0/\n/base64/\n%s\n", encodedKey)

	// TODO-pnet-1
	nunetDir, err := getBasePath(fs)
	if err != nil {
		return nil, err
	}

	swarmKeyPath := filepath.Join(nunetDir, "swarm.key")
	// nolint:gofumpt
	if err := afero.WriteFile(fs, swarmKeyPath, []byte(swarmKeyWithCodec), 0600); err != nil {
		return nil, fmt.Errorf("error writing swarm key to file: %w", err)
	}

	psk, err := pnet.DecodeV1PSK(bytes.NewReader([]byte(swarmKeyWithCodec)))
	if err != nil {
		return nil, fmt.Errorf("failed to decode generated swarm key: %s", err)
	}

	zlog.Sugar().Infof("A new Swarm key was generated and written to %s\n"+
		"IMPORTANT: If you'd like to create the swarm key using a cryptography algorithm "+
		"of your choice, just modify the swarm.key file with your own key.\n"+
		"The content of `swarm.key` should look like: `/key/swarm/psk/1.0.0/<base_encoding>/<your_key>`\n"+
		"where `<base_encoding>` is either `bin`, `base16`, or `base64`.\n",
		swarmKeyPath,
	)

	return psk, nil
}
