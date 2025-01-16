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

	var (
		req  LoggerConfigRequest
		resp LoggerConfigResponse
	)

	if err := json.Unmarshal(msg.Message, &req); err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	if req.Interval != 0 {
		if err := observability.SetFlushInterval(req.Interval); err != nil {
			resp.Error = err.Error()
			n.sendReply(msg, resp)
			return
		}
	}
	if req.Level != "" {
		if err := observability.SetLogLevel(req.Level); err != nil {
			resp.Error = err.Error()
			n.sendReply(msg, resp)
			return
		}
	}
	if req.URL != "" {
		if err := observability.SetElasticsearchEndpoint(req.URL); err != nil {
			resp.Error = err.Error()
			n.sendReply(msg, resp)
			return
		}
	}
	if req.APIKey != "" { // Handle API Key
		if err := observability.SetAPIKey(req.APIKey); err != nil {
			resp.Error = err.Error()
			n.sendReply(msg, resp)
			return
		}
	}
	if req.APMURL != "" { // Handle APM URL
		if err := observability.SetAPMURL(req.APMURL); err != nil {
			resp.Error = err.Error()
			n.sendReply(msg, resp)
			return
		}
	}
	if req.ElasticEnabled != nil { // Handle Elasticsearch Enabled
		if err := observability.EnableElasticsearchLogging(*req.ElasticEnabled); err != nil {
			resp.Error = err.Error()
			n.sendReply(msg, resp)
			return
		}
	}

	resp.OK = true
	n.sendReply(msg, resp)
}
