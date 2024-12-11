package types

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

type HealthCheckResponse struct {
	Type  string `json:"type"`  // type of response
	Value string `json:"value"` // value of response
}

type HealthCheckManifest struct {
	Type     string              `json:"type"`     // type of healthcheck
	Exec     []string            `json:"exec"`     // command to execute
	Endpoint string              `json:"endpoint"` // endpoint to check
	Response HealthCheckResponse `json:"response"` // expected response
	Interval time.Duration       `json:"interval"` // interval between healthchecks
}

func NewHealthCheck(mf HealthCheckManifest, fn func(HealthCheckManifest) error) (func() error, error) {
	switch mf.Type {
	case "command":
		healthcheck := func() error {
			return fn(mf)
		}

		return healthcheck, nil
	case "http":
		healthcheck := func() error {
			res, err := http.Get(mf.Endpoint)
			if err != nil {
				return fmt.Errorf("health check request failed: %w", err)
			}
			defer res.Body.Close()

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				return fmt.Errorf("failed to read health check request output: %w", err)
			}

			if string(bytes) != mf.Response.Value {
				return fmt.Errorf("unexpected health check request output: %s", string(bytes))
			}

			return nil
		}

		return healthcheck, nil

	default:
		return nil, fmt.Errorf("unknown healthcheck type")
	}
}
