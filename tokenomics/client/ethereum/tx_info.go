// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package ethereum

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
)

func hexToAddress(topic string) string {
	return "0x" + topic[len(topic)-40:]
}

func hexToBigInt(hexStr string) *big.Int {
	n := new(big.Int)
	n.SetString(strings.TrimPrefix(hexStr, "0x"), 16)
	return n
}

type ERC20Tx struct {
	From   string
	To     string
	Amount string
	TxHash string
}

func GetERC20Transfers(c Caller, tokenAddress, toAddress string, fromBlock, toBlock string) ([]ERC20Tx, error) {
	transferTopic := "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
	paddedTo := "0x" + strings.Repeat("0", 24) + strings.ToLower(strings.TrimPrefix(toAddress, "0x"))

	params := []interface{}{
		map[string]interface{}{
			"fromBlock": fromBlock,
			"toBlock":   toBlock,
			"address":   tokenAddress,
			"topics":    []interface{}{transferTopic, nil, paddedTo},
		},
	}

	resp, err := c.Call("eth_getLogs", params)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, err
	}

	var logs []map[string]interface{}
	if err := json.Unmarshal(resp.Result, &logs); err != nil {
		return nil, err
	}
	txs := make([]ERC20Tx, len(logs))
	for i, l := range logs {
		topics := l["topics"].([]interface{})
		if len(topics) == 0 {
			continue
		}

		var txHash string
		if h, ok := l["transactionHash"].(string); ok {
			txHash = h
		}
		fromAddr := hexToAddress(topics[1].(string))
		toAddr := hexToAddress(topics[2].(string))
		amount := hexToBigInt(l["data"].(string))
		humanAmount := convertToDecimals(amount, 6)
		txs[i] = ERC20Tx{
			From:   fromAddr,
			To:     toAddr,
			Amount: humanAmount,
			TxHash: txHash,
		}
	}

	return txs, nil
}

// convertToDecimals converts a big.Int token amount into a string with decimals
func convertToDecimals(amount *big.Int, decimals int) string {
	dec := big.NewInt(0).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil) // 10^decimals
	intPart := new(big.Int).Div(amount, dec)
	fracPart := new(big.Int).Mod(amount, dec)
	fracStr := fmt.Sprintf("%0*d", decimals, fracPart)

	return fmt.Sprintf("%s.%s", intPart.String(), fracStr)
}
