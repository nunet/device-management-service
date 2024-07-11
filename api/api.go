package api

import (
	"fmt"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"

	"gitlab.com/nunet/device-management-service/network/libp2p"
	"gitlab.com/nunet/device-management-service/telemetry/logger"
)

type RESTServer struct {
	router *gin.Engine
	logger *logger.Logger
	p2p    *libp2p.Libp2p
	port   int
}

func NewRestServer(logger *logger.Logger, p2p *libp2p.Libp2p, mid []gin.HandlerFunc, port int) *RESTServer {
	return &RESTServer{
		router: setupRouter(mid),
		logger: logger,
		p2p:    p2p,
		port:   port,
	}
}

func setupRouter(mid []gin.HandlerFunc) *gin.Engine {
	mid = append(mid, cors.New(getCustomCorsConfig()))
	router := gin.Default()
	router.Use(mid...)
	return router
}

func (s *RESTServer) InitializeRoutes() {
	v1 := s.router.Group("/api/v1")

	onboarding := v1.Group("/onboarding")
	{
		onboarding.GET("/metadata", GetMetadataHandler)
		onboarding.GET("/provisioned", ProvisionedCapacityHandler)
		onboarding.GET("/address/new", CreatePaymentAddressHandler)
		onboarding.GET("/status", OnboardStatusHandler)
		onboarding.POST("/onboard", OnboardHandler)
		onboarding.POST("/resource-config", ResourceConfigHandler)
		onboarding.DELETE("/offboard", OffboardHandler)
	}

	device := v1.Group("/device")
	{
		device.GET("/status", DeviceStatusHandler)
		device.POST("/status", ChangeDeviceStatusHandler)
	}

	vm := v1.Group("/vm")
	{
		vm.POST("/start-default", StartDefaultHandler)
		vm.POST("/start-custom", StartCustomHandler)
	}

	run := v1.Group("/run")
	{
		run.GET("/deploy", DeploymentRequestHandler) // websocket
		run.GET("/checkpoints", ListCheckpointHandler)
		run.POST("/request-service", RequestServiceHandler)
	}

	tx := v1.Group("/transactions")
	{
		tx.GET("", GetJobTxHashesHandler)
		tx.POST("/request-reward", RequestRewardHandler)
		tx.POST("/send-status", SendTxStatusHandler)
		tx.POST("/update-status", UpdateTxStatusHandler)
	}

	tele := v1.Group("/telemetry")
	{
		tele.GET("/free", GetFreeResourcesHandler)
	}

	ph := P2pHandler{p2p: s.p2p}
	p2p := v1.Group("/peers")
	{
		p2p.GET("", ph.ListPeersHandler)
		p2p.GET("/self", ph.SelfPeerInfoHandler)

		// DEBUGGING ONLY
		if _, debugMode := os.LookupEnv("NUNET_DEBUG"); debugMode {
			p2p.GET("/ping", ph.PingPeerHandler)
			p2p.GET("/dht", ph.KnownPeersHandler)
			// p2p.GET("/dht/dump", ph.DumpDHTHandler)
		}
	}

	// Swagger API documentation
	s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

func (s *RESTServer) Run() {
	s.router.Run(fmt.Sprintf(":%d", s.port))
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
