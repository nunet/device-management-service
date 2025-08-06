package observability

import (
	context "context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/natefinch/lumberjack"
	"github.com/olivere/elastic/v7"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/did"
)

const timestampKey = "timestamp"

// Global variables for observability control
var (
	EventBus           event.Bus
	customEventEmitter event.Emitter

	noOpMode         bool
	ObservabilityCfg config.Observability = config.DefaultConfig.Observability
	ApmCfg           config.APM           = config.DefaultConfig.APM
	mutex            sync.RWMutex
	combinedCore     zapcore.Core
	esSyncerInstance *bufferedElasticsearchSyncer
	atomicLevel      zap.AtomicLevel = zap.NewAtomicLevel()
	log                              = logging.Logger("observability")
	didID            did.DID
)

// CustomEvent represents a custom event structure
type CustomEvent struct {
	Name      string
	Timestamp time.Time
	Data      map[string]interface{}
}

var esDisabledFlag int32 // 0 => false, 1 => true

func disableES() {
	atomic.StoreInt32(&esDisabledFlag, 1)
}

func isESDisabled() bool {
	return atomic.LoadInt32(&esDisabledFlag) == 1
}

// Initialize sets up the logger, tracing, and event bus
func Initialize(host host.Host, did did.DID, cfg *config.Config) error {
	mutex.Lock()
	ObservabilityCfg = cfg.Observability
	ApmCfg = cfg.APM
	mutex.Unlock()

	if IsNoOpMode() {
		return nil
	}

	didID = did

	// Initialize the event bus
	if err := initEventBus(host); err != nil {
		return err
	}

	// Initialize the logger with configurations
	if err := initLogger(ObservabilityCfg); err != nil {
		// Non-fatal: we log a warning and proceed
		log.Warn("Failed to initialize logger", zap.Error(err))
	}

	// Initialize Elastic APM tracing only if the APM URL is provided
	if ApmCfg.ServerURL != "" {
		initTracing(ApmCfg)
	} else {
		log.Warn("APM Server URL not provided, tracing will be disabled")
	}

	return nil
}

// OverrideLoggerForTesting reconfigures the logger for unit tests
func OverrideLoggerForTesting() error {
	// Set observability to no-op mode for unit tests
	SetNoOpMode(true)
	return nil
}

// initLogger configures the global logger with console, file, Elasticsearch logging, and event emission
func initLogger(observabilityConfig config.Observability) error {
	// Acquire the lock only briefly
	mutex.Lock()
	localNoOp := noOpMode
	mutex.Unlock()

	// If we're in no-op mode, do nothing and return
	if localNoOp {
		return nil
	}

	// 1. Check if GOLOG_LOG_LEVEL is set. If so, let that override any config-based level
	if envLogLevel := os.Getenv("GOLOG_LOG_LEVEL"); envLogLevel != "" {
		observabilityConfig.LogLevel = envLogLevel
	}

	// 2. Parse the final log level string
	logLevel, err := parseLogLevel(observabilityConfig.LogLevel)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}
	atomicLevel.SetLevel(logLevel)

	// Close existing ES syncer if present
	if esSyncerInstance != nil {
		esSyncerInstance.Close()
		esSyncerInstance = nil
	}

	// Create console/file cores
	consoleCore := createConsoleCore(atomicLevel)
	fileCore := createFileCore(observabilityConfig, atomicLevel)

	var esCore zapcore.Core
	if observabilityConfig.ElasticsearchEnabled && !isESDisabled() {
		esCore, err = createElasticsearchCore(observabilityConfig, atomicLevel)
		if err != nil {
			log.Warn("Unable to create Elasticsearch logger (will disable ES logging).", zap.Error(err))
			disableES()
			esCore = nil
		}
	}

	// Create the event emitter core
	eventCore := newEventEmitterCore(atomicLevel)

	// Attach DID field
	didField := zap.String("did", didID.String())
	consoleCore = consoleCore.With([]zapcore.Field{didField})
	fileCore = fileCore.With([]zapcore.Field{didField})
	if esCore != nil {
		esCore = esCore.With([]zapcore.Field{didField})
	}
	eventCore = eventCore.With([]zapcore.Field{didField})

	// Combine the cores into a Tee
	cores := []zapcore.Core{consoleCore, fileCore, eventCore}
	if esCore != nil {
		cores = append(cores, esCore)
	}
	baseTee := zapcore.NewTee(cores...)

	// Wrap the combined tee with our label injection core
	labelInjectedCore := newLabelInjectionCore(baseTee, atomicLevel)

	// Lock again to replace global references
	mutex.Lock()
	defer mutex.Unlock()

	combinedCore = labelInjectedCore
	logging.SetPrimaryCore(combinedCore)

	return nil
}

