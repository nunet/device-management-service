package cardano

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeTransfersForAsset(t *testing.T) {
	bfAPIKey := os.Getenv("DMS_TEST_BF_API_KEY")
	if bfAPIKey == "" {
		t.Skip("DMS_TEST_BF_API_KEY environment variable not set")
	}
	client := NewClient(
		bfAPIKey,
		"https://cardano-mainnet.blockfrost.io/api/v0",
	)

	// example transaction hash and asset
	txHash := "1aefa026d15057876f02be28b45d6ad7326f6c97d05f53f93f68e3b3a426fb0f"
	asset := "edfd7a1d77bcb8b884c474bdc92a16002d1fb720e454fa6e993444794e5458"
	expectedFrom := "addr1q95c4gzekg6df03a5armclydpwaxejc2tnqux9430chyz6hctqxrm0qlrysfsgp5j2kg6s79c5kkyypz7gj66s2034js5cddvj"
	expectedTo := "addr1q92dxz3puff3zzy2h2zxgh88j4ruxxysftrc8czg79j3qxpr99gl4fnhwn73t4at6pvsmf7h84frr5ava0pxgww2zztqppa6rz"
	expectedAmount := "243188"

	transfers := client.ComputeTransfersForAsset(t.Context(), txHash, asset)
	require.NotNil(t, transfers)

	for _, transfer := range transfers {
		assert.Equal(
			t,
			expectedFrom,
			transfer.From,
		)
		assert.Equal(
			t,
			expectedTo,
			transfer.To,
		)
		for _, ast := range transfer.Assets {
			if ast.Unit == asset {
				assert.Equal(
					t,
					expectedAmount,
					ast.Quantity.String(),
				)
			}
		}
	}
}
