package api

import (
	"fmt"
	"time"

	"gitlab.com/nunet/device-management-service/types"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"gitlab.com/nunet/device-management-service/dms/onboarding"
	"gitlab.com/nunet/device-management-service/network/libp2p"
	"gitlab.com/nunet/device-management-service/telemetry/logger"
)

type RESTServerConfig struct {
	P2P        *libp2p.Libp2p
	Onboarding *onboarding.Onboarding
	Logger     *logger.Logger
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
	return &RESTServer{
		router: setupRouter(config.MidW),
		config: config,
	}
}

func setupRouter(mid []gin.HandlerFunc) *gin.Engine {
	mid = append(mid, cors.New(getCustomCorsConfig()))
	router := gin.Default()
	router.Use(mid...)
	return router
}

// InitializeRoutes sets up all the endpoint routes
func (rs *RESTServer) InitializeRoutes() {
	v1 := rs.router.Group("/api/v1")

	// /actor routes
	actor := v1.Group("/actor")
	{
		actor.GET("/handle", rs.ActorHandle)
		actor.POST("/send", rs.ActorSendMessage)
		actor.POST("/invoke", rs.ActorInvoke)
		actor.POST("/broadcast", rs.ActorBroadcast)
	}
}

// Run starts the server on the specified port
func (rs *RESTServer) Run() error {
	return rs.router.Run(fmt.Sprintf("%s:%d", rs.config.Addr, rs.config.Port))
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
