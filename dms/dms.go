package dms

import (
	// "context"
	"context"
	"fmt"
	"log"

	"gitlab.com/nunet/device-management-service/api"
	"gitlab.com/nunet/device-management-service/db"
	"gitlab.com/nunet/device-management-service/db/repositories"
	gdb "gitlab.com/nunet/device-management-service/db/repositories/gorm"
	"gitlab.com/nunet/device-management-service/dms/onboarding"
	"gitlab.com/nunet/device-management-service/dms/resources"
	"gitlab.com/nunet/device-management-service/internal"
	backgroundtasks "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/telemetry/logger"

	// "gitlab.com/nunet/device-management-service/internal/messaging"
	"gitlab.com/nunet/device-management-service/network/libp2p"
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/multiformats/go-multiaddr"
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
	config.LoadConfig()

	db, err := db.ConnectDatabase(config.GetConfig().General.WorkDir)
	if err != nil {
		zlog.Sugar().Fatalf("unable to connect to database: %w", err)
	}

	repos := resources.ManagerRepos{
		FreeResources:      gdb.NewFreeResources(db),
		OnboardedResources: gdb.NewOnboardedResources(db),
		RequiredResources:  gdb.NewRequiredResources(db),
		VirtualMachine:     gdb.NewVirtualMachine(db),
		Services:           gdb.NewServices(db),
	}
	resourceManager := resources.NewResourceManager(repos)

	onboardR := gdb.NewOnboardingParams(db)
	p2pR := gdb.NewLibp2pInfo(db)
	uuidR := gdb.NewMachineUUID(db)
	avResR := gdb.NewAvailableResources(db)

	// check if onboarded
	oConf, err := onboardR.Get(ctx)
	if err != nil {
		if err == repositories.ErrNotFound {
			zlog.Info("machine not onboarded")
		} else {
			zlog.Sugar().Errorf("no onboarding config: %w", err)
		}
	}

	onboard := onboarding.New(&onboarding.Config{
		Fs:              afero.Afero{Fs: afero.NewOsFs()},
		ParamsRepo:      onboardR,
		P2PRepo:         p2pR,
		UUIDRepo:        uuidR,
		ResourceManager: resourceManager,
		AvResourceRepo:  avResR,
		WorkDir:         config.GetConfig().WorkDir,
		DatabasePath:    fmt.Sprintf("%s/nunet.db", config.GetConfig().General.WorkDir),
		Channels:        []string{"nunet", "nunet-test", "nunet-team", "nunet-edge"},
	})

	var p2pNet *libp2p.Libp2p

	if onboarded, _ := onboard.IsOnboarded(ctx); onboarded {
		ValidateOnboarding(&oConf)

		p2pParams, err := p2pR.Get(ctx)
		if err != nil {
			log.Fatalf("Failed to get libp2p info: %v", err)
		}

		bootstrapPeers := make([]multiaddr.Multiaddr, len(config.GetConfig().P2P.BootstrapPeers))
		for i, addr := range config.GetConfig().P2P.BootstrapPeers {
			bootstrapPeers[i], _ = multiaddr.NewMultiaddr(addr)
		}

		priv, err := crypto.UnmarshalPrivateKey(p2pParams.PrivateKey)
		if err != nil {
			zlog.Sugar().Fatalf("unable to unmarshal private key: %v", err)
		}

		cfg := &types.Libp2pConfig{
			PrivateKey:              priv,
			BootstrapPeers:          bootstrapPeers,
			Rendezvous:              "nunet-test",
			Server:                  false,
			Scheduler:               backgroundtasks.NewScheduler(10),
			CustomNamespace:         "/nunet-dht-1/",
			ListenAddress:           config.GetConfig().ListenAddress,
			PeerCountDiscoveryLimit: 40,
		}

		p2p, err := libp2p.New(cfg, afero.NewMemMapFs())
		if err != nil {
			zlog.Sugar().Errorf("unable to create libp2p instance: %v", err)
		}

		if err = p2p.Init(ctx); err != nil {
			zlog.Sugar().Errorf("unable to initialize libp2p: %v", err)
		}

		if err = p2p.Start(ctx); err != nil {
			zlog.Sugar().Errorf("unable to start libp2p: %v", err)
		}

		p2pNet = p2p
	}

	// initialize rest api server
	restConfig := api.RESTServerConfig{
		P2P:        p2pNet,
		Onboarding: onboard,
		Logger:     logger.New("rest-server"),
		Resource:   resourceManager,
		MidW:       nil,
		Port:       config.GetConfig().Rest.Port,
		Addr:       config.GetConfig().Rest.Addr,
	}
	rServer := api.NewRESTServer(&restConfig)
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
