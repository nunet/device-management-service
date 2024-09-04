package basiccontroller

import "gitlab.com/nunet/device-management-service/telemetry"

var st = telemetry.GetTelemetry()

// contextKey is a custom type to avoid context key collisions.
type contextKey string

const (
	pathKey        contextKey = "path"
	errorKey       contextKey = "error"
	identifierKey  contextKey = "identifier"
	idTypeKey      contextKey = "idType"
	volumeCountKey contextKey = "volume_count"
	sizeKey        contextKey = "size"
	volumeIDKey    contextKey = "volumeID"
)
