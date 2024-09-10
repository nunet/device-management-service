package s3

import (
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"gitlab.com/nunet/device-management-service/telemetry"
	"gitlab.com/nunet/device-management-service/telemetry/logger"
)

var (
	zlog *otelzap.Logger
	st   = telemetry.NewTelemetry(nil, nil, true)
)

// Context keys used for tracing
type contextKey string

const (
	pathKey             contextKey = "path"
	SourceSpecsKey      contextKey = "sourceSpecs"
	errorKey            contextKey = "error"
	OutputPathKey       contextKey = "outputPath"
	bucketKey           contextKey = "bucket"
	S3KeyKey            contextKey = "key"
	ContentLength       contextKey = "content_length"
	FilePathKey         contextKey = "file_path"
	VolumePathKey       contextKey = "volume_path"
	sanitizedKeyContext contextKey = "sanitized_key"
)

func init() {
	zlog = logger.OtelZapLogger("s3")
}
