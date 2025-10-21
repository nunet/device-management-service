// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package node

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime/trace"
	"sync"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/observability"
)

type LoggerConfigRequest struct {
	Interval       int    `json:"interval,omitempty"`
	URL            string `json:"url,omitempty"`
	Level          string `json:"level,omitempty"`
	APIKey         string `json:"api_key,omitempty"`
	APMURL         string `json:"apm_url,omitempty"`
	ElasticEnabled *bool  `json:"elastic_enabled,omitempty"`
}

type LoggerConfigResponse struct {
	Error string `json:"error,omitempty"`
	OK    bool
}

func (n *Node) handleLoggerConfig(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		log.Errorw("logger_config_error",
			"labels", string(observability.LabelNode),
			"error", err)
		n.sendReply(msg, LoggerConfigResponse{Error: err.Error()})
	}

	var (
		req  LoggerConfigRequest
		resp LoggerConfigResponse
	)

	if err := json.Unmarshal(msg.Message, &req); err != nil {
		handleErr(err)
		return
	}

	log.Debugw("logger_config_request_received",
		"labels", string(observability.LabelNode),
		"configRequest", req)

	if req.Interval != 0 {
		if err := observability.SetFlushInterval(req.Interval); err != nil {
			handleErr(err)
			return
		}
		log.Debugw("logger_flush_interval_updated",
			"labels", string(observability.LabelNode),
			"interval", req.Interval)
	}
	if req.Level != "" {
		if err := observability.SetLogLevel(req.Level); err != nil {
			handleErr(err)
			return
		}
		log.Debugw("logger_level_updated",
			"labels", string(observability.LabelNode),
			"level", req.Level)
	}
	if req.URL != "" {
		if err := observability.SetElasticsearchEndpoint(req.URL); err != nil {
			handleErr(err)
			return
		}
		log.Debugw("logger_elasticsearch_endpoint_updated",
			"labels", string(observability.LabelNode),
			"url", req.URL)
	}
	if req.APIKey != "" { // Handle API Key
		if err := observability.SetAPIKey(req.APIKey); err != nil {
			handleErr(err)
			return
		}
		log.Debugw("logger_api_key_updated",
			"labels", string(observability.LabelNode))
	}
	if req.APMURL != "" { // Handle APM URL
		if err := observability.SetAPMURL(req.APMURL); err != nil {
			handleErr(err)
			return
		}
		log.Debugw("logger_apm_url_updated",
			"labels", string(observability.LabelNode),
			"apmUrl", req.APMURL)
	}
	if req.ElasticEnabled != nil { // Handle Elasticsearch Enabled
		if err := observability.EnableElasticsearchLogging(*req.ElasticEnabled); err != nil {
			handleErr(err)
			return
		}
		log.Debugw("logger_elasticsearch_enabled_flag_updated",
			"labels", string(observability.LabelNode),
			"enabled", *req.ElasticEnabled)
	}

	resp.OK = true
	n.sendReply(msg, resp)
}

func (n *Node) handleFlightrec(msg actor.Envelope) {
	defer msg.Discard()

	if n.flightrec == nil {
		return
	}

	go captureSnapshot(n.flightrec, filepath.Join(n.dmsConfig.WorkDir, "logs"), "flightrec.trace")
	n.sendReply(msg, PingResponse{})
}

var once sync.Once

// captureSnapshot captures a flight recorder snapshot.
func captureSnapshot(fr *trace.FlightRecorder, dir, filename string) {
	// once.Do ensures that the provided function is executed only once.
	once.Do(func() {
		_ = os.MkdirAll(dir, 0o755)

		f, err := os.Create(filepath.Join(dir, filename))
		if err != nil {
			log.Errorw("opening_flightrec", "file", f.Name(), "error", err)
			return
		}
		defer f.Close() // ignore error

		// WriteTo writes the flight recorder data to the provided io.Writer.
		_, err = fr.WriteTo(f)
		if err != nil {
			log.Errorw("writing_flightrec", "file", f.Name(), "error", err)
			return
		}

		// Stop the flight recorder after the snapshot has been taken.
		fr.Stop()
		log.Infow("flightrec_captured", "file", f.Name())
	})
}