// parseLogLevel parses a string into a zapcore.Level
func parseLogLevel(levelStr string) (zapcore.Level, error) {
	var level zapcore.Level
	err := level.UnmarshalText([]byte(levelStr))
	if err != nil {
		return 0, err
	}
	return level, nil
}

// createConsoleCore creates a console logging core
func createConsoleCore(levelEnabler zapcore.LevelEnabler) zapcore.Core {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = timestampKey
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
	consoleWS := zapcore.AddSync(os.Stdout)
	return zapcore.NewCore(consoleEncoder, consoleWS, levelEnabler)
}

// createFileCore creates a file logging core
func createFileCore(observabilityConfig config.Observability, levelEnabler zapcore.LevelEnabler) zapcore.Core {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = timestampKey
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	fileEncoder := zapcore.NewJSONEncoder(encoderConfig)
	fileWS := zapcore.AddSync(&lumberjack.Logger{
		Filename:   observabilityConfig.LogFile,
		MaxSize:    observabilityConfig.MaxSize,    // in MB
		MaxBackups: observabilityConfig.MaxBackups, // number of backups
		MaxAge:     observabilityConfig.MaxAge,     // in days
		Compress:   true,
	})

	return zapcore.NewCore(fileEncoder, fileWS, levelEnabler)
}

// createElasticsearchCore creates an Elasticsearch logging core with "preflight" fallback
func createElasticsearchCore(observabilityConfig config.Observability, levelEnabler zapcore.LevelEnabler) (zapcore.Core, error) {
	// Basic validations
	if observabilityConfig.ElasticsearchURL == "" {
		return nil, fmt.Errorf("elasticsearch URL is not configured")
	}
	if observabilityConfig.ElasticsearchIndex == "" {
		return nil, fmt.Errorf("elasticsearch index is not configured")
	}
	if observabilityConfig.ElasticsearchAPIKey == "" {
		return nil, fmt.Errorf("elasticsearch API key is not configured")
	}

	// Attempt to build the WriteSyncer
	esWS, err := newElasticsearchWriteSyncer(
		observabilityConfig.ElasticsearchURL,
		observabilityConfig.ElasticsearchIndex,
		time.Duration(observabilityConfig.FlushInterval)*time.Second,
		observabilityConfig.ElasticsearchAPIKey,
		observabilityConfig.InsecureSkipVerify,
	)
	if err != nil {
		return nil, err
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = timestampKey
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	esEncoder := zapcore.NewJSONEncoder(encoderConfig)
	return zapcore.NewCore(esEncoder, esWS, levelEnabler), nil
}

// newElasticsearchWriteSyncer creates a WriteSyncer for Elasticsearch with buffering
func newElasticsearchWriteSyncer(
	url string,
	index string,
	flushInterval time.Duration,
	apiKey string,
	insecureSkipVerify bool,
) (zapcore.WriteSyncer, error) {
	// 1) Short preflight check to ensure ES is reachable
	if err := preflightCheckES(url, apiKey, insecureSkipVerify); err != nil {
		return nil, fmt.Errorf("ES preflight: %w", err)
	}

	// 2) Build the actual transport + client
	dialer := &net.Dialer{
		Timeout: 3 * time.Second,
	}
	tlsConfig := &tls.Config{InsecureSkipVerify: insecureSkipVerify}

	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext:         dialer.DialContext,
			TLSClientConfig:     tlsConfig,
			DisableKeepAlives:   false,
			MaxIdleConns:        10,
			IdleConnTimeout:     30 * time.Second,
			TLSHandshakeTimeout: 3 * time.Second,
		},
		Timeout: 5 * time.Second,
	}

	clientOptions := []elastic.ClientOptionFunc{
		elastic.SetURL(url),
		elastic.SetHttpClient(httpClient),
		elastic.SetSniff(false),
		elastic.SetHealthcheck(false),
	}

	if apiKey != "" {
		clientOptions = append(clientOptions, elastic.SetHeaders(http.Header{
			"Authorization": []string{"ApiKey " + apiKey},
		}))
	}

	client, err := elastic.NewClient(clientOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create elastic client: %v", err)
	}

	esSyncer := newBufferedElasticsearchSyncer(client, index, flushInterval)
	esSyncerInstance = esSyncer
	return esSyncer, nil
}

