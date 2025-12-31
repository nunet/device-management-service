// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	homeDir, _     = os.UserHomeDir()
	defaultCfgName = "dms_config"
	defaultCfgExt  = "json"
	validate       = validator.New()
)

var DefaultConfig = Config{
	General: General{
		Env:                    "production",
		UserDir:                fmt.Sprintf("%s/.nunet", homeDir),
		WorkDir:                fmt.Sprintf("%s/nunet", homeDir),
		DataDir:                fmt.Sprintf("%s/nunet/data", homeDir),
		Debug:                  false,
		PortAvailableRangeFrom: 16384,
		PortAvailableRangeTo:   65536,
		StorageCADirectory:     fmt.Sprintf("%s/.nunet/storage_ca_directory", homeDir),
		StorageBricksDir:       fmt.Sprintf("%s/.nunet/storage_bricks_dir", homeDir),
		PaymentProvider: PaymentProvider{
			Mode:               false,
			EthereumRPCURL:     "https://ethereum-sepolia-rpc.publicnode.com",
			NtxContractAddress: "0xB37216b70a745129966E553cF8Ee2C51e1cB359A", // TSTNTX
			EthereumRPCToken:   "",
		},
	},
	Rest: Rest{
		Addr: "127.0.0.1",
		Port: 9999,
	},
	Profiler: Profiler{
		Enabled: true,
		Addr:    "127.0.0.1",
		Port:    6060,
	},
	P2P: P2P{
		ListenAddress: []string{
			"/ip4/0.0.0.0/tcp/9000",
			"/ip4/0.0.0.0/udp/9000/quic-v1",
		},
		BootstrapPeers: []string{
			"/dnsaddr/bootstrap.p2p.nunet.io/p2p/12D3KooWHzew9HTYzywFuvTHGK5Yzoz7qAhMfxagtCvhvjheoBQ3",
			"/dnsaddr/bootstrap.p2p.nunet.io/p2p/12D3KooWJMtMN1mTNRfgMqUygT7eSXamVzc9ihpSjeairm9PebmB",
			"/dnsaddr/bootstrap.p2p.nunet.io/p2p/12D3KooWKjSodxxi7UfRHzuk7eGgUF49MoPUCJvtva9K12TqDDsi",
		},
		Memory:          1024,
		FileDescriptors: 512,
	},
	Observability: Observability{
		// TODO bind in observability
		Logging: Logging{
			Level: "INFO",
			File:  fmt.Sprintf("%s/nunet/logs/nunet-dms-logs.jsonl", homeDir),
			Rotation: Rotation{
				MaxSizeMB:  100,
				MaxBackups: 3,
				MaxAgeDays: 28,
			},
		},
		Elastic: Elastic{
			URL:                "https://telemetry.nunet.io",
			Index:              "nunet-dms",
			FlushInterval:      5,
			Enabled:            false,
			APIKey:             "",
			InsecureSkipVerify: false,
		},
	},
	APM: APM{
		ServerURL:   "https://apm.telemetry.nunet.io",
		ServiceName: "nunet-dms",
		Environment: "production",
		APIKey:      "",
	},
	Job: Job{
		AllowPrivilegedDocker: false,
	},
}

// Loader encapsulates Viper, the loaded Config, and an abstract filesystem.
type Loader struct {
	v       *viper.Viper
	cfg     *Config
	fs      afero.Fs
	cfgMu   sync.RWMutex
	once    sync.Once
	cfgFile *string
}

// Option configures a Loader (functional-options pattern).
type Option func(*Loader)

// WithFS swaps the filesystem (defaults to OS FS).
func WithFS(fs afero.Fs) Option { return func(l *Loader) { l.fs = fs } }

// WithConfig swaps the default config (defaults to DefaultConfig).
func WithConfig(cfg *Config) Option { return func(l *Loader) { l.cfg = cfg } }

// NewLoader creates a Loader with sane defaults (does not read files).
func NewLoader(opts ...Option) *Loader {
	l := &Loader{
		v:   viper.New(),
		fs:  afero.NewOsFs(),
		cfg: &Config{},
	}

	*l.cfg = DefaultConfig

	for _, opt := range opts {
		opt(l)
	}

	l.init()
	return l
}

