// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package api

import (
	"fmt"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gitlab.com/nunet/device-management-service/dms/onboarding"
	"gitlab.com/nunet/device-management-service/network/libp2p"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

type RESTServerConfig struct {
	P2P        *libp2p.Libp2p
	Onboarding *onboarding.Onboarding
	Resource   types.ResourceManager
	MidW       []gin.HandlerFunc
	Port       uint32
	Addr       string
}

// RESTServer represents a HTTP server
type RESTServer struct {
	router *gin.Engine
	config *RESTServerConfig
}

// NewRESTServer is a constructor function for RESTServer
// It returns a pointer to RESTServer
func NewRESTServer(config *RESTServerConfig) *RESTServer {
	endTrace := observability.StartTrace("rest_server_init_duration")
	defer endTrace()

	rs := &RESTServer{
		router: setupRouter(config.MidW),
		config: config,
	}

	log.Infow("rest_server_init_success", "addr", config.Addr, "port", config.Port)
	return rs
}

func setupRouter(mid []gin.HandlerFunc) *gin.Engine {
	mid = append(mid, cors.New(getCustomCorsConfig()))
	router := gin.Default()
	router.Use(mid...)
	return router
}

// InitializeRoutes sets up all the endpoint routes
func (rs *RESTServer) InitializeRoutes() {
	endTrace := observability.StartTrace("rest_server_route_init_duration")
	defer endTrace()

	v1 := rs.router.Group("/api/v1")

	// /actor routes
	actor := v1.Group("/actor")
	{
		actor.GET("/handle", rs.ActorHandle)
		actor.POST("/send", rs.ActorSendMessage)
		actor.POST("/invoke", rs.ActorInvoke)
		actor.POST("/broadcast", rs.ActorBroadcast)
	}

	log.Infow("rest_server_route_init_success", "endpoint", "/api/v1/actor")
}

// Run starts the server on the specified port
func (rs *RESTServer) Run() error {
	endTrace := observability.StartTrace("rest_server_run_duration")
	defer endTrace()

	addr := fmt.Sprintf("%s:%d", rs.config.Addr, rs.config.Port)
	if err := rs.router.Run(addr); err != nil {
		log.Errorw("rest_server_run_failure", "addr", addr, "error", err)
		return err
	}

	log.Infow("rest_server_run_success", "addr", addr)
	return nil
}

func getCustomCorsConfig() cors.Config {
	config := defaultConfig()
	// FIXME: This is a security concern.
	config.AllowOrigins = []string{"http://localhost:9991", "http://localhost:9992"}
	return config
}

// defaultConfig returns a generic default configuration mapped to localhost.
func defaultConfig() cors.Config {
	return cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Access-Control-Allow-Origin", "Origin", "Content-Length", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}
}