// preflightCheckES does a quick GET /_cluster/health to ensure ES is reachable
func preflightCheckES(url, apiKey string, insecureSkip bool) error {
	preflightURL := url + "/_cluster/health"
	log.Infow("Preflight: Attempting to initialize Elasticsearch",
		"url", url,
		"apiKey", apiKey,
	)

	testCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(testCtx, http.MethodGet, preflightURL, nil)
	if err != nil {
		return fmt.Errorf("preflight request creation failed: %w", err)
	}
	req.Header.Set("Authorization", "ApiKey "+apiKey)

	ephemeralTransport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 2 * time.Second,
		}).DialContext,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: insecureSkip},
		DisableKeepAlives:   true,
		MaxIdleConns:        2,
		IdleConnTimeout:     2 * time.Second,
		TLSHandshakeTimeout: 2 * time.Second,
	}

	ephemeralClient := &http.Client{
		Transport: ephemeralTransport,
		Timeout:   5 * time.Second,
	}

	resp, err := ephemeralClient.Do(req)
	if err != nil {
		log.Warnw("Elasticsearch preflight request failed", "err", err, "url", url)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		log.Warnw("Elasticsearch preflight returned 401 Unauthorized", "url", url)
		return fmt.Errorf("invalid credentials (401 unauthorized)")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Warnw("Elasticsearch preflight returned unexpected status",
			"statusCode", resp.StatusCode,
			"url", url,
		)
		return fmt.Errorf("got unexpected HTTP %d during ES preflight", resp.StatusCode)
	}

	log.Infow("Elasticsearch preflight succeeded", "url", url, "statusCode", resp.StatusCode)
	return nil
}

// bufferedElasticsearchSyncer implements zapcore.WriteSyncer to send logs to ES with buffering
type bufferedElasticsearchSyncer struct {
	client        *elastic.Client
	index         string
	ctx           context.Context
	cancelFunc    context.CancelFunc
	buffer        []string
	bufferMutex   sync.Mutex
	flushInterval time.Duration
	lastErrorTime time.Time
	errorCount    int
	warnLogged    bool
	maxBufferSize int
	wg            sync.WaitGroup
}

func newBufferedElasticsearchSyncer(client *elastic.Client, index string, flushInterval time.Duration) *bufferedElasticsearchSyncer {
	ctx, cancel := context.WithCancel(context.Background())
	syncer := &bufferedElasticsearchSyncer{
		client:        client,
		index:         index,
		ctx:           ctx,
		cancelFunc:    cancel,
		buffer:        make([]string, 0),
		flushInterval: flushInterval,
		maxBufferSize: 1000,
	}
	syncer.wg.Add(1)
	go syncer.start()
	return syncer
}

func (b *bufferedElasticsearchSyncer) start() {
	ticker := time.NewTicker(b.flushInterval)
	defer ticker.Stop()
	defer b.wg.Done()

	for {
		select {
		case <-ticker.C:
			b.Flush()
		case <-b.ctx.Done():
			return
		}
	}
}

// Write buffers the log entry
func (b *bufferedElasticsearchSyncer) Write(p []byte) (n int, err error) {
	b.bufferMutex.Lock()
	defer b.bufferMutex.Unlock()

	if len(b.buffer) >= b.maxBufferSize {
		if !b.warnLogged {
			b.warnLogged = true
		}
		return 0, fmt.Errorf("log buffer is full")
	}
	b.buffer = append(b.buffer, string(p))
	return len(p), nil
}

// Sync flushes the buffer to Elasticsearch
func (b *bufferedElasticsearchSyncer) Sync() error {
	b.Flush()
	return nil
}

