package resolve

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"gitlab.com/nunet/device-management-service/lib/env"
)

func TestEnvResolver(t *testing.T) {
	t.Parallel()
	mock := env.NewMockEnvironment()
	_ = mock.Setenv("FOO", "bar")

	r := NewEnvResolver(mock)
	val, err := r.Resolve("FOO")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(val) != "bar" {
		t.Fatalf("expected 'bar', got '%s'", string(val))
	}

	_, err = r.Resolve("MISSING")
	if !errors.Is(err, ErrNotExist) {
		t.Fatalf("expected ErrNotExist, got %v", err)
	}
}

func TestFileResolver(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	base := "/work"
	_ = fs.MkdirAll(base, 0o755)
	_ = afero.WriteFile(fs, filepath.Join(base, "file.txt"), []byte("content"), 0o644)

	r := NewFileResolver(fs, base)
	val, err := r.Resolve("file.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(val) != "content" {
		t.Fatalf("expected 'content', got '%s'", string(val))
	}

	_, err = r.Resolve("missing.txt")
	if !errors.Is(err, ErrNotExist) {
		t.Fatalf("expected ErrNotExist, got %v", err)
	}
}

func TestResolverProcess_EnvAndDefaults(t *testing.T) {
	t.Parallel()
	mock := env.NewMockEnvironment()
	_ = mock.Setenv("NAME", "Alice")

	res := NewResolver(map[string]Handler{
		"env":  NewEnvResolver(mock),
		"file": NewFileResolver(afero.NewMemMapFs(), "/work"),
	}, nil)

	// Explicit env
	out, err := res.Process("Hello ${env:NAME}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "Hello Alice" {
		t.Fatalf("expected 'Hello Alice', got '%s'", out)
	}

	// Implicit env default source
	out, err = res.Process("Hi ${NAME}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "Hi Alice" {
		t.Fatalf("expected 'Hi Alice', got '%s'", out)
	}

	// Default value with ':-'
	out, err = res.Process("Missing ${env:MISSING:-default}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "Missing default" {
		t.Fatalf("expected 'Missing default', got '%s'", out)
	}

	// Missing without default -> error wraps ErrNotExist
	_, err = res.Process("Oops ${env:MISSING}")
	if !errors.Is(err, ErrNotExist) {
		t.Fatalf("expected ErrNotExist, got %v", err)
	}
}

func TestResolverProcess_FileAndUnknownSource(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	base := "/work"
	_ = fs.MkdirAll(base, 0o755)
	_ = afero.WriteFile(fs, filepath.Join(base, "data.txt"), []byte("DATA"), 0o644)

	res := NewResolver(map[string]Handler{
		"file": NewFileResolver(fs, base),
	}, nil)

	out, err := res.Process("File ${file:data.txt}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "File DATA" {
		t.Fatalf("expected 'File DATA', got '%s'", out)
	}

	// Default for missing file
	out, err = res.Process("File ${file:missing.txt:-EMPTY}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "File EMPTY" {
		t.Fatalf("expected 'File EMPTY', got '%s'", out)
	}

	// Unknown source
	_, err = res.Process("X ${unknown:KEY}")
	if err == nil {
		t.Fatalf("expected error for unknown source")
	}
}
