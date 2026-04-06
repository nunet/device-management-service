// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package parser

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/dms/jobs/parser/types"
)

type mockParser struct {
	decodeF func(data []byte, dest any, opts *types.Options) error
	encodeF func(data any) ([]byte, error)
}

func (m mockParser) Decode(data []byte, dest any, opts *types.Options) error {
	if m.decodeF == nil {
		return nil
	}

	return m.decodeF(data, dest, opts)
}

func (m mockParser) Encode(data any) ([]byte, error) {
	if m.encodeF == nil {
		return nil, nil
	}

	return m.encodeF(data)
}

func withTestRegistry(t *testing.T) {
	t.Helper()

	prev := registry
	registry = &Registry{parsers: make(map[SpecType]types.Parser)}
	t.Cleanup(func() {
		registry = prev
	})
}

func TestDecodeReturnsErrorForUnknownSpecType(t *testing.T) {
	withTestRegistry(t)

	var out any
	err := Decode(SpecType("unknown"), []byte("{}"), &out, &Options{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parser for spec type unknown not found")
}

func TestEncodeReturnsErrorForUnknownSpecType(t *testing.T) {
	withTestRegistry(t)

	b, err := Encode(SpecType("unknown"), map[string]any{"a": 1})
	require.Error(t, err)
	assert.Nil(t, b)
	assert.Contains(t, err.Error(), "parser for spec type unknown not found")
}

func TestDecodeDispatchesToRegisteredParser(t *testing.T) {
	withTestRegistry(t)

	const spec = SpecType("test-spec")
	wantData := []byte(`{"k":"v"}`)
	wantOpts := &Options{WorkingDir: "/tmp/work"}
	out := map[string]any{}

	called := false
	registry.RegisterParser(spec, mockParser{
		decodeF: func(data []byte, dest any, opts *types.Options) error {
			called = true
			assert.Equal(t, wantData, data)
			assert.Same(t, &out, dest)
			require.NotNil(t, opts)
			assert.Equal(t, wantOpts.WorkingDir, opts.WorkingDir)
			return nil
		},
	})

	err := Decode(spec, wantData, &out, wantOpts)
	require.NoError(t, err)
	assert.True(t, called)
}

func TestDecodePropagatesParserError(t *testing.T) {
	withTestRegistry(t)

	const spec = SpecType("test-spec")
	wantErr := errors.New("decode failed")

	registry.RegisterParser(spec, mockParser{
		decodeF: func(_ []byte, _ any, _ *types.Options) error {
			return wantErr
		},
	})

	var out any
	err := Decode(spec, []byte("{}"), &out, &Options{})
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
}

func TestEncodeDispatchesToRegisteredParser(t *testing.T) {
	withTestRegistry(t)

	const spec = SpecType("test-spec")
	in := map[string]any{"hello": "world"}
	want := []byte("encoded")

	called := false
	registry.RegisterParser(spec, mockParser{
		encodeF: func(data any) ([]byte, error) {
			called = true
			assert.Equal(t, in, data)
			return want, nil
		},
	})

	got, err := Encode(spec, in)
	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, want, got)
}

func TestEncodePropagatesParserError(t *testing.T) {
	withTestRegistry(t)

	const spec = SpecType("test-spec")
	wantErr := errors.New("encode failed")

	registry.RegisterParser(spec, mockParser{
		encodeF: func(_ any) ([]byte, error) {
			return nil, wantErr
		},
	})

	b, err := Encode(spec, map[string]any{"k": "v"})
	require.Error(t, err)
	assert.Nil(t, b)
	assert.ErrorIs(t, err, wantErr)
}
