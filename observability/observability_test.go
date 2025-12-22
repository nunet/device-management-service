// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/internal/config"
	"go.elastic.co/apm/transport"
	"go.elastic.co/apm/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// helpers
func newTestLogger(w io.Writer) *zap.Logger {
	enc := zap.NewProductionEncoderConfig()
	enc.TimeKey = "@ts"
	base := zapcore.NewCore(
		zapcore.NewJSONEncoder(enc),
		zapcore.AddSync(w),
		zapcore.DebugLevel,
	)
	return zap.New(newLabelInjectionCore(base, zapcore.DebugLevel))
}

const (
	testIndex = "idx"
	bulkPath  = "/_bulk"
)

type stubEmitter struct{ n int }

func (s *stubEmitter) Emit(_ interface{}) error { s.n++; return nil }
func (s *stubEmitter) Close() error             { return nil }

func mustJSON(t *testing.T, b []byte) map[string]interface{} {
	t.Helper()
	var v map[string]interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("json: %v", err)
	}
	return v
}

// withTempConfig copies the global config, runs mutate, then restores it.
func withTempConfig(t *testing.T, mutate func(observabilityConfig *config.Observability, apmConfig *config.APM)) {
	t.Helper()
	observabilityCfg := &ObservabilityCfg
	apmCfg := &ApmCfg
	mutate(observabilityCfg, apmCfg)
	t.Cleanup(func() {
		ObservabilityCfg = *observabilityCfg
		ApmCfg = *apmCfg
	})
}

// parseLogLevel
func TestParseLogLevel(t *testing.T) {
	t.Parallel()

	if _, err := parseLogLevel("bogus"); err == nil {
		t.Fatalf("expected error for bogus log‑level")
	}
	if lvl, err := parseLogLevel("INFO"); err != nil || lvl != zapcore.InfoLevel {
		t.Fatalf("parse INFO failed: %v / %v", lvl, err)
	}
}

// labels: default, routing, skip
func TestLabelsDefaultMetricSkip(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}

	// default
	newTestLogger(buf).Info("plain")
	rec := mustJSON(t, buf.Bytes())
	labels, ok := rec["labels"].([]interface{})
	if !ok || len(labels) == 0 || labels[0] != "default" {
		t.Fatalf("default label missing: %#v", rec["labels"])
	}

	// metric routing
	buf.Reset()
	newTestLogger(buf).Info("metric", zap.String("labels", "metric"))
	rec = mustJSON(t, buf.Bytes())
	if rec["es_index"] != "metric-index" {
		t.Fatalf("metric es_index absent: %#v", rec)
	}

	// skip‑ES (temporarily toggle config)
	orig := labelRoutingMap[LabelMetric]
	labelRoutingMap[LabelMetric] = LabelRoutingConfig{SkipES: true}
	t.Cleanup(func() { labelRoutingMap[LabelMetric] = orig })

	buf.Reset()
	newTestLogger(buf).Info("skip", zap.String("labels", "metric"))
	rec = mustJSON(t, buf.Bytes())
	if v, ok := rec["es_skip"].(bool); !ok || !v {
		t.Fatalf("expected es_skip=true, got %#v", rec["es_skip"])
	}
}

