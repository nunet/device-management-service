package ethereum

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetBlockNumber_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := RPCResponse{
			JSONRPC: "2.0",
			ID:      1,
			Result:  json.RawMessage(`"0x1234"`),
		}
		err := json.NewEncoder(w).Encode(resp)
		assert.NoError(t, err)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	blockNum, err := GetBlockNumber(c)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0x1234), blockNum)
}

func TestGetBlockNumber_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := RPCResponse{
			JSONRPC: "2.0",
			ID:      1,
			Error:   &RPCError{Code: -32000, Message: "Server error"},
		}
		err := json.NewEncoder(w).Encode(resp)
		assert.NoError(t, err)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	blockNum, err := GetBlockNumber(c)
	assert.Error(t, err)
	assert.Equal(t, uint64(0), blockNum)
	assert.Contains(t, err.Error(), "Server error")
}
