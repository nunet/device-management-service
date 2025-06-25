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
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/types"
)

type ServerConfig struct {
	P2P         network.Network
	Onboarding  *onboarding.Onboarding
	Resource    types.ResourceManager
	Middlewares []gin.HandlerFunc
	Port        uint32
	Addr        string
}

// getCorsConfig returns the default CORS configuration
func getCorsConfig() cors.Config {
	return cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Access-Control-Allow-Origin", "Origin", "Content-Length", "Content-Type"},
		AllowOrigins:     []string{"http://localhost:9991", "http://localhost:9992"}, // TODO: this is a security risk
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}
}

func setupRouter(middlewares []gin.HandlerFunc) *gin.Engine {
	middlewares = append(middlewares, cors.New(getCorsConfig()))
	router := gin.Default()
	router.Use(middlewares...)
	return router
}

// Server represents a REST server
type Server struct {
	router *gin.Engine
	config *ServerConfig
}

// NewServer creates a new REST server
func NewServer(config *ServerConfig) *Server {
	rs := &Server{
		router: setupRouter(config.Middlewares),
		config: config,
	}

	log.Infow("rest_server_init_success", "addr", config.Addr, "port", config.Port)
	return rs
}

// HealthCheck is a health check endpoint
func (rs *Server) HealthCheck(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}

// SetupRoutes sets up all the endpoint routes
func (rs *Server) SetupRoutes() {
	// /health route
	rs.router.GET("/health", rs.HealthCheck)

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
func (rs *Server) Run() error {
	addr := fmt.Sprintf("%s:%d", rs.config.Addr, rs.config.Port)
	if err := rs.router.Run(addr); err != nil {
		log.Errorw("rest_server_run_failure", "addr", addr, "error", err)
		return err
	}

	log.Infow("rest_server_run_success", "addr", addr)
	return nil
}
