package ensemblev1

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/dms/jobs/parser/types"
	"gitlab.com/nunet/device-management-service/lib/env"
)

func TestResolvePlaceholders_EnvAndFile(t *testing.T) {
	t.Parallel()
	// Build a dummy config tree that contains placeholders at different depths
	data := any(map[string]any{
		"top": "Hello ${env:USER_NAME:-guest}",
		"nested": map[string]any{
			"msg": "File says: ${file:msg.txt}",
		},
	})

	// Setup env and in-memory FS
	mockEnv := env.NewMockEnvironment()
	_ = mockEnv.Setenv("USER_NAME", "alice")

	mem := afero.NewMemMapFs()
	base := "/work"
	require.NoError(t, mem.MkdirAll(base, 0o755))
	relFile := "msg.txt"
	absFile := filepath.Join(base, relFile)
	require.NoError(t, afero.WriteFile(mem, absFile, []byte("hello-from-file"), 0o644))

	opts := &types.Options{
		Env:        mockEnv,
		Fs:         afero.Afero{Fs: mem},
		WorkingDir: base,
	}

	require.NoError(t, resolvePlaceholders(&data, opts))

	root := data.(map[string]any)
	require.Equal(t, "Hello alice", root["top"]) // env with default syntax used but env exists

	nested := root["nested"].(map[string]any)
	require.Equal(t, "File says: hello-from-file", nested["msg"]) // file source
}

func TestResolvePlaceholders_DefaultFallback(t *testing.T) {
	t.Parallel()
	data := any(map[string]any{
		"a": "${env:NOT_SET:-fallback}",
		"b": map[string]any{"c": "pre ${NOT_SET:-x}"}, // implicit env
	})

	opts := &types.Options{Env: env.NewMockEnvironment(), Fs: afero.Afero{Fs: afero.NewMemMapFs()}, WorkingDir: "/w"}
	require.NoError(t, resolvePlaceholders(&data, opts))

	m := data.(map[string]any)
	require.Equal(t, "fallback", m["a"])
	inner := m["b"].(map[string]any)
	require.Equal(t, "pre x", inner["c"])
}
