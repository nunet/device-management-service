package dms

import (
	// "context"
	"fmt"
	"log"
	"os"
	"time"

	"gitlab.com/nunet/device-management-service/api"
	"gitlab.com/nunet/device-management-service/db"
	"gitlab.com/nunet/device-management-service/internal"
	"gitlab.com/nunet/device-management-service/internal/config"

	// "gitlab.com/nunet/device-management-service/internal/messaging"
	"gitlab.com/nunet/device-management-service/models"
	"gitlab.com/nunet/device-management-service/network/libp2p"
	"gitlab.com/nunet/device-management-service/utils"

	"github.com/libp2p/go-libp2p/core/crypto"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// NewP2P is stub, real implementation is needed in order to pass it to
// routers which access them in some handlers.
func NewP2P() libp2p.Libp2p {
	return libp2p.Libp2p{}
}

func Run() {
	// ctx := context.Background()
	log.Println("WARNING: Most parts commented out in dms.Run()")
	config.LoadConfig()

	db.ConnectDatabase()

	go startServer()

	// go messaging.DeploymentWorker()

	// go messaging.FileTransferWorker(ctx)

	// wait for server to start properly before sending requests below
	time.Sleep(time.Second * 5)

	// check if onboarded
	if onboarded, _ := utils.IsOnboarded(); onboarded {
		metadata, err := utils.ReadMetadataFile()
		if err != nil {
			zlog.Sugar().Errorf("unable to read metadata.json: %v", err)
			os.Exit(1)
		}
		ValidateOnboarding(metadata)

		p2pParams := GetP2PParams()
		_, err = crypto.UnmarshalPrivateKey(p2pParams.PrivateKey)
		if err != nil {
			zlog.Sugar().Fatalf("unable to unmarshal private key: %v", err)
		}

		// libp2p.RunNode(priv, p2pParams.ServerMode, p2pParams.Available)
		// if libp2p.GetP2P().Host != nil {
		// 	SanityCheck(db.DB)
		// }
	}

	// wait for SIGINT or SIGTERM
	sig := <-internal.ShutdownChan
	fmt.Printf("Shutting down after receiving %v...\n", sig)

	// add actual cleanup code here
	fmt.Println("Cleaning up before shutting down")

	// exit
	os.Exit(0)
}

func GetP2PParams() (libp2pInfo models.Libp2pInfo) {
	result := db.DB.Where("id = ?", 1).Find(&libp2pInfo)
	if result.Error == nil && libp2pInfo.PrivateKey != nil {
		return
	}
	return
}

func ValidateOnboarding(metadata *models.Metadata) {
	// Check 1: Check if payment address is valid
	err := utils.ValidateAddress(metadata.PublicKey)
	if err != nil {
		zlog.Sugar().Errorf("the payment address %s is not valid", metadata.PublicKey)
		zlog.Sugar().Error("exiting DMS")
		os.Exit(1)
	}
}

func startServer() {
	router := api.SetupRouter()
	// router.Use(otelgin.Middleware(tracing.MachineName))

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	router.Run(fmt.Sprintf(":%d", config.GetConfig().Rest.Port))

}