// tryReadConfig mirrors v.ReadInConfig() but falls back for custom filesystems.
func tryReadConfig(vip *viper.Viper, fs afero.Fs) error {
	if err := vip.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			name := defaultCfgName + "." + defaultCfgExt // dms_config.json
			rel := "./" + name                           // ./dms_config.json
			for _, cand := range []string{rel, name} {   // test & prod cases
				if ok, _ := afero.Exists(fs, cand); ok {
					raw, _ := afero.ReadFile(fs, cand)
					vip.SetConfigFile(cand)
					return vip.ReadConfig(bytes.NewReader(raw))
				}
			}
		} else {
			return err // syntax / permission errors, etc.
		}
	}
	return nil
}

func (l *Loader) init() {
	v := l.v
	v.SetFs(l.fs)

	v.SetConfigName(defaultCfgName)
	v.SetConfigType(defaultCfgExt)
	v.AddConfigPath("./")
	if homeDir, err := os.UserHomeDir(); err == nil {
		v.AddConfigPath(fmt.Sprintf("%s/.nunet", homeDir)) // $HOME
	}
	if configDir, err := os.UserConfigDir(); err == nil {
		v.AddConfigPath(fmt.Sprintf("%s/nunet", configDir)) // $CONFIG
	}
	if runtime.GOOS != "windows" {
		v.AddConfigPath("/etc/nunet/") // system
	}

	_ = l.setConfig(*l.cfg, true)

	v.SetEnvPrefix("DMS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
}

// Public Loader API
// Get reads the config file (if any), overlays env & flags, validates, and
// stores the result. Subsequent calls are cheap no-ops.
func (l *Loader) Load() (*Config, error) {
	var err error
	l.once.Do(func() { err = l.Reload() })
	return l.cfg, err
}

func (l *Loader) ConfigFile() string {
	return l.v.ConfigFileUsed()
}

// Reload forces a fresh read; useful for SIGHUP hot-reload.
func (l *Loader) Reload() error {
	l.cfgMu.Lock()
	migrated, err := l.readAndUnmarshal()
	l.cfgMu.Unlock()

	if migrated && err == nil {
		if err = l.Write(true); err != nil {
			return fmt.Errorf("persist migrated config: %w", err)
		}
	}

	return err
}

func (l *Loader) SetConfig(c Config) {
	l.cfgMu.Lock()
	*l.cfg = c
	l.cfgMu.Unlock()
	_ = l.setConfig(c, false)
}

func (l *Loader) setConfig(cfg Config, def bool) error {
	l.cfgMu.Lock()
	defer l.cfgMu.Unlock()

	tmp := viper.New()
	tmp.SetConfigType("json")
	raw, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	if err = tmp.ReadConfig(bytes.NewReader(raw)); err != nil {
		return err
	}
	for _, k := range tmp.AllKeys() {
		v := tmp.Get(k)
		if def {
			l.v.SetDefault(k, v)
		} else {
			l.v.Set(k, v)
		}
	}
	return nil
}

func (l *Loader) GetConfig() (*Config, error) {
	l.cfgMu.Lock()
	defer l.cfgMu.Unlock()
	return l.cfg, nil
}

// Update sets a single dotted key, validates the struct, then writes to disk.
func (l *Loader) Set(key string, value interface{}) error {
	key = strings.ToLower(key)

	tmp := viper.New()
	tmp.SetConfigType("json")
	if err := tmp.MergeConfigMap(l.v.AllSettings()); err != nil {
		return err
	}
	tmp.Set(key, value)

	var probe Config
	if err := tmp.UnmarshalExact(&probe); err != nil {
		return err // reject: unknown key or wrong type
	}

	if err := l.v.MergeConfigMap(map[string]any{key: value}); err != nil {
		return err
	}

	l.cfgMu.Lock()
	*l.cfg = probe
	l.cfgMu.Unlock()

	return l.Write(false)
}

// Write persists the current in-memory config to disk atomically.
// • Creates the file (and parent directories) on first run.
// • Uses a temp-file + rename so it can’t be truncated on crash.
func (l *Loader) Write(backupExisting bool) error {
	l.cfgMu.RLock()
	defer l.cfgMu.RUnlock()

	cfgPath := l.v.ConfigFileUsed()
	if cfgPath == "" {
		// First run – default to "./dms_config.json" (same as search path 1)
		cfgPath = fmt.Sprintf("./%s.%s", defaultCfgName, defaultCfgExt)
		l.v.SetConfigFile(cfgPath)
	}

	if backupExisting {
		backupPath := fmt.Sprintf("%s.bk.%d", cfgPath, time.Now().Unix())
		if err := l.fs.Rename(cfgPath, backupPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("couldn't make backup of config file ahead of write. aborting write. Error: %w", err)
		}
	}

	// Ensure directory hierarchy exists.
	if err := l.fs.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	raw, err := json.Marshal(l.cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return fmt.Errorf("unmarshal map: %w", err)
	}
	if err := l.v.MergeConfigMap(m); err != nil {
		return fmt.Errorf("merge into viper: %w", err)
	}

	tmpPath := strings.TrimSuffix(cfgPath, filepath.Ext(cfgPath)) + ".tmp" + filepath.Ext(cfgPath)

	// Always try to clean up the temp file If Rename succeeds the file
	// no longer exists so this is a harmless no-op
	defer func() { _ = l.fs.Remove(tmpPath) }()

	if err := l.v.WriteConfigAs(tmpPath); err != nil {
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := l.fs.Rename(tmpPath, cfgPath); err != nil {
		return fmt.Errorf("atomic rename failed: %w", err)
	}

	return nil
}

// BindFlags attaches CLI flags to the Loader’s Viper instance.
func (l *Loader) BindFlags(fs *pflag.FlagSet) {
	v := l.v

	// --config is special: record its value, do NOT bind to Viper.
	cfgFile := new(string)
	fs.StringVar(cfgFile, "config", "", "config file (override search paths)")
	l.cfgFile = cfgFile

	type flag struct {
		name, key, short, usage string
		isBool, isInt           bool
	}
	flags := []flag{
		{"rest-addr", "rest.addr", "", "REST API host", false, false},
		{"rest-port", "rest.port", "", "REST API port", false, true},
		{"user-dir", "general.user_dir", "", "user directory", false, false},
		{"work-dir", "general.work_dir", "", "work directory", false, false},
		{"data-dir", "general.data_dir", "", "data directory", false, false},
		{"debug", "general.debug", "", "debug mode", true, false},
		{"profiler-enabled", "profiler.enabled", "", "enable profiler", true, false},
		{"profiler-addr", "profiler.addr", "", "profiler address", false, false},
		{"profiler-port", "profiler.port", "", "profiler port", false, true},
	}

	for _, f := range flags {
		switch {
		case f.isBool:
			fs.BoolP(f.name, f.short, v.GetBool(f.key), f.usage)
		case f.isInt:
			fs.IntP(f.name, f.short, v.GetInt(f.key), f.usage)
		default:
			fs.StringP(f.name, f.short, v.GetString(f.key), f.usage)
		}
		_ = v.BindPFlag(f.key, fs.Lookup(f.name))
	}
}

func (l *Loader) readAndUnmarshal() (migrated bool, err error) {
	// honour --config flag if supplied
	if l.cfgFile != nil && *l.cfgFile != "" {
		l.v.SetConfigFile(*l.cfgFile)
	}

	if err := tryReadConfig(l.v, l.fs); err != nil {
		return false, fmt.Errorf("read config: %w", err)
	}

	if err := l.v.UnmarshalExact(
		l.cfg,
		func(c *mapstructure.DecoderConfig) { c.ZeroFields = true },
	); err != nil {
		return false, fmt.Errorf("unmarshal config: %w", err)
	}

	migrated = migrateLegacyObservability(l)

	if err := validate.Struct(l.cfg); err != nil {
		return false, fmt.Errorf("validate config: %w", err)
	}
	return migrated, nil
}

// GetValue fetches a value by dotted key. second return is false if unset.
func (l *Loader) GetValue(key string) (interface{}, bool) {
	key = strings.ToLower(key)
	if !l.v.IsSet(key) {
		return nil, false
	}
	return l.v.Get(key), true
}

// migrateLegacyObservability copies values from the deprecated flat
// observability keys into the new nested structure IF, AND ONLY IF, the
// nested fields have not been set  This lets old and new config files
// work side-by-side and means we can remove the flat keys in a later
// release without breaking users
func migrateLegacyObservability(l *Loader) (migrated bool) {
	oCfg := &l.cfg.Observability

	// return if no legacy keys are set
	if oCfg.LogLevel == "" &&
		oCfg.LogFile == "" &&
		oCfg.MaxSize == 0 &&
		oCfg.MaxBackups == 0 &&
		oCfg.MaxAge == 0 &&
		oCfg.ElasticsearchURL == "" &&
		oCfg.ElasticsearchIndex == "" &&
		oCfg.FlushInterval == 0 &&
		!oCfg.ElasticsearchEnabled &&
		oCfg.ElasticsearchAPIKey == "" &&
		!oCfg.InsecureSkipVerify {
		return false
	}

	oMap, ok := l.v.Get("observability").(map[string]any)
	if !ok {
		return false
	}

	if oCfg.Logging.Level == "" && oCfg.LogLevel != "" {
		l.v.Set("observability.logging.level", oCfg.LogLevel)
	}
	if oCfg.Logging.File == "" && oCfg.LogFile != "" {
		l.v.Set("observability.logging.file", oCfg.LogFile)
	}
	if oCfg.Logging.Rotation.MaxSizeMB == 0 && oCfg.MaxSize != 0 {
		l.v.Set("observability.logging.rotation.max_size_mb", oCfg.MaxSize)
	}
	if oCfg.Logging.Rotation.MaxBackups == 0 && oCfg.MaxBackups != 0 {
		l.v.Set("observability.logging.rotation.max_backups", oCfg.MaxBackups)
	}
	if oCfg.Logging.Rotation.MaxAgeDays == 0 && oCfg.MaxAge != 0 {
		l.v.Set("observability.logging.rotation.max_age_days", oCfg.MaxAge)
	}

	// NOTE: elastic flat keys are still valid and left untouched - they'll
	// be removed in a later major version once users have migrated
	if oCfg.ElasticsearchURL != "" {
		l.v.Set("observability.elastic.url", oCfg.ElasticsearchURL)
	}
	if oCfg.ElasticsearchIndex != "" {
		l.v.Set("observability.elastic.index", oCfg.ElasticsearchIndex)
	}
	if oCfg.FlushInterval != 0 {
		l.v.Set("observability.elastic.flush_interval", oCfg.FlushInterval)
	}
	if !oCfg.Elastic.Enabled && oCfg.ElasticsearchEnabled {
		l.v.Set("observability.elastic.enabled", oCfg.ElasticsearchEnabled)
	}
	if oCfg.Elastic.APIKey == "" && oCfg.ElasticsearchAPIKey != "" {
		l.v.Set("observability.elastic.api_key", oCfg.ElasticsearchAPIKey)
	}
	if !oCfg.Elastic.InsecureSkipVerify && oCfg.InsecureSkipVerify {
		l.v.Set("observability.elastic.insecure_skip_verify", oCfg.InsecureSkipVerify)
	}

	// clean up old keys
	delete(oMap, "max_size")
	delete(oMap, "max_backups")
	delete(oMap, "max_age")
	delete(oMap, "log_file")
	delete(oMap, "log_level")
	delete(oMap, "elasticsearch_url")
	delete(oMap, "elasticsearch_index")
	delete(oMap, "flush_interval")
	delete(oMap, "elasticsearch_enabled")
	delete(oMap, "elasticsearch_api_key")
	delete(oMap, "insecure_skip_verify")

	err := l.v.MergeConfigMap(map[string]any{"observability": oMap})
	return err == nil
}
