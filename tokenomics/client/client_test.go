package client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCall_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected application/json, got %s", ct)
		}

		resp := RPCResponse{
			JSONRPC: "2.0",
			ID:      1,
			Result:  json.RawMessage(`"0x123"`),
		}
		err := json.NewEncoder(w).Encode(resp)
		assert.NoError(t, err)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	resp, err := c.Call("eth_blockNumber", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(resp.Result) != "\"0x123\"" {
		t.Errorf("expected 0x123, got %s", resp.Result)
	}
}

func TestCall_WithAuthorization(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "Bearer testtoken" {
			t.Errorf("expected Authorization header, got %s", auth)
		}
		resp := RPCResponse{JSONRPC: "2.0", ID: 1, Result: json.RawMessage(`"ok"`)}
		err := json.NewEncoder(w).Encode(resp)
		assert.NoError(t, err)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "testtoken")
	_, err := c.Call("eth_chainId", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCall_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := RPCResponse{
			JSONRPC: "2.0",
			ID:      1,
			Error:   &RPCError{Code: -32601, Message: "Method not found"},
		}
		err := json.NewEncoder(w).Encode(resp)
		assert.NoError(t, err)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	resp, err := c.Call("unknown_method", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Error == nil || resp.Error.Message != "Method not found" {
		t.Errorf("expected Method not found error, got %+v", resp.Error)
	}
}

func TestCall_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := io.WriteString(w, "{invalid-json}")
		assert.NoError(t, err)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	_, err := c.Call("eth_blockNumber", nil)
	if err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Errorf("expected JSON error, got %v", err)
	}
}

func TestCall_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := io.WriteString(w, "server error")
		assert.NoError(t, err)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	_, err := c.Call("eth_blockNumber", nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
