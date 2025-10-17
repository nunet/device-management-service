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
	"errors"
	"math/big"
	"reflect"
	"testing"
)

func TestHexToAddress(t *testing.T) {
	got := hexToAddress("0x00000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c1")
	want := "0x90f8bf6a479f320ead074411a4b0e7944ea8c9c1"
	if got != want {
		t.Errorf("hexToAddress() = %s, want %s", got, want)
	}
}

func TestHexToBigInt(t *testing.T) {
	got := hexToBigInt("0x2a") // 42
	want := big.NewInt(42)
	if got.Cmp(want) != 0 {
		t.Errorf("hexToBigInt() = %v, want %v", got, want)
	}
}

func TestConvertToDecimals(t *testing.T) {
	amount := big.NewInt(123456789)
	got := convertToDecimals(amount, 6)
	want := "123.456789"
	if got != want {
		t.Errorf("convertToDecimals() = %s, want %s", got, want)
	}
}

func TestGetERC20Transfers_Success(t *testing.T) {
	logs := []map[string]interface{}{
		{
			"topics": []interface{}{
				"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
				"0x0000000000000000000000001111111111111111111111111111111111111111",
				"0x0000000000000000000000002222222222222222222222222222222222222222",
			},
			"data":            "0x0000000000000000000000000000000000000000000000000000000005f5e100", // 100,000,000
			"transactionHash": "0xabc123",                                                           // add this
		},
	}
	raw, _ := json.Marshal(logs)

	c := &mockCaller{resp: &RPCResponse{Result: raw, Error: nil}}
	txs, err := GetERC20Transfers(c, "0xToken", "0x2222222222222222222222222222222222222222", "0x0", "latest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []ERC20Tx{
		{
			From:   "0x1111111111111111111111111111111111111111",
			To:     "0x2222222222222222222222222222222222222222",
			Amount: "100.000000",
			TxHash: "0xabc123",
		},
	}
	if !reflect.DeepEqual(txs, want) {
		t.Errorf("GetERC20Transfers() = %+v, want %+v", txs, want)
	}
}

func TestGetERC20Transfers_CallError(t *testing.T) {
	c := &mockCaller{resp: nil, err: errors.New("rpc failed")}

	_, err := GetERC20Transfers(c, "0xToken", "0xAddr", "0x0", "latest")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestGetERC20Transfers_RPCResponseError(t *testing.T) {
	c := &mockCaller{resp: &RPCResponse{Result: []byte(`[]`)}, err: errors.New("some error")}

	_, err := GetERC20Transfers(c, "0xToken", "0xAddr", "0x0", "latest")
	if err == nil {
		t.Fatalf("expected rpc error, got nil")
	}
}

func TestGetERC20Transfers_JSONError(t *testing.T) {
	c := &mockCaller{resp: &RPCResponse{Result: []byte(`not-json`), Error: nil}}

	_, err := GetERC20Transfers(c, "0xToken", "0xAddr", "0x0", "latest")
	if err == nil {
		t.Fatalf("expected JSON unmarshal error, got nil")
	}
}

type mockCaller struct {
	resp *RPCResponse
	err  error
}

func (m *mockCaller) Call(_ string, _ []interface{}) (*RPCResponse, error) {
	return m.resp, m.err
}
