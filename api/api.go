package api

import (
	"fmt"
	"os"
	"time"

	"gitlab.com/nunet/device-management-service/types"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"

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

	// onboardHandler := NewOnboardingHandler(s.config.Onboarding)
	onboarding := v1.Group("/onboarding")
	{
		onboarding.GET("/provisioned", rs.ProvisionedCapacity)
		onboarding.GET("/address/new", rs.CreatePaymentAddress)
		onboarding.GET("/status", rs.Status)
		onboarding.GET("/info", rs.Info)
		onboarding.POST("/onboard", rs.Onboard)
		onboarding.POST("/resource-config", rs.ResourceConfig)
		onboarding.DELETE("/offboard", rs.Offboard)
	}

	// deviceHandler := DeviceHandler{}
	device := v1.Group("/device")
	{
		device.GET("/status", rs.DeviceStatus)
		device.POST("/status", rs.UpdateDeviceStatus)
	}

	// vmHandler := VMHandler{}
	vm := v1.Group("/vm")
	{
		vm.POST("/start-default", rs.StartDefault)
		vm.POST("/start-custom", rs.StartCustom)
	}

	// ph := P2PHandler{p2p: rs.config.P2P}
	p2p := v1.Group("/peers")
	{
		p2p.GET("", rs.ListPeers)
		p2p.GET("/self", rs.SelfPeerInfo)

		// DEBUGGING ONLY
		if _, debugMode := os.LookupEnv("NUNET_DEBUG"); debugMode {
			p2p.GET("/ping", rs.PingPeer)
			p2p.GET("/dht", rs.KnownPeers)
			// p2p.GET("/dht/dump", ph.DumpDHTHandler)
		}
	}

	// /actor routes
	actor := v1.Group("/actor")
	{
		actor.GET("/handle", rs.ActorHandle)
		actor.POST("/send", rs.ActorSendMessage)
		actor.POST("/invoke", rs.ActorInvoke)
		actor.POST("/broadcast", rs.ActorBroadcast)
	}

	// Swagger API documentation
	rs.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
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
