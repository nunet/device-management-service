package observability

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/olivere/elastic/v7"
	"gitlab.com/nunet/device-management-service/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const timestampKey = "timestamp"

var (
	// EventBus is the global event bus instance
	EventBus event.Bus

	// customEventEmitter is the emitter for CustomEvent
	customEventEmitter event.Emitter
)

// CustomEvent represents a custom event structure
type CustomEvent struct {
	Name      string
	Timestamp time.Time
	Data      map[string]interface{}
}

// Initialize sets up the logger, tracing, and event bus
func Initialize(host host.Host) error {
	if isNoOp() {
		return nil
	}

	// Load the configuration
	cfg := config.GetConfig()

	// Initialize the event bus
	if err := initEventBus(host); err != nil {
		return err
	}

	// Initialize the logger with configurations
	if err := initLogger(cfg.Observability); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize logger: %v\n", err)
	}
	// Initialize Elastic APM tracing
	if err := initTracing(cfg.APM); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize tracing: %v\n", err)
	}

	return nil
}

// OverrideLoggerForTesting reconfigures the logger to log only to console
func OverrideLoggerForTesting() error {
	// Use the existing configuration
	cfg := config.GetConfig()

	// Parse the global log level
	logLevel, err := parseLogLevel(cfg.Observability.LogLevel)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}

	// Reconfigure the logger to log only to console
	consoleCore := createConsoleCore(logLevel)
	combinedCore = zapcore.NewTee(consoleCore)

	// Replace the global logger
	logging.SetPrimaryCore(combinedCore)

	return nil
}

// Global variables to hold references to cores for dynamic updates
var (
	combinedCore     zapcore.Core
	esSyncerInstance *bufferedElasticsearchSyncer
)

// initLogger configures the global logger with console, file, Elasticsearch logging, and event emission
func initLogger(observabilityConfig config.Observability) error {
	// Parse the global log level
	logLevel, err := parseLogLevel(observabilityConfig.LogLevel)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}

	// Create cores
	consoleCore := createConsoleCore(logLevel)
	fileCore := createFileCore(observabilityConfig, logLevel)
	esCore, err := createElasticsearchCore(observabilityConfig, logLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Unable to create Elasticsearch logger: %v\n", err)
		esCore = nil // Proceed without Elasticsearch core
	}
	eventCore := newEventEmitterCore(logLevel)

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
func createConsoleCore(logLevel zapcore.Level) zapcore.Core {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = timestampKey
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
	consoleWS := zapcore.AddSync(os.Stdout)

	return zapcore.NewCore(consoleEncoder, consoleWS, logLevel)
}

// createFileCore creates a file logging core
func createFileCore(observabilityConfig config.Observability, logLevel zapcore.Level) zapcore.Core {
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

	return zapcore.NewCore(fileEncoder, fileWS, logLevel)
}

