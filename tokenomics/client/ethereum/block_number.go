package ethereum

import (
	"encoding/json"
)

// GetBlockNumber returns the number of the most recent block.
func GetBlockNumber(c Caller) (uint64, error) {
	resp, err := c.Call("eth_blockNumber", nil)
	if err != nil {
		return 0, err
	}
	if resp.Error != nil {
		return 0, resp.Error
	}

	var result string
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return 0, err
	}

	return hexToBigInt(result).Uint64(), nil
}
