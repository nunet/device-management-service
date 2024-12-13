// observability.go
// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package observability

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"sync"
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

var (
	// EventBus is the global event bus instance
	EventBus event.Bus

	// customEventEmitter is the emitter for CustomEvent
	customEventEmitter event.Emitter

	// Global variables for observability control
	noOpMode bool
	mutex    sync.RWMutex

	// Global variables to hold references for dynamic updates
	combinedCore     zapcore.Core
	esSyncerInstance *bufferedElasticsearchSyncer
	atomicLevel      zap.AtomicLevel = zap.NewAtomicLevel()

	// Logger for the observability package
	log = logging.Logger("observability")

	// Global variable to hold the DID
	didID did.DID
)

// CustomEvent represents a custom event structure
type CustomEvent struct {
	Name      string
	Timestamp time.Time
	Data      map[string]interface{}
}

// Initialize sets up the logger, tracing, and event bus
func Initialize(host host.Host, did did.DID, cfg *config.Config) error {
	if IsNoOpMode() {
		return nil
	}
	// Store the DID
	didID = did

	// Initialize the event bus
	if err := initEventBus(host); err != nil {
		return err
	}

	// Initialize the logger with configurations
	if err := initLogger(cfg.Observability); err != nil {
		log.Warn("Failed to initialize logger", zap.Error(err))
	}

	// Initialize Elastic APM tracing only if the APM URL is provided
	if cfg.APM.ServerURL != "" {
		initTracing(cfg.APM)
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
	mutex.Lock()
	defer mutex.Unlock()

	// Parse the global log level
	logLevel, err := parseLogLevel(observabilityConfig.LogLevel)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}

	// Set the atomic level
	atomicLevel.SetLevel(logLevel)

	// Before replacing the global logger, flush and close existing cores if necessary
	if esSyncerInstance != nil {
		esSyncerInstance.Close()
		esSyncerInstance = nil
	}

	// Create cores, passing atomicLevel as LevelEnabler
	consoleCore := createConsoleCore(atomicLevel)
	fileCore := createFileCore(observabilityConfig, atomicLevel)

	var esCore zapcore.Core
	if observabilityConfig.ElasticsearchEnabled {
		esCore, err = createElasticsearchCore(observabilityConfig, atomicLevel)
		if err != nil {
			log.Warn("Unable to create Elasticsearch logger", zap.Error(err))
			esCore = nil // Proceed without Elasticsearch core
		}
	}

	eventCore := newEventEmitterCore(atomicLevel)

	// Wrap cores with the DID field
	didField := zap.String("did", didID.String())
	consoleCore = consoleCore.With([]zapcore.Field{didField})
	fileCore = fileCore.With([]zapcore.Field{didField})
	if esCore != nil {
		esCore = esCore.With([]zapcore.Field{didField})
	}
	eventCore = eventCore.With([]zapcore.Field{didField})

	// Combine cores, excluding nil cores
	var cores []zapcore.Core
	cores = append(cores, consoleCore, fileCore)
	if esCore != nil {
		cores = append(cores, esCore)
	}
	cores = append(cores, eventCore)
	combinedCore = zapcore.NewTee(cores...)

	// Replace the global logger
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
		MaxSize:    observabilityConfig.MaxSize, // megabytes
		MaxBackups: observabilityConfig.MaxBackups,
		MaxAge:     observabilityConfig.MaxAge, // days
		Compress:   true,
	})

	return zapcore.NewCore(fileEncoder, fileWS, levelEnabler)
}