// Flush sends the buffered log entries to Elasticsearch with advanced routing
// but never sends the same log to both the override index and the default index.
func (b *bufferedElasticsearchSyncer) Flush() {
	// Lock only the buffer mutex
	b.bufferMutex.Lock()
	defer b.bufferMutex.Unlock()

	if len(b.buffer) == 0 {
		return
	}

	if b.client == nil {
		if !b.warnLogged {
			b.warnLogged = true
		}
		return
	}

	// Copy and clear the buffer so we can release the lock sooner
	bufferCopy := b.buffer
	b.buffer = nil

	bulkRequest := b.client.Bulk()

	for _, logEntry := range bufferCopy {
		// Parse the JSON to check "es_skip" and "es_index"
		var record map[string]interface{}
		if err := json.Unmarshal([]byte(logEntry), &record); err != nil {
			// If parsing fails, skip or store as-is. We'll skip to avoid malformed JSON in ES.
			continue
		}

		// Skip if "es_skip" == true
		if skipVal, ok := record["es_skip"].(bool); ok && skipVal {
			continue
		}

		// If "es_index" is set, store in that index
		if overrideIndex, ok := record["es_index"].(string); ok && overrideIndex != "" {
			req := elastic.NewBulkIndexRequest().Index(overrideIndex).Doc(record)
			bulkRequest = bulkRequest.Add(req)
		} else {
			// Otherwise, store in the default index
			req := elastic.NewBulkIndexRequest().Index(b.index).Doc(record)
			bulkRequest = bulkRequest.Add(req)
		}
	}

	flushCtx, cancel := context.WithTimeout(b.ctx, 3*time.Second)
	defer cancel()

	_, err := bulkRequest.Do(flushCtx)
	if err != nil {
		// If it's a 401, disable ES immediately (atomic, no global lock)
		if esErr, ok := err.(*elastic.Error); ok && esErr.Status == 401 {
			disableES()
			return
		}

		now := time.Now()
		// Throttle repeated warnings
		if b.errorCount == 0 || now.Sub(b.lastErrorTime) > 5*time.Minute {
			b.lastErrorTime = now
			b.errorCount = 1
		} else {
			b.errorCount++
		}

		// If it fails too many times in a short window, disable ES
		if b.errorCount >= 3 {
			disableES()
		}
	} else {
		b.errorCount = 0
	}
}

func (b *bufferedElasticsearchSyncer) Close() {
	b.cancelFunc()
	b.wg.Wait()
	b.Flush()
}

func (b *bufferedElasticsearchSyncer) setFlushInterval(interval time.Duration) {
	b.cancelFunc()
	b.wg.Wait()

	b.ctx, b.cancelFunc = context.WithCancel(context.Background())
	b.flushInterval = interval

	b.wg.Add(1)
	go b.start()
}

// SetElasticsearchEndpoint updates the Elasticsearch URL and reinitializes the logger.
func SetElasticsearchEndpoint(url string) error {
	mutex.Lock()
	ObservabilityCfg.ElasticsearchURL = url
	mutex.Unlock()

	err := initLogger(ObservabilityCfg)
	if err != nil {
		return fmt.Errorf("failed to reinitialize logger after updating ES endpoint: %v", err)
	}
	return nil
}

// initEventBus initializes the global event bus
func initEventBus(host host.Host) error {
	EventBus = host.EventBus()
	var err error
	customEventEmitter, err = EventBus.Emitter(new(CustomEvent))
	if err != nil {
		return fmt.Errorf("failed to create custom event emitter: %w", err)
	}
	return nil
}

// newEventEmitterCore creates a zapcore.Core that emits log entries to the event bus
func newEventEmitterCore(levelEnabler zapcore.LevelEnabler) zapcore.Core {
	return &eventEmitterCore{
		LevelEnabler: levelEnabler,
	}
}

// eventEmitterCore is a zapcore.Core that emits log entries to the event bus
type eventEmitterCore struct {
	zapcore.LevelEnabler
	fields []zapcore.Field
}

func (e *eventEmitterCore) With(fields []zapcore.Field) zapcore.Core {
	return &eventEmitterCore{
		LevelEnabler: e.LevelEnabler,
		fields:       append(e.fields, fields...),
	}
}

func (e *eventEmitterCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if e.LevelEnabler.Enabled(entry.Level) {
		return ce.AddCore(entry, e)
	}
	return ce
}

func (e *eventEmitterCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	if IsNoOpMode() {
		return nil
	}

	eventData := map[string]interface{}{
		"level":     entry.Level.String(),
		"timestamp": entry.Time.UTC().Format(time.RFC3339),
		"message":   entry.Message,
		"logger":    entry.LoggerName,
		"caller":    entry.Caller.String(),
	}

	for _, field := range fields {
		eventData[field.Key] = field.Interface
	}

	customEvent := CustomEvent{
		Name:      "log_event",
		Timestamp: entry.Time,
		Data:      eventData,
	}

	// nil-guard: only emit if the bus is initialised
	if customEventEmitter != nil {
		_ = customEventEmitter.Emit(customEvent)
	}

	return nil
}