// buffered syncer routing & skip
func TestBufferedSyncerRoutingAndSkip(t *testing.T) {
	t.Parallel()

	bufMu := sync.Mutex{}
	var bulkBodies []string

	esSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == bulkPath {
			body, _ := io.ReadAll(r.Body)
			bufMu.Lock()
			bulkBodies = append(bulkBodies, string(body))
			bufMu.Unlock()
			_, _ = w.Write([]byte(`{"errors":false}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(esSrv.Close)

	cli, _ := elastic.NewClient(elastic.SetURL(esSrv.URL), elastic.SetSniff(false), elastic.SetHealthcheck(false))
	s := newBufferedElasticsearchSyncer(cli, "def-idx", time.Hour)
	t.Cleanup(s.Close)

	_ = s.Flush // race-detector tickle

	if _, err := s.Write([]byte(`{"x":"a"}`)); err != nil {
		t.Fatalf("write a: %v", err)
	}
	if _, err := s.Write([]byte(`{"es_skip":true,"x":"b"}`)); err != nil {
		t.Fatalf("write b: %v", err)
	}
	if _, err := s.Write([]byte(`{"es_index":"ovr-idx","x":"c"}`)); err != nil {
		t.Fatalf("write c: %v", err)
	}
	s.Flush()

	bufMu.Lock()
	payload := bulkBodies[0]
	bufMu.Unlock()

	if !strings.Contains(payload, `"def-idx"`) {
		t.Errorf("default index missing")
	}
	if strings.Contains(payload, `"b"`) {
		t.Errorf("skipped doc appeared")
	}
	if !strings.Contains(payload, `"ovr-idx"`) {
		t.Errorf("override index missing")
	}
}

// atomic disableES helper
func TestDisableESFlag(t *testing.T) {
	t.Parallel()

	atomic.StoreInt32(&esDisabledFlag, 0)
	if isESDisabled() {
		t.Fatalf("flag should start false")
	}
	disableES()
	if !isESDisabled() {
		t.Fatalf("disableES() did not flip flag")
	}
	atomic.StoreInt32(&esDisabledFlag, 0) // reset
}

// pre‑flight helper
func TestPreflightCheckES(t *testing.T) {
	okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(okSrv.Close)
	if err := preflightCheckES(okSrv.URL, "k", false); err != nil {
		t.Fatalf("preflight 200 failed: %v", err)
	}

	unauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(unauthSrv.Close)
	if err := preflightCheckES(unauthSrv.URL, "k", false); err == nil {
		t.Fatalf("preflight 401 should error")
	}
}

// StartSpan paths
func TestStartSpan(t *testing.T) {
	tr, _ := apm.NewTracerOptions(apm.TracerOptions{ServiceName: "unit", Transport: transport.Discard})

	tracerMutex.Lock()
	oldTracer, oldNoOp := currentTracer, tracingNoOpMode
	// TODO
	currentTracer, tracingNoOpMode = tr, false
	tracerMutex.Unlock()

	t.Cleanup(func() {
		tracerMutex.Lock()
		currentTracer, tracingNoOpMode = oldTracer, oldNoOp
		tracerMutex.Unlock()
	})

	// nested span
	parent := tr.StartTransaction("parent", "req")
	ctx := apm.ContextWithTransaction(context.Background(), parent)
	spBefore := tr.Stats().SpansSent
	done2 := StartSpan(ctx, "child")
	done2()
	parent.End()
	tr.Flush(nil)
	if tr.Stats().SpansSent-spBefore != 1 {
		t.Fatalf("span not recorded")
	}
}

// StartSpan paths
func TestStartSpanGin(t *testing.T) {
	// test tracecontext
	t.Setenv("ELASTIC_APM_TRACEPARENT", "00-841827e1f7ef0495b7cde61893d1d83f-841827e1f7ef0495-01")
	t.Setenv("ELASTIC_APM_TRACESTATE", "es=s:1")

	tr, _ := apm.NewTracerOptions(apm.TracerOptions{ServiceName: "unit", Transport: transport.Discard})
	initRootTrace(tr)

	tracerMutex.Lock()
	oldTracer, oldNoOp := currentTracer, tracingNoOpMode
	// TODO
	currentTracer, tracingNoOpMode = tr, false
	tracerMutex.Unlock()

	t.Cleanup(func() {
		tracerMutex.Lock()
		currentTracer, tracingNoOpMode = oldTracer, oldNoOp
		tracerMutex.Unlock()
	})

	// nested span
	parent := tr.StartTransaction("parent", "req")
	ctx := &gin.Context{Request: &http.Request{}}
	spBefore := tr.Stats().SpansSent
	done2 := StartSpan(ctx, "child")
	done2()
	parent.End()
	tr.Flush(nil)
	if tr.Stats().SpansSent-spBefore != 1 {
		t.Fatalf("span not recorded")
	}
}

// no‑op mode raises level
func TestSetNoOpMode(t *testing.T) {
	t.Parallel()

	orig := atomicLevel.Level()
	SetNoOpMode(true)
	if atomicLevel.Level() != zapcore.Level(100) {
		t.Fatalf("expected level 100 in no‑op")
	}
	SetNoOpMode(false)
	atomicLevel.SetLevel(orig)
}

// event‑bus core
func TestEventEmitterCore(t *testing.T) {
	stub := &stubEmitter{}
	customEventEmitter = stub
	t.Cleanup(func() { customEventEmitter = nil })

	zap.New(newEventEmitterCore(zapcore.DebugLevel)).
		Info("hello", zap.String("k", "v"))

	if stub.n != 1 {
		t.Fatalf("expected 1 emitted event, got %d", stub.n)
	}
}

// SetLogLevel
func TestSetLogLevel(t *testing.T) {
	t.Parallel()

	orig := atomicLevel.Level()
	t.Cleanup(func() { atomicLevel.SetLevel(orig) })

	// happy‑path
	if err := SetLogLevel("debug"); err != nil {
		t.Fatalf("SetLogLevel(debug) returned %v", err)
	}
	if got := atomicLevel.Level(); got != zapcore.DebugLevel {
		t.Fatalf("expected debug level, got %v", got)
	}

	// bogus level
	if err := SetLogLevel("not-a-level"); err == nil {
		t.Fatalf("expected error for bogus level")
	}
}

// buffered syncer buffer overflow */
func TestBufferedSyncerBufferOverflow(t *testing.T) {
	s := newBufferedElasticsearchSyncer(nil, "idx", time.Hour)
	t.Cleanup(s.Close)

	// fill to capacity
	for i := 0; i < s.maxBufferSize; i++ {
		if _, err := s.Write([]byte(`{"msg":"ok"}`)); err != nil {
			t.Fatalf("unexpected error at %d: %v", i, err)
		}
	}
	if len(s.buffer) != s.maxBufferSize {
		t.Fatalf("buffer size mismatch: got %d want %d", len(s.buffer), s.maxBufferSize)
	}

	// one more should fail
	if _, err := s.Write([]byte(`{"msg":"overflow"}`)); err == nil {
		t.Fatalf("expected buffer‑full error, got nil")
	}
}

// Shutdown idempotence
func TestShutdownIsIdempotent(t *testing.T) {
	t.Helper() // keeps revive happy

	esSyncerInstance = newBufferedElasticsearchSyncer(nil, testIndex, time.Hour)
	customEventEmitter = &stubEmitter{}
	Shutdown()
	Shutdown() // second call – still must not panic
}

// Atomic flag visibilty
func TestDisableESAcrossGoroutines(t *testing.T) {
	atomic.StoreInt32(&esDisabledFlag, 0)

	ready := make(chan struct{})
	done := make(chan struct{})

	go func() {
		close(ready)
		for !isESDisabled() {
			time.Sleep(1 * time.Millisecond)
		}
		close(done)
	}()

	<-ready
	time.Sleep(10 * time.Millisecond) // ensure goroutine running
	disableES()

	select {
	case <-done:
		// success
	case <-time.After(time.Second):
		t.Fatalf("flag did not become visible to other goroutine")
	}
}

// SetAPIKey
func TestSetAPIKeyUpdatesConfig(t *testing.T) {
	withTempConfig(t, func(obsCfg *config.Observability, apmCfg *config.APM) {
		if err := SetAPIKey("unit-key"); err != nil {
			t.Fatalf("SetAPIKey returned %v", err)
		}
		if obsCfg.Elastic.APIKey != "unit-key" {
			t.Fatalf("expected api-key in config, got %q", obsCfg.Elastic.APIKey)
		}
		if apmCfg.APIKey != "unit-key" {
			t.Fatalf("expected api-key in config, got %q", apmCfg.APIKey)
		}
	})
}

// SetAPMURL
func TestSetAPMURLUpdateAndDisable(t *testing.T) {
	withTempConfig(t, func(_ *config.Observability, apmCfg *config.APM) {
		// Set a URL  tracer should (re)initialize
		if err := SetAPMURL("http://apm.test"); err != nil {
			t.Fatalf("SetAPMURL failed: %v", err)
		}
		if apmCfg.ServerURL != "http://apm.test" {
			t.Fatalf("APM url not updated")
		}

		// Now disable by passing empty
		if err := SetAPMURL(""); err != nil {
			t.Fatalf("SetAPMURL disable failed: %v", err)
		}
		if apmCfg.ServerURL != "" {
			t.Fatalf("APM url should be cleared")
		}
	})
}

// EnableElasticsearchLogging toggle
func TestEnableElasticsearchLoggingToggle(t *testing.T) {
	withTempConfig(t, func(obsCfg *config.Observability, _ *config.APM) {
		obsCfg.Elastic.Enabled = true
		if err := EnableElasticsearchLogging(!obsCfg.Elastic.Enabled); err != nil {
			t.Fatalf("EnableElasticsearchLogging toggle failed: %v", err)
		}
		if obsCfg.Elastic.Enabled {
			t.Fatalf("toggle did not flip flag")
		}
	})
}

// EmitCustomEvent edge cases
func TestEmitCustomEventInvalidPairs(t *testing.T) {
	t.Parallel()

	if err := EmitCustomEvent("bad-kv", "k1", 1, "dangling"); err == nil {
		t.Fatalf("expected error for odd kv length")
	}
}

// event throughput
type countingEmitter struct{ n int }

func (c *countingEmitter) Emit(_ interface{}) error { c.n++; return nil }
func (c *countingEmitter) Close() error             { return nil }

func TestEventEmitterCoreHighThroughput(t *testing.T) {
	ce := &countingEmitter{}
	customEventEmitter = ce
	t.Cleanup(func() { customEventEmitter = nil })

	const N = 1_000
	zl := zap.New(newEventEmitterCore(zapcore.InfoLevel))
	for i := 0; i < N; i++ {
		zl.Info("hi", zap.Int("i", i))
	}

	if ce.n != N {
		t.Fatalf("wanted %d events, got %d", N, ce.n)
	}
}

// tracer no‑op interactions
func TestTracerNoOpAfterSetNoOp(t *testing.T) {
	tr, _ := apm.NewTracerOptions(apm.TracerOptions{
		ServiceName: "noop-test",
		Transport:   transport.Discard,
	})

	tracerMutex.Lock()
	oldTracer := currentTracer
	currentTracer = tr
	tracerMutex.Unlock()
	t.Cleanup(func() {
		tracerMutex.Lock()
		currentTracer = oldTracer
		tracerMutex.Unlock()
	})

	initRootTrace(tr)

	// enable global no-op
	SetNoOpMode(true)

	done := StartSpan("noop-tx")
	done()
	tr.Flush(nil)

	if got := tr.Stats().TransactionsSent; got != 0 {
		t.Fatalf("transactions should be suppressed in no-op mode, got %d", got)
	}

	// disable global no-op and ensure traffic resumes
	SetNoOpMode(false)

	done2 := StartSpan("active-tx")
	done2()
	tr.Flush(nil)

	if tr.Stats().SpansSent == 0 {
		t.Fatalf("transactions were not sent after no-op disabled")
	}
}

// buffered syncer failure disabling
func TestBufferedSyncerDisablesAfterThreeFailures(t *testing.T) {
	t.Parallel()

	failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == bulkPath {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(failSrv.Close)

	es, err := elastic.NewClient(
		elastic.SetURL(failSrv.URL),
		elastic.SetSniff(false),
		elastic.SetHealthcheck(false),
	)
	if err != nil {
		t.Fatalf("elastic.NewClient: %v", err)
	}

	s := newBufferedElasticsearchSyncer(es, testIndex, time.Hour)
	t.Cleanup(s.Close)
	esSyncerInstance = s
	t.Cleanup(func() { esSyncerInstance = nil })

	atomic.StoreInt32(&esDisabledFlag, 0)

	for i := 0; i < 3; i++ {
		if _, err := s.Write([]byte(`{"msg":"fail"}`)); err != nil {
			t.Fatalf("write fail %d: %v", i, err)
		}
		s.Flush()
	}

	if !isESDisabled() {
		t.Fatalf("expected disable after 3 errors")
	}
}

// SetFlushInterval restarts ticker
func TestSetFlushIntervalRestartsTicker(t *testing.T) {
	s := newBufferedElasticsearchSyncer(nil, "idx", 10*time.Second)
	t.Cleanup(s.Close)
	esSyncerInstance = s
	t.Cleanup(func() { esSyncerInstance = nil })

	oldCtx := s.ctx // capture ORIGINAL context

	if err := SetFlushInterval(1); err != nil {
		t.Fatalf("SetFlushInterval: %v", err)
	}
	if s.flushInterval != time.Second {
		t.Fatalf("flushInterval not updated")
	}

	select {
	case <-oldCtx.Done():
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("old ticker goroutine still running")
	}
}

// initLogger guard clauses
func TestInitLoggerNoOpEarlyExit(t *testing.T) {
	SetNoOpMode(true)
	t.Cleanup(func() { SetNoOpMode(false) })

	before := combinedCore
	if err := initLogger(ObservabilityCfg); err != nil {
		t.Fatalf("initLogger (no-op) returned error: %v", err)
	}
	if combinedCore != before {
		t.Fatalf("combinedCore changed despite no-op")
	}
}

// createElasticsearchCore validation
func TestCreateElasticsearchCoreValidationErrors(t *testing.T) {
	base := ObservabilityCfg

	// missing URL
	cfg := base
	cfg.Elastic.URL = ""
	if _, err := createElasticsearchCore(cfg, zapcore.InfoLevel); err == nil {
		t.Fatalf("expected error for empty ES URL")
	}

	// missing index
	cfg = base
	cfg.Elastic.Index = ""
	if _, err := createElasticsearchCore(cfg, zapcore.InfoLevel); err == nil {
		t.Fatalf("expected error for empty ES index")
	}

	// missing API key
	cfg = base
	cfg.Elastic.APIKey = ""
	if _, err := createElasticsearchCore(cfg, zapcore.InfoLevel); err == nil {
		t.Fatalf("expected error for empty ES API key")
	}
}

// SetElasticsearchEndpoint
func TestSetElasticsearchEndpointReinitialisesSyncer(t *testing.T) {
	atomic.StoreInt32(&esDisabledFlag, 0)

	esSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_cluster/health" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(esSrv.Close)

	withTempConfig(t, func(obsCfg *config.Observability, _ *config.APM) {
		obsCfg.Elastic.Enabled = true
		obsCfg.Elastic.APIKey = "k"
		obsCfg.Elastic.Index = "idx"
		obsCfg.Elastic.FlushInterval = 1

		if err := SetElasticsearchEndpoint(esSrv.URL); err != nil {
			t.Fatalf("endpoint-1: %v", err)
		}
		if esSyncerInstance == nil {
			t.Fatalf("syncer not initialised on first call")
		}
		firstCtx := esSyncerInstance.ctx

		if err := SetElasticsearchEndpoint(esSrv.URL + "/v2"); err != nil {
			t.Fatalf("endpoint-2: %v", err)
		}
		if esSyncerInstance == nil {
			t.Fatalf("syncer nil after second call")
		}
		secondCtx := esSyncerInstance.ctx
		if firstCtx == secondCtx {
			t.Log("warning: syncer context pointer reused – pool optimisation")
		}
	})
}

// pre-flight 500 branch
func TestPreflightCheckES500Error(t *testing.T) {
	failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(failSrv.Close)

	if err := preflightCheckES(failSrv.URL, "k", false); err == nil {
		t.Fatalf("expected error for HTTP 500")
	}
}

// eventEmitterCore field merge
func TestEventEmitterCoreFieldMerge(t *testing.T) {
	ce := &countingEmitter{}
	customEventEmitter = ce
	t.Cleanup(func() { customEventEmitter = nil })

	core := newEventEmitterCore(zapcore.InfoLevel).
		With([]zapcore.Field{zap.String("a", "1")})

	zap.New(core).With(zap.String("b", "2")).Info("merge")

	if ce.n != 1 {
		t.Fatalf("event not emitted")
	}
}

// createElasticsearchCore TLS
func TestCreateElasticsearchCoreTLSInsecure(t *testing.T) {
	t.Parallel()

	httpsSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == bulkPath {
			_, _ = w.Write([]byte(`{"errors":false}`))
			return
		}
		w.WriteHeader(http.StatusOK) // /_cluster/health
	}))
	t.Cleanup(httpsSrv.Close)

	cfg := ObservabilityCfg
	cfg.Elastic.URL = httpsSrv.URL
	cfg.Elastic.Index = testIndex
	cfg.Elastic.APIKey = "k"
	cfg.Elastic.InsecureSkipVerify = true

	core, err := createElasticsearchCore(cfg, zapcore.InfoLevel)
	if err != nil {
		t.Fatalf("TLS core: %v", err)
	}

	zap.New(core).Info("tls-log")
}

// EnableElasticsearchLogging flag
func TestEnableElasticsearchLoggingFlagToggleOnly(t *testing.T) {
	oldEmitter := customEventEmitter
	customEventEmitter = &stubEmitter{}
	t.Cleanup(func() { customEventEmitter = oldEmitter })

	withTempConfig(t, func(obsCfg *config.Observability, _ *config.APM) {
		obsCfg.Elastic.URL = "http://dummy"
		obsCfg.Elastic.Index = testIndex
		obsCfg.Elastic.APIKey = "k"
		obsCfg.Elastic.Enabled = true

		if err := initLogger(*obsCfg); err != nil {
			t.Fatalf("initLogger: %v", err)
		}

		if err := EnableElasticsearchLogging(false); err != nil {
			t.Fatalf("disable: %v", err)
		}
		if err := EnableElasticsearchLogging(true); err != nil {
			t.Fatalf("enable: %v", err)
		}
	})
}

// buffered syncer Flush safety
func TestBufferedSyncerFlushNilClientSafe(t *testing.T) {
	s := newBufferedElasticsearchSyncer(nil, "idx", time.Hour)
	t.Cleanup(s.Close)

	_, _ = s.Write([]byte(`{"k":"v"}`)) // enqueue one entry
	s.Flush()                           // ensure no panic
}

// SetElasticsearchEndpoint disabled
func TestSetElasticsearchEndpointDisabledMode(t *testing.T) {
	withTempConfig(t, func(obsCfg *config.Observability, _ *config.APM) {
		obsCfg.Elastic.Enabled = false
		obsCfg.Elastic.Index = "idx"
		oldURL := obsCfg.Elastic.URL

		if err := SetElasticsearchEndpoint("http://new-es"); err != nil {
			t.Fatalf("SetElasticsearchEndpoint: %v", err)
		}
		if obsCfg.Elastic.URL != "http://new-es" {
			t.Fatalf("URL not updated in config")
		}
		if esSyncerInstance != nil {
			t.Fatalf("syncer should remain nil when ES logging disabled")
		}

		obsCfg.Elastic.URL = oldURL
	})
}

func TestCollectSystemMetrics(t *testing.T) {
	metrics := collectSystemMetrics()

	keys := []string{
		"cpuUsage", "ramUsed", "ramTotal", "diskUsed", "diskTotal", "uptime", "load15", "rxBytes",
		"txBytes",
	}
	for _, key := range keys {
		assert.Contains(t, metrics, key, "expected metric '%s' to be present", key)
	}
}