// createElasticsearchCore creates an Elasticsearch logging core
func createElasticsearchCore(observabilityConfig config.Observability, levelEnabler zapcore.LevelEnabler) (zapcore.Core, error) {
	// Validate necessary configurations
	if observabilityConfig.ElasticsearchURL == "" {
		return nil, fmt.Errorf("elasticsearch URL is not configured")
	}
	if observabilityConfig.ElasticsearchIndex == "" {
		return nil, fmt.Errorf("elasticsearch index is not configured")
	}
	// If Elasticsearch requires an API key, check for it
	if observabilityConfig.ElasticsearchAPIKey == "" {
		return nil, fmt.Errorf("elasticsearch API key is not configured")
	}

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
func newElasticsearchWriteSyncer(url string, index string, flushInterval time.Duration, apiKey string, insecureSkipVerify bool) (zapcore.WriteSyncer, error) {
	// Create TLS configuration
	tlsConfig := &tls.Config{
		InsecureSkipVerify: insecureSkipVerify, // WARNING: defaults to true Only for testing purposes
	}
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	// Prepare client options
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

	// Create Elasticsearch client
	client, err := elastic.NewClient(clientOptions...)
	if err != nil {
		return nil, err
	}

	esSyncer := newBufferedElasticsearchSyncer(client, index, flushInterval)

	// Store the instance globally for dynamic updates
	esSyncerInstance = esSyncer

	return esSyncer, nil
}

// bufferedElasticsearchSyncer implements zapcore.WriteSyncer to send logs to Elasticsearch with buffering
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

// newBufferedElasticsearchSyncer creates a new bufferedElasticsearchSyncer
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

	// Start the flush goroutine
	go syncer.start()

	return syncer
}