func (e *eventEmitterCore) Sync() error {
	return nil
}

// SetLogLevel sets the global log level for all collectors
func SetLogLevel(level string) error {
	mutex.Lock()
	defer mutex.Unlock()

	logLevel, err := parseLogLevel(level)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}

	ObservabilityCfg.LogLevel = level
	atomicLevel.SetLevel(logLevel)

	return nil
}

// SetFlushInterval sets the flush interval for Elasticsearch logging dynamically
func SetFlushInterval(seconds int) error {
	mutex.Lock()
	ObservabilityCfg.FlushInterval = seconds
	localEsSyncer := esSyncerInstance
	mutex.Unlock()

	if localEsSyncer != nil {
		localEsSyncer.setFlushInterval(time.Duration(seconds) * time.Second)
	}
	return nil
}

// EmitCustomEvent allows developers to emit custom events with variadic key-value pairs
func EmitCustomEvent(eventName string, keyValues ...interface{}) error {
	if len(keyValues)%2 != 0 {
		return fmt.Errorf("keyValues must be in key-value pairs")
	}
	eventData := make(map[string]interface{})
	for i := 0; i < len(keyValues); i += 2 {
		key, ok := keyValues[i].(string)
		if !ok {
			return fmt.Errorf("key must be a string")
		}
		eventData[key] = keyValues[i+1]
	}

	customEvent := CustomEvent{
		Name:      eventName,
		Timestamp: time.Now(),
		Data:      eventData,
	}

	if err := customEventEmitter.Emit(customEvent); err != nil {
		log.Debug("Failed to emit custom event", zap.Error(err))
	}
	return nil
}

// Shutdown cleans up resources
func Shutdown() {
	mutex.Lock()
	if customEventEmitter != nil {
		customEventEmitter.Close()
	}
	if esSyncerInstance != nil {
		esSyncerInstance.Close()
		esSyncerInstance = nil
	}
	mutex.Unlock()

	// Shutdown the tracer
	ShutdownTracer()
}

// ShutdownTracer wraps the Shutdown function from tracing.go
func ShutdownTracer() {
	shutdownTracer()
}

// SetNoOpMode enables or disables the no-op mode for observability.
func SetNoOpMode(enabled bool) {
	mutex.Lock()
	noOpMode = enabled
	if noOpMode {
		// If in no-op mode, set the log level to a very high threshold
		atomicLevel.SetLevel(zapcore.Level(100))
	} else {
		logLevel, err := parseLogLevel(ObservabilityCfg.LogLevel)
		if err != nil {
			logLevel = zapcore.InfoLevel
		}
		atomicLevel.SetLevel(logLevel)
	}
	mutex.Unlock()
}

// IsNoOpMode returns whether observability is in no-op mode.
func IsNoOpMode() bool {
	mutex.RLock()
	defer mutex.RUnlock()
	return noOpMode
}

// SetAPIKey updates the API key for both Elasticsearch and APM.
func SetAPIKey(apiKey string) error {
	mutex.Lock()
	ObservabilityCfg.ElasticsearchAPIKey = apiKey
	ApmCfg.APIKey = apiKey
	mutex.Unlock()

	// Reinit logger outside the lock
	err := initLogger(ObservabilityCfg)
	if err != nil {
		return fmt.Errorf("failed to reinitialize logger: %v", err)
	}

	if ApmCfg.ServerURL != "" {
		initTracing(ApmCfg)
	}

	return nil
}

// SetAPMURL updates the APM server URL and reinitializes the APM tracer.
func SetAPMURL(url string) error {
	mutex.Lock()
	if noOpMode {
		mutex.Unlock()
		return nil
	}
	ApmCfg.ServerURL = url
	mutex.Unlock()

	if ApmCfg.ServerURL != "" {
		initTracing(ApmCfg)
	} else {
		ShutdownTracer()
	}
	return nil
}

// EnableElasticsearchLogging enables or disables Elasticsearch logging dynamically.
func EnableElasticsearchLogging(enabled bool) error {
	mutex.Lock()
	ObservabilityCfg.ElasticsearchEnabled = enabled
	mutex.Unlock()

	// Attempt reinit
	err := initLogger(ObservabilityCfg)
	if err != nil {
		return fmt.Errorf("failed to reinitialize logger: %v", err)
	}

	return nil
}
