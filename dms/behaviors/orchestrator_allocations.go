package behaviors

import (
	"gitlab.com/nunet/device-management-service/types"
)

// Behavior payloads for behaviors invoked between allocations and
// orchestrators

// TODO: keep here temporarily. We must organize types and behavior payloads.
// issue: https://gitlab.com/nunet/device-management-service/-/issues/893

type SubnetAddPeerRequest struct {
	SubnetID string
	PeerID   string
	IP       string
}

type SubnetAddPeerResponse struct {
	OK    bool
	Error string
}

type SubnetDNSAddRecordsRequest struct {
	SubnetID string
	// map of domain name:ip
	Records map[string]string
}

type SubnetDNSAddRecordsResponse struct {
	OK    bool
	Error string
}

type SubnetMapPortRequest struct {
	SubnetID   string
	Protocol   string
	SourceIP   string
	SourcePort string
	DestIP     string
	DestPort   string
}

type SubnetMapPortResponse struct {
	OK    bool
	Error string
}

type AllocationStartRequest struct {
	SubnetIP    string
	GatewayIP   string
	PortMapping map[int]int
}

type AllocationStartResponse struct {
	OK    bool
	Error string
}

type RegisterHealthcheckRequest struct {
	EnsembleID  string
	HealthCheck types.HealthCheckManifest
}

type RegisterHealthcheckResponse struct {
	OK    bool
	Error string
}

type HealthCheckResponse struct {
	OK    bool
	Error string
}

type TaskTerminationNotification struct {
	AllocationID string
	Status       string

	Error TerminationError

	Stdout []byte
	Stderr []byte
}

// TerminationError holds information necessary to handle
// failure recovery given retry policies.
type TerminationError struct {
	Err      error
	ExitCode int
	// Killed is used to identify if the application was killed
	// by external means, rather than app exiting itself
	Killed bool
}

type AllocationRestartResponse struct {
	OK    bool
	Error string
}
