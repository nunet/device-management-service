package api

import (
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gitlab.com/nunet/device-management-service/network/libp2p"
)

func SetupRouter() *gin.Engine {
	// Note: While rearranging routes in groups, make sure to also update the
	// route handler swagger annotaion @Router with the correct path.

	router := gin.Default()
	router.Use(cors.New(getCustomCorsConfig()))

	v1 := router.Group("/api/v1")

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

	if _, debugMode := os.LookupEnv("NUNET_DEBUG"); debugMode {
		dht := v1.Group("/dht")
		{
			dht.GET("/update", func(c *gin.Context) { ManualDHTUpdateHandler(c, libp2p.Libp2p{}) })
		}
		kadDHT := v1.Group("/kad-dht")
		{
			kadDHT.GET("", DumpKademliaDHTHandler)
		}
		v1.GET("/ping", func(c *gin.Context) { PingPeerHandler(c, libp2p.Libp2p{}) })
		v1.GET("/oldping", func(c *gin.Context) { OldPingPeerHandler(c, libp2p.Libp2p{}) })
		v1.GET("/cleanup", CleanupPeerHandler)
	}

	p2pGrp := v1.Group("/peers")
	{
		p2pGrp.GET("", func(c *gin.Context) { ListPeersHandler(c, libp2p.Libp2p{}) })
		p2pGrp.GET("/dht", func(c *gin.Context) { ListDHTPeersHandler(c, libp2p.Libp2p{}) })
		p2pGrp.GET("/dht/dump", func(c *gin.Context) { DumpDHTHandler(c, libp2p.Libp2p{}) })
		p2pGrp.GET("/kad-dht", func(c *gin.Context) { ListKadDHTPeersHandler(c, libp2p.Libp2p{}) })
		p2pGrp.GET("/self", func(c *gin.Context) { SelfPeerInfoHandler(c, libp2p.Libp2p{}) })
		p2pGrp.GET("/depreq", func(c *gin.Context) { DefaultDepReqPeerHandler(c, libp2p.Libp2p{}) })
		p2pGrp.GET("/chat", func(c *gin.Context) { ListChatHandler(c, libp2p.Libp2p{}) })
		p2pGrp.GET("/chat/start", func(c *gin.Context) { StartChatHandler(c, libp2p.Libp2p{}) })
		p2pGrp.GET("/chat/join", func(c *gin.Context) { JoinChatHandler(c, libp2p.Libp2p{}) })
		p2pGrp.GET("/chat/clear", func(c *gin.Context) { ClearChatHandler(c, libp2p.Libp2p{}) })
		p2pGrp.GET("/file", func(c *gin.Context) { ListFileTransferRequestsHandler(c, libp2p.Libp2p{}) })
		p2pGrp.GET("/file/send", func(c *gin.Context) { SendFileTransferHandler(c, libp2p.Libp2p{}) })
		p2pGrp.GET("/file/accept", func(c *gin.Context) { AcceptFileTransferHandler(c, libp2p.Libp2p{}) })
		p2pGrp.GET("/file/clear", func(c *gin.Context) { ClearFileTransferRequestsHandler(c, libp2p.Libp2p{}) })
	}

	return router
}

func getCustomCorsConfig() cors.Config {
	config := DefaultConfig()
	// FIXME: This is a security concern.
	config.AllowOrigins = []string{"http://localhost:9991", "http://localhost:9992"}
	return config
}

// DefaultConfig returns a generic default configuration mapped to localhost.
func DefaultConfig() cors.Config {
	return cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Access-Control-Allow-Origin", "Origin", "Content-Length", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}
}
