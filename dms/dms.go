package dms

import (
	// "context"
	"context"
	"fmt"
	"log"
	"time"

	"gitlab.com/nunet/device-management-service/api"
	gdb "gitlab.com/nunet/device-management-service/db/repositories/gorm"
	"gitlab.com/nunet/device-management-service/dms/onboarding"
	"gitlab.com/nunet/device-management-service/internal"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/telemetry/logger"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	// "gitlab.com/nunet/device-management-service/internal/messaging"
	"gitlab.com/nunet/device-management-service/network/libp2p"
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/spf13/afero"
)

// NewP2P is stub, real implementation is needed in order to pass it to
// routers which access them in some handlers.
func NewP2P() libp2p.Libp2p {
	return libp2p.Libp2p{}
}

// QUESTION(dms-initialization): should the db instance be constructed here?
func Run() {
	ctx := context.Background()
	log.Println("WARNING: Most parts commented out in dms.Run()")
	config.LoadConfig()

	// XXX: wait for server to start properly before sending requests below
	// TODO: should be removed
	time.Sleep(time.Second * 5)

	// check if onboarded
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("%s/nunet.db", config.GetConfig().General.WorkDir)), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	onboardR := gdb.NewOnboardingParams(db)
	p2pR := gdb.NewLibp2pInfo(db)
	uuidR := gdb.NewMachineUUID(db)
	avResR := gdb.NewAvailableResources(db)

	oConf, err := onboardR.Get(ctx)
	if err != nil {
		log.Fatalf("Failed to get onboarding config: %v", err)
	}

	onboard := onboarding.New(onboarding.Config{
		Fs:             afero.Afero{Fs: afero.NewOsFs()},
		P2PRepo:        p2pR,
		UUIDRepo:       uuidR,
		AvResourceRepo: avResR,
		WorkDir:        config.GetConfig().WorkDir,
		DatabasePath:   fmt.Sprintf("%s/nunet.db", config.GetConfig().General.WorkDir),
		Channels:       []string{"nunet", "nunet-test", "nunet-team", "nunet-edge"},
	})

	if onboarded, _ := onboard.IsOnboarded(ctx); onboarded {
		ValidateOnboarding(&oConf)

		p2pParams, err := p2pR.Get(ctx)
		if err != nil {
			log.Fatalf("Failed to get libp2p info: %v", err)
		}

		_, err = crypto.UnmarshalPrivateKey(p2pParams.PrivateKey)
		if err != nil {
			zlog.Sugar().Fatalf("unable to unmarshal private key: %v", err)
		}
	}

	// initialize rest api server
	restConfig := api.RESTServerConfig{
		P2p:        nil,
		Onboarding: nil,
		Logger:     logger.New("rest-server"),
		MidW:       nil,
		Port:       config.GetConfig().Rest.Port,
		Addr:       config.GetConfig().Rest.Addr,
	}
	rServer := api.NewRESTServer(restConfig)
	rServer.InitializeRoutes()

	go func() {
		err := rServer.Run()
		if err != nil {
			log.Fatal(err)
		}
	}()

	// wait for SIGINT or SIGTERM
	sig := <-internal.ShutdownChan
	fmt.Printf("Shutting down after receiving %v...\n", sig)

	// add cleanup code here
	fmt.Println("Cleaning up before shutting down")
}

func ValidateOnboarding(oConf *types.OnboardingConfig) {
	// Check 1: Check if payment address is valid
	err := utils.ValidateAddress(oConf.PublicKey)
	if err != nil {
		zlog.Sugar().Errorf("the payment address %s is not valid", oConf.PublicKey)
		zlog.Sugar().Error("exiting DMS")
		return
	}
}
