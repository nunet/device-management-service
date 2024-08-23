package cmd

import (
	"fmt"

	"gitlab.com/nunet/device-management-service/telemetry/logger"

	"github.com/coreos/go-systemd/sdjournal"
	"gitlab.com/nunet/device-management-service/cmd/backend"
)

var (
	networkService    = &backend.Network{}
	webSocketClient   = &backend.WebSocket{}
	fileSystemService = &backend.OS{}
	resourceService   = &backend.Resources{}
	p2pService        = &backend.P2P{}
	utilsService      = &backend.Utils{}
	walletService     = &backend.Wallet{}
	journalService    *backend.Journal
	zlog              = logger.OtelZapLogger("cmd")
)

func init() {
	j, err := sdjournal.NewJournal()
	if err != nil {
		fmt.Printf("Error: could not initialize sdjournal: %v\n", err)
		return
	}

	journalService = backend.SetRealJournal(j)

	// initialize commands and sub-commands
	logCmd = NewLogCmd(networkService, fileSystemService, journalService)
	gpuCapacityCmd.Flags().BoolVarP(&flagCudaTensor, "cuda-tensor", "c", false, "check CUDA Tensor")
	gpuCapacityCmd.Flags().BoolVarP(&flagRocmHip, "rocm-hip", "r", false, "check ROCM-HIP")
	gpuCapacityCmd.Flags().BoolVarP(&flagIntelXPU, "intel-xpu", "i", false, "check Intel XPU")
	gpuCmd.AddCommand(gpuCapacityCmd)
	gpuCmd.AddCommand(gpuStatusCmd)
	gpuCmd.AddCommand(gpuOnboardCmd)
	offboardCmd.Flags().BoolVarP(&flagForce, "force", "f", false, "force offboarding")
	shellCmd.Flags().StringVar(&flagNode, "node-id", "", "set nodeID value")

	// initialize top level commands
	rootCmd.AddCommand(gpuCmd)
	rootCmd.AddCommand(offboardCmd)
	rootCmd.AddCommand(onboardMLCmd)
	rootCmd.AddCommand(chatCmd)
	rootCmd.AddCommand(shellCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(peerCmd)
	rootCmd.AddCommand(onboardCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(deviceCmd)
	rootCmd.AddCommand(capacityCmd)
	rootCmd.AddCommand(resourceConfigCmd)
	rootCmd.AddCommand(logCmd)
	rootCmd.AddCommand(walletCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(autocompleteCmd)
}
