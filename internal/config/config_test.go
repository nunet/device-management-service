// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package config

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

// Helpers

func writeSampleConfig(fs afero.Fs) error {
	// build a Config struct and marshal it instead of hard-coded JSON
	sample := Config{
		Rest: Rest{
			Addr: "0.0.0.0",
			Port: 4242,
		},
		General: General{
			UserDir: "/var/lib/nunet",
			WorkDir: "/srv/nunet",
		},
		Observability: Observability{
			Logging: Logging{
				Level: "DEBUG",
				File:  "/tmp/dms.log",
				Rotation: Rotation{
					MaxSizeMB:  20,
					MaxBackups: 2,
					MaxAgeDays: 14,
				},
			},
		},
		P2P: P2P{
			Memory:          2048,
			FileDescriptors: 1024,
			ListenAddress:   []string{"/ip4/127.0.0.1/tcp/1234"},
			BootstrapPeers:  []string{},
		},
	}

	raw, err := json.MarshalIndent(sample, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sample config: %w", err)
	}
	return afero.WriteFile(fs, "./dms_config.json", raw, 0o644)
}

// Tests (run serial for deterministic global state)
func TestLoadDefaults(t *testing.T) {
	fs := afero.NewMemMapFs()
	ldr := NewLoader(WithFS(fs))

	cfg, err := ldr.Load()
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", cfg.Rest.Addr)
	require.Equal(t, uint32(9999), cfg.Rest.Port)
	require.Equal(t, "INFO", cfg.LogLevel())
}

func TestLoadFromFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, writeSampleConfig(fs))

	ldr := NewLoader(WithFS(fs))

	cfg, err := ldr.Load()
	require.NoError(t, err)
	require.Equal(t, "0.0.0.0", cfg.Rest.Addr)
	require.Equal(t, uint32(4242), cfg.Rest.Port)
	require.Equal(t, "/var/lib/nunet", cfg.General.UserDir)
	require.Equal(t, 2048, cfg.P2P.Memory)
	require.Equal(t, "DEBUG", cfg.LogLevel())
}

func TestLoadSliceValue(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, writeSampleConfig(fs))

	ldr := NewLoader(WithFS(fs))

	cfg, err := ldr.Load()
	require.NoError(t, err)
	require.Len(t, cfg.P2P.ListenAddress, 1)
	require.Equal(t, "/ip4/127.0.0.1/tcp/1234", cfg.P2P.ListenAddress[0])
	require.Len(t, cfg.P2P.BootstrapPeers, 0)
}

func TestEnvOverride(t *testing.T) {
	t.Setenv("DMS_OBSERVABILITY_LOGGING_LEVEL", "WARN")

	fs := afero.NewMemMapFs()
	_ = writeSampleConfig(fs)

	ldr := NewLoader(WithFS(fs))
	cfg, err := ldr.Load()
	require.NoError(t, err)
	require.Equal(t, "WARN", cfg.LogLevel())
}

func TestFlagOverride(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = writeSampleConfig(fs)

	flagSet := pflag.NewFlagSet("t", pflag.ContinueOnError)
	ldr := NewLoader(WithFS(fs))
	ldr.BindFlags(flagSet)
	_ = flagSet.Parse([]string{"--rest-port", "5151"})

	cfg, err := ldr.Load()
	require.NoError(t, err)
	require.Equal(t, uint32(5151), cfg.Rest.Port)
}

func TestPrecedenceFlagsEnvFile(t *testing.T) {
	t.Setenv("DMS_REST_PORT", "6000")

	fs := afero.NewMemMapFs()
	_ = writeSampleConfig(fs)

	flagSet := pflag.NewFlagSet("t", pflag.ContinueOnError)
	ldr := NewLoader(WithFS(fs))
	ldr.BindFlags(flagSet)
	_ = flagSet.Parse([]string{"--rest-port", "7000"})

	cfg, err := ldr.Load()
	require.NoError(t, err)
	require.Equal(t, uint32(7000), cfg.Rest.Port)
}

func TestWriteAndReload(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = writeSampleConfig(fs)

	ldr := NewLoader(WithFS(fs))
	_, _ = ldr.Load()

	ldr.cfg.Rest.Port = 9090
	require.NoError(t, ldr.Write())
	require.NoError(t, ldr.Reload())
	require.Equal(t, uint32(9090), ldr.cfg.Rest.Port)
}

func TestInvalidJSONError(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = afero.WriteFile(fs, "./dms_config.json", []byte("{bad"), 0o644)

	ldr := NewLoader(WithFS(fs))
	_, err := ldr.Load()
	require.Error(t, err)
}

func TestUnknownFieldError(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = afero.WriteFile(fs, "./dms_config.json", []byte(`{"foo":42}`), 0o644)

	ldr := NewLoader(WithFS(fs))
	_, err := ldr.Load()
	require.Error(t, err)
}

func TestConcurrentLoadSafe(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = writeSampleConfig(fs)

	ldr := NewLoader(WithFS(fs))

	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := ldr.Load()
			require.NoError(t, err)
		}()
	}
	wg.Wait()
}

func TestReloadAfterExternalChange(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = writeSampleConfig(fs)

	ldr := NewLoader(WithFS(fs))
	_, _ = ldr.Load()

	raw, _ := afero.ReadFile(fs, "./dms_config.json")
	var doc map[string]any
	_ = json.Unmarshal(raw, &doc)
	doc["rest"].(map[string]any)["port"] = 8181
	newRaw, _ := json.MarshalIndent(doc, "", " ")
	_ = afero.WriteFile(fs, "./dms_config.json", newRaw, 0o644)

	require.NoError(t, ldr.Reload())
	require.Equal(t, uint32(8181), ldr.cfg.Rest.Port)
}

func TestSafeWriteCreatesFile(t *testing.T) {
	fs := afero.NewMemMapFs()

	ldr := NewLoader(WithFS(fs))
	ldr.cfg.Rest.Port = 5050
	require.NoError(t, ldr.Write())

	other := NewLoader(WithFS(fs))
	_, _ = other.Load()
	require.Equal(t, uint32(5050), other.cfg.Rest.Port)
}

func TestDeprecatedFlatKeyStillWorks(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = afero.WriteFile(fs, "./dms_config.json",
		[]byte(`{"observability":{"log_level":"TRACE"}}`), 0o644)

	ldr := NewLoader(WithFS(fs))
	cfg, _ := ldr.Load()
	require.Equal(t, "TRACE", cfg.LogLevel())
}

func TestFlagsWithCustomConfigFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = writeSampleConfig(fs)

	_ = afero.WriteFile(fs, "./my.json",
		[]byte(`{"rest":{"addr":"1.2.3.4","port":8888}}`), 0o644)

	flagSet := pflag.NewFlagSet("t", pflag.ContinueOnError)
	ldr := NewLoader(WithFS(fs))
	ldr.BindFlags(flagSet)
	_ = flagSet.Parse([]string{"--config", "my.json"})

	cfg, _ := ldr.Load()
	require.Equal(t, uint32(8888), cfg.Rest.Port)
	require.Equal(t, "1.2.3.4", cfg.Rest.Addr)
}