// createElasticsearchCore creates an Elasticsearch logging core
func createElasticsearchCore(observabilityConfig config.Observability, logLevel zapcore.Level) (zapcore.Core, error) {
	esWS, err := newElasticsearchWriteSyncer(
		observabilityConfig.ElasticsearchURL,
		observabilityConfig.ElasticsearchIndex,
		time.Duration(observabilityConfig.FlushInterval)*time.Second,
	)
	if err != nil {
		return nil, err
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = timestampKey
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	esEncoder := zapcore.NewJSONEncoder(encoderConfig)

	return zapcore.NewCore(esEncoder, esWS, logLevel), nil
}

// newElasticsearchWriteSyncer creates a WriteSyncer for Elasticsearch with buffering
func newElasticsearchWriteSyncer(url string, index string, flushInterval time.Duration) (zapcore.WriteSyncer, error) {
	// Create Elasticsearch client
	client, err := elastic.NewClient(
		elastic.SetURL(url),
		elastic.SetSniff(false),       // Disable sniffing if not using a cluster
		elastic.SetHealthcheck(false), // Disable initial health check
	)
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
	buffer        []string
	bufferMutex   sync.Mutex
	flushInterval time.Duration
	ticker        *time.Ticker
	done          chan struct{}
}

// newBufferedElasticsearchSyncer creates a new bufferedElasticsearchSyncer
func newBufferedElasticsearchSyncer(client *elastic.Client, index string, flushInterval time.Duration) *bufferedElasticsearchSyncer {
	syncer := &bufferedElasticsearchSyncer{
		client:        client,
		index:         index,
		ctx:           context.Background(),
		buffer:        make([]string, 0),
		flushInterval: flushInterval,
		done:          make(chan struct{}),
	}

	// Start the flush ticker
	syncer.ticker = time.NewTicker(syncer.flushInterval)
	go syncer.start()

	return syncer
}

// start begins the periodic flushing of the buffer
func (b *bufferedElasticsearchSyncer) start() {
	for {
		select {
		case <-b.ticker.C:
			b.Flush()
		case <-b.done:
			return
		}
	}
}

// Write buffers the log entry
func (b *bufferedElasticsearchSyncer) Write(p []byte) (n int, err error) {
	b.bufferMutex.Lock()
	b.buffer = append(b.buffer, string(p))
	b.bufferMutex.Unlock()
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
	bufferCopy := b.buffer
	b.buffer = make([]string, 0)
	b.bufferMutex.Unlock()

	if len(bufferCopy) == 0 {
		return
	}

	bulkRequest := b.client.Bulk()
	for _, logEntry := range bufferCopy {
		req := elastic.NewBulkIndexRequest().Index(b.index).Doc(logEntry)
		bulkRequest = bulkRequest.Add(req)
	}

	_, err := bulkRequest.Do(b.ctx)
	if err != nil {
		// Handle the error (e.g., log it)
		fmt.Fprintf(os.Stderr, "Error flushing logs to Elasticsearch: %v\n", err)
	}
}

// Close stops the ticker and flushes remaining logs
func (b *bufferedElasticsearchSyncer) Close() {
	b.ticker.Stop()
	close(b.done)
	b.Flush()
}

// setFlushInterval allows changing the flush interval dynamically
func (b *bufferedElasticsearchSyncer) setFlushInterval(interval time.Duration) {
	b.ticker.Stop()
	b.flushInterval = interval
	b.ticker = time.NewTicker(b.flushInterval)
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
func newEventEmitterCore(level zapcore.LevelEnabler) zapcore.Core {
	return &eventEmitterCore{
		LevelEnabler: level,
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
	customEvent := &CustomEvent{
		Name:      "log_event",
		Timestamp: entry.Time,
		Data:      eventData,
	}

	// Emit the event using the customEventEmitter
	if err := customEventEmitter.Emit(customEvent); err != nil {
		// Handle error if necessary
		fmt.Fprintf(os.Stderr, "Error emitting event: %v\n", err)
	}

	return nil
}

// Sync implements zapcore.Core
func (e *eventEmitterCore) Sync() error {
	return nil
}

// SetLogLevel sets the global log level for all collectors
func SetLogLevel(level string) error {
	_, err := parseLogLevel(level)
	if err != nil {
		return err
	}

	// Update the configuration
	cfg := config.GetConfig()
	cfg.Observability.LogLevel = level

	// Rebuild the logger with the new log level
	return rebuildLogger(cfg.Observability)
}

// rebuildLogger rebuilds the combined core and updates the global logger
func rebuildLogger(observabilityConfig config.Observability) error {
	// Re-initialize the logger with the updated configuration
	return initLogger(observabilityConfig)
}

// SetFlushInterval sets the flush interval for Elasticsearch logging dynamically
func SetFlushInterval(seconds int) error {
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
	customEvent := &CustomEvent{
		Name:      eventName,
		Timestamp: time.Now(),
		Data:      eventData,
	}

	// Emit the event using the customEventEmitter
	if err := customEventEmitter.Emit(customEvent); err != nil {
		return fmt.Errorf("failed to emit custom event: %w", err)
	}
	return nil
}

// Shutdown cleans up resources
func Shutdown() {
	if customEventEmitter != nil {
		customEventEmitter.Close()
	}
	if esSyncerInstance != nil {
		esSyncerInstance.Close()
	}
}
