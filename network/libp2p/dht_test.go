// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package libp2p

import (
	"bytes"
	"context"
	"crypto/rand"
	"testing"
	"time"

	dht_pb "github.com/libp2p/go-libp2p-kad-dht/pb"
	record_pb "github.com/libp2p/go-libp2p-record/pb"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
	msgio "github.com/libp2p/go-msgio"
	"github.com/libp2p/go-msgio/protoio" //nolint:staticcheck
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	dmscrypto "gitlab.com/nunet/device-management-service/lib/crypto"
	commonproto "gitlab.com/nunet/device-management-service/proto/generated/v1/common"
	"gitlab.com/nunet/device-management-service/types"
)

func TestDHT_Validate(t *testing.T) {
	t.Parallel()
	customNamespace := "/test-namespace"
	validator := dhtValidator{
		PS:              nil, // Not used in Validate method
		customNamespace: customNamespace,
	}

	priv, pub, err := crypto.GenerateSecp256k1Key(rand.Reader)
	require.NoError(t, err)

	pubRaw, err := pub.Raw()
	require.NoError(t, err)

	t.Run("Empty value (deletion case)", func(t *testing.T) {
		t.Parallel()
		// Empty value is considered a deletion
		err := validator.Validate("/test-namespace:some-key", []byte{})
		assert.NoError(t, err)
	})

	t.Run("Invalid namespace prefix", func(t *testing.T) {
		t.Parallel()
		// Value doesn't matter for this test
		err := validator.Validate("/invalid:some-key", []byte("some-value"))
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKeyNamespace)
	})

	t.Run("Invalid envelope (unmarshal error)", func(t *testing.T) {
		t.Parallel()
		// Valid key but invalid envelope bytes
		err := validator.Validate("/test-namespace:some-key", []byte("not-a-valid-protobuf"))
		assert.Error(t, err)
		assert.ErrorIs(t, err, types.ErrUnmarshal)
	})

	t.Run("Invalid public key", func(t *testing.T) {
		t.Parallel()
		// Create a valid envelope but with invalid public key
		envelope := &commonproto.Advertisement{
			PeerId:    "test-peer-id",
			Timestamp: 123456789,
			Data:      []byte("test-data"),
			PublicKey: []byte("invalid-public-key"), // Invalid public key
			Signature: []byte("test-signature"),
		}

		value, err := proto.Marshal(envelope)
		require.NoError(t, err)

		err = validator.Validate("/test-namespace:some-key", value)
		assert.Error(t, err)
		assert.ErrorIs(t, err, dmscrypto.ErrUnmarshalPublicKey)
	})

	t.Run("envelope with invalid signature", func(t *testing.T) {
		t.Parallel()
		envelope := &commonproto.Advertisement{
			PeerId:    "test-peer-id",
			Timestamp: 123456789,
			Data:      []byte("test-data"),
			PublicKey: pubRaw,
			Signature: []byte("invalid-signature"), // Invalid signature
		}

		value, err := proto.Marshal(envelope)
		require.NoError(t, err)

		err = validator.Validate("/test-namespace/some-key", value)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrValidateEnvelopeByPbkey)
	})

	t.Run("Valid envelope with valid signature", func(t *testing.T) {
		t.Parallel()
		// Create the message that will be signed
		peerID := "test-peer-id"
		timestamp := int64(123456789)
		data := []byte("test-data")

		concatenatedBytes := bytes.Join([][]byte{
			[]byte(peerID),
			{byte(timestamp)},
			data,
			pubRaw,
		}, nil)

		// Sign the message
		signature, err := priv.Sign(concatenatedBytes)
		require.NoError(t, err)

		// Create envelope with valid signature
		envelope := &commonproto.Advertisement{
			PeerId:    peerID,
			Timestamp: timestamp,
			Data:      data,
			PublicKey: pubRaw,
			Signature: signature,
		}

		value, err := proto.Marshal(envelope)
		require.NoError(t, err)

		err = validator.Validate("/test-namespace:some-key", value)
		assert.NoError(t, err)
	})
}

