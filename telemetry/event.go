package telemetry

type EventType string

const (
	Heartbeat EventType = "heart_beat"
	Message   EventType = "message"
	Metric    EventType = "metric"
	// Add other event types here
)

type Event struct {
	Type    EventType
	Payload map[string]interface{}
	Index   string
}