// start begins the periodic flushing of the buffer
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
		// Handle buffer full scenario, drop the log entry and log a warning
		if !b.warnLogged {
			log.Warn("Elasticsearch log buffer is full, dropping log entries")
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

// Flush sends the buffered log entries to Elasticsearch
func (b *bufferedElasticsearchSyncer) Flush() {
	b.bufferMutex.Lock()
	defer b.bufferMutex.Unlock()

	if len(b.buffer) == 0 {
		return
	}

	if b.client == nil {
		if !b.warnLogged {
			log.Warn("Elasticsearch client is not initialized, cannot flush logs")
			b.warnLogged = true
		}
		return
	}

	// Reset warnLogged if client becomes available
	b.warnLogged = false
	bufferCopy := b.buffer
	b.buffer = make([]string, 0)

	bulkRequest := b.client.Bulk()
	for _, logEntry := range bufferCopy {
		req := elastic.NewBulkIndexRequest().Index(b.index).Doc(logEntry)
		bulkRequest = bulkRequest.Add(req)
	}

	_, err := bulkRequest.Do(b.ctx)
	if err != nil {
		// Implement error suppression
		now := time.Now()
		if b.errorCount == 0 || now.Sub(b.lastErrorTime) > 5*time.Minute {
			log.Warn("Error flushing logs to Elasticsearch", zap.Error(err))
			b.lastErrorTime = now
			b.errorCount = 1
		} else {
			b.errorCount++
		}
	} else {
		// Reset error count on successful flush
		b.errorCount = 0
	}
}

// Close stops the flush goroutine and flushes remaining logs
func (b *bufferedElasticsearchSyncer) Close() {
	b.cancelFunc()
	b.wg.Wait()
	b.Flush()
}

// setFlushInterval allows changing the flush interval dynamically
func (b *bufferedElasticsearchSyncer) setFlushInterval(interval time.Duration) {
	// Cancel the existing context to stop the goroutine
	b.cancelFunc()
	b.wg.Wait()

	// Create a new context and start a new goroutine
	b.ctx, b.cancelFunc = context.WithCancel(context.Background())
	b.flushInterval = interval

	// Start the flush goroutine with the new interval
	go b.start()
}

// SetElasticsearchEndpoint updates the Elasticsearch URL and reinitializes the logger.
func SetElasticsearchEndpoint(url string) error {
	mutex.Lock()
	defer mutex.Unlock()

	// Update the configuration
	cfg := config.GetConfig()
	cfg.Observability.ElasticsearchURL = url

	// Reinitialize the logger to apply the new URL
	err := initLogger(cfg.Observability)
	if err != nil {
		return fmt.Errorf("failed to reinitialize logger: %v", err)
	}

	return nil
}

// initEventBus initializes the global event bus
func initEventBus(host host.Host) error {
	EventBus = host.EventBus()

	// Create an emitter for CustomEvent
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

// With implements zapcore.Core
func (e *eventEmitterCore) With(fields []zapcore.Field) zapcore.Core {
	return &eventEmitterCore{
		LevelEnabler: e.LevelEnabler,
		fields:       append(e.fields, fields...),
	}
}

// Check implements zapcore.Core
func (e *eventEmitterCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if e.Enabled(entry.Level) {
		return ce.AddCore(entry, e)
	}
	return ce
}

// Write implements zapcore.Core
func (e *eventEmitterCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	if IsNoOpMode() {
		return nil
	}

	// Convert the log entry and fields into a map[string]interface{}
	eventData := map[string]interface{}{
		"level":     entry.Level.String(),
		"timestamp": entry.Time.UTC().Format(time.RFC3339),
		"message":   entry.Message,
		"logger":    entry.LoggerName,
		"caller":    entry.Caller.String(),
	}

	// Add fields
	for _, field := range fields {
		eventData[field.Key] = field.Interface
	}

	// Create a CustomEvent
	customEvent := CustomEvent{
		Name:      "log_event",
		Timestamp: entry.Time,
		Data:      eventData,
	}

	// Emit the event using the customEventEmitter
	if err := customEventEmitter.Emit(customEvent); err != nil {
		// Log the error at Debug level to avoid cluttering
		log.Debug("Error emitting event", zap.Error(err))
	}

	return nil
}

// Sync implements zapcore.Core
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

	// Update the configuration
	cfg := config.GetConfig()
	cfg.Observability.LogLevel = level

	// Set the new log level in atomicLevel
	atomicLevel.SetLevel(logLevel)

	return nil
}

// SetFlushInterval sets the flush interval for Elasticsearch logging dynamically
func SetFlushInterval(seconds int) error {
	mutex.Lock()
	defer mutex.Unlock()

	// Update the configuration
	cfg := config.GetConfig()
	cfg.Observability.FlushInterval = seconds

	// Update the flush interval in the elasticsearchWriteSyncer
	if esSyncerInstance != nil {
		esSyncerInstance.setFlushInterval(time.Duration(seconds) * time.Second)
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

	// Create the custom event
	customEvent := CustomEvent{
		Name:      eventName,
		Timestamp: time.Now(),
		Data:      eventData,
	}

	// Emit the event using the customEventEmitter
	if err := customEventEmitter.Emit(customEvent); err != nil {
		log.Debug("Failed to emit custom event", zap.Error(err))
	}
	return nil
}

// Shutdown cleans up resources
func Shutdown() {
	mutex.Lock()
	defer mutex.Unlock()

	if customEventEmitter != nil {
		customEventEmitter.Close()
	}
	if esSyncerInstance != nil {
		esSyncerInstance.Close()
		esSyncerInstance = nil
	}
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
	defer mutex.Unlock()
	noOpMode = enabled

	if noOpMode {
		// Disable logging by setting the level to a higher level than Panic
		atomicLevel.SetLevel(zapcore.Level(100)) // An arbitrarily high level
	} else {
		// Restore the log level from configuration
		cfg := config.GetConfig()
		logLevel, err := parseLogLevel(cfg.Observability.LogLevel)
		if err != nil {
			logLevel = zapcore.InfoLevel
		}
		atomicLevel.SetLevel(logLevel)
	}
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
	defer mutex.Unlock()

	// Update the configuration
	cfg := config.GetConfig()
	cfg.Observability.ElasticsearchAPIKey = apiKey
	cfg.APM.APIKey = apiKey

	// Reinitialize the logger to apply the new API key for Elasticsearch
	err := initLogger(cfg.Observability)
	if err != nil {
		return fmt.Errorf("failed to reinitialize logger: %v", err)
	}

	// Reinitialize tracing to apply the new API key for APM
	if cfg.APM.ServerURL != "" {
		initTracing(cfg.APM)
	}

	return nil
}

// SetAPMURL updates the APM server URL and reinitializes the APM tracer.
func SetAPMURL(url string) error {
	mutex.Lock()
	defer mutex.Unlock()

	if IsNoOpMode() {
		return nil
	}

	// Update the configuration
	cfg := config.GetConfig()
	cfg.APM.ServerURL = url

	// Reinitialize the tracer with the updated configuration
	if cfg.APM.ServerURL != "" {
		initTracing(cfg.APM)
	} else {
		ShutdownTracer()
	}

	return nil
}

// EnableElasticsearchLogging enables or disables Elasticsearch logging dynamically.
func EnableElasticsearchLogging(enabled bool) error {
	mutex.Lock()
	defer mutex.Unlock()

	// Update the configuration
	cfg := config.GetConfig()
	cfg.Observability.ElasticsearchEnabled = enabled

	// Reinitialize the logger to apply the change
	err := initLogger(cfg.Observability)
	if err != nil {
		return fmt.Errorf("failed to reinitialize logger: %v", err)
	}

	return nil
}
