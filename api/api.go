package api

import (
	"fmt"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"

	"gitlab.com/nunet/device-management-service/dms/onboarding"
	"gitlab.com/nunet/device-management-service/network/libp2p"
	"gitlab.com/nunet/device-management-service/telemetry/logger"
)

type RESTServerConfig struct {
	P2p        *libp2p.Libp2p
	Onboarding *onboarding.Onboarding
	Logger     *logger.Logger
	MidW       []gin.HandlerFunc
	Port       uint32
	Addr       string
}

// RESTServer represents a HTTP server
type RESTServer struct {
	router *gin.Engine
	config RESTServerConfig
}

// NewRESTServer is a constructor function for RESTServer
// It returns a pointer to RESTServer
func NewRESTServer(config RESTServerConfig) *RESTServer {
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
func (s *RESTServer) InitializeRoutes() {
	v1 := s.router.Group("/api/v1")

	onboardHandler := NewOnboardingHandler(s.config.Onboarding)
	onboarding := v1.Group("/onboarding")
	{
		onboarding.GET("/provisioned", onboardHandler.ProvisionedCapacity)
		onboarding.GET("/address/new", onboardHandler.CreatePaymentAddress)
		onboarding.GET("/status", onboardHandler.OnboardStatus)
		onboarding.POST("/onboard", onboardHandler.Onboard)
		onboarding.POST("/resource-config", onboardHandler.ResourceConfig)
		onboarding.DELETE("/offboard", onboardHandler.Offboard)
	}

	deviceHandler := DeviceHandler{}
	device := v1.Group("/device")
	{
		device.GET("/status", deviceHandler.DeviceStatus)
		device.POST("/status", deviceHandler.UpdateDeviceStatus)
	}

	vmHandler := VMHandler{}
	vm := v1.Group("/vm")
	{
		vm.POST("/start-default", vmHandler.StartDefault)
		vm.POST("/start-custom", vmHandler.StartCustom)
	}

	ph := P2PHandler{p2p: s.config.P2p}
	p2p := v1.Group("/peers")
	{
		p2p.GET("", ph.ListPeers)
		p2p.GET("/self", ph.SelfPeerInfo)

		// DEBUGGING ONLY
		if _, debugMode := os.LookupEnv("NUNET_DEBUG"); debugMode {
			p2p.GET("/ping", ph.PingPeer)
			p2p.GET("/dht", ph.KnownPeers)
			// p2p.GET("/dht/dump", ph.DumpDHTHandler)
		}
	}

	// Swagger API documentation
	s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

// Run starts the server on the specified port
func (s *RESTServer) Run() error {
	return s.router.Run(fmt.Sprintf("%s:%d", s.config.Addr, s.config.Port))
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
