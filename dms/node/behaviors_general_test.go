package node

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/observability"
)

func TestHandleLoggerConfig(t *testing.T) {
	t.Parallel()
	const (
		flushInterval = 10
		elasticURL    = "http://example.com/logs"
		logLevel      = "debug"
		apiKey        = "test-api-key"
		apmURL        = "http://example.com/apm"
	)
	elasticEnabled := false

	// template message
	msg, err := actor.Message(
		actor.Handle{},
		actor.Handle{},
		behaviors.LoggerConfigBehavior,
		LoggerConfigRequest{
			Interval:       flushInterval,
			URL:            elasticURL,
			Level:          logLevel,
			APIKey:         apiKey,
			APMURL:         apmURL,
			ElasticEnabled: &elasticEnabled,
		},
		actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
	)
	require.NoError(t, err)

	t.Run("successfully config of all params", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.LoggerConfigBehavior)

		msg.From = sActor.Handle()
		msg.To = node.actor.Handle()
		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp LoggerConfigResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.True(t, resp.OK)
		assert.Empty(t, resp.Error)

		// confirm config
		observabilityConfig := observability.ObservabilityCfg
		assert.Equal(t, flushInterval, observabilityConfig.FlushInterval)
		assert.Equal(t, elasticURL, observabilityConfig.ElasticsearchURL)
		assert.Equal(t, logLevel, observabilityConfig.LogLevel)
		assert.Equal(t, apiKey, observabilityConfig.ElasticsearchAPIKey)
	})
}