func TestDHT_Select(t *testing.T) {
	t.Parallel()
	validator := dhtValidator{}
	idx, err := validator.Select("any-key", [][]byte{[]byte("value1"), []byte("value2")})
	assert.NoError(t, err)
	assert.Equal(t, 0, idx)
}

func TestDHT_SendMessage(t *testing.T) {
	hosts := newNetwork(t, 2, false)
	require.Len(t, hosts, 2)
	hostAlice := hosts[0]
	hostBob := hosts[1]

	testProto := protocol.ID("/test/dht/1.0.0")
	testKey := "test-key"

	// Create message handlers for the second host to receive messages
	messageReceived := make(chan struct{})
	hostBob.Host.SetStreamHandler(testProto, func(s network.Stream) {
		defer s.Close()

		// Read the message
		r := msgio.NewVarintReaderSize(s, network.MessageSizeMax)
		bytes, err := r.ReadMsg()
		require.NoError(t, err)
		defer r.ReleaseMsg(bytes)

		// Unmarshal the message
		msg := new(dht_pb.Message)
		err = msg.Unmarshal(bytes)
		require.NoError(t, err)

		// Verify the message content
		assert.Equal(t, dht_pb.Message_PUT_VALUE, msg.GetType())
		assert.Equal(t, testKey, string(msg.GetKey()))

		close(messageReceived)
	})

	// alice as dht sender
	messenger := newDHTMessageSender(hostAlice.Host, testProto)

	// Create a test message
	testMsg := &dht_pb.Message{
		Type: dht_pb.Message_PUT_VALUE,
		Key:  []byte(testKey),
	}

	err := messenger.SendMessage(context.Background(), hostBob.Host.ID(), testMsg)
	require.NoError(t, err)

	// Wait for the message to be received with timeout
	select {
	case <-messageReceived:
		// Message received successfully
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for message to be received")
	}
}

func TestDHT_SendRequest(t *testing.T) {
	hosts := newNetwork(t, 2, false)
	require.Len(t, hosts, 2)
	hostAlice := hosts[0]
	hostBob := hosts[1]

	testProto := protocol.ID("/test/dht/1.0.0")
	testKey := "test-key"
	testValue := []byte("test-value")

	expectedResponse := &dht_pb.Message{
		Type: dht_pb.Message_GET_VALUE,
		Key:  []byte(testKey),
		Record: &record_pb.Record{
			Value: testValue,
		},
	}

	// Create message handlers for bob
	messageReceived := make(chan struct{})
	hostBob.Host.SetStreamHandler(testProto, func(s network.Stream) {
		defer s.Close()

		// Read the request
		r := msgio.NewVarintReaderSize(s, network.MessageSizeMax)
		bytes, err := r.ReadMsg()
		require.NoError(t, err)
		defer r.ReleaseMsg(bytes)

		// Unmarshal the request
		msg := new(dht_pb.Message)
		err = msg.Unmarshal(bytes)
		require.NoError(t, err)

		// Verify the message content
		assert.Equal(t, dht_pb.Message_GET_VALUE, msg.GetType())
		assert.Equal(t, testKey, string(msg.GetKey()))

		// Write the response
		w := protoio.NewDelimitedWriter(s)
		err = w.WriteMsg(expectedResponse)
		require.NoError(t, err)

		close(messageReceived)
	})

	messenger := newDHTMessageSender(hostAlice.Host, testProto)

	testRequest := &dht_pb.Message{
		Type: dht_pb.Message_GET_VALUE,
		Key:  []byte(testKey),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	response, err := messenger.SendRequest(ctx, hostBob.Host.ID(), testRequest)
	require.NoError(t, err)

	// Wait for the message to be received
	select {
	case <-messageReceived:
		// Message received successfully
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for message to be received")
	}

	require.NotNil(t, response)
	assert.Equal(t, dht_pb.Message_GET_VALUE, response.GetType())
	assert.Equal(t, testKey, string(response.GetKey()))
	assert.Equal(t, testValue, response.Record.GetValue())
}
