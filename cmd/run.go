package cmd

import (
	"fmt"
	"net/http"
	_ "net/http/pprof" //#nosec
	"os"

	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/dms"
	"gitlab.com/nunet/device-management-service/internal/config"
)

func newRunCmd() *cobra.Command {
	var context string
	pprof := config.GetConfig().Profiler.Enabled
	pprofAddr := config.GetConfig().Profiler.Addr
	pprofPort := config.GetConfig().Profiler.Port

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Start the Device Management Service",
		Long: `Start the Device Management Service

The Device Management Service (DMS) is a system application for running a node in the NuNet decentralized network of compute providers.

By default, DMS listens on port 9999. For more information on configuration, see:

  nunet config --help

Or manually create a dms_config.json file and refer to the README for available settings.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			passphrase := os.Getenv("DMS_PASSPHRASE")

			var err error
			if passphrase == "" {
				fmt.Print("Please enter the DMS passphrase. This will be used to encrypt/decrypt the keystore containing necessary secrets for DMS:\n")
				passphrase, err = utils.PromptForPassphrase(false)
				if err != nil {
					return fmt.Errorf("error reading passphrase from stdin: %w", err)
				}

				// TODO: validate passphrase (minimum x characters)
				if passphrase == "" {
					return fmt.Errorf("invalid passphrase")
				}
			}

			if pprof {
				go func() {
					pprofMux := http.DefaultServeMux
					http.DefaultServeMux = http.NewServeMux()

					profilerAddr := fmt.Sprintf("%s:%d", pprofAddr, pprofPort)
					log.Infof("Starting profiler on %s\n", profilerAddr)
					// #nosec
					if err := http.ListenAndServe(profilerAddr, pprofMux); err != nil {
						log.Errorf("Error starting profiler: %v\n", err)
					}
				}()
			}

			return dms.Run(passphrase, context)
		},
	}
	cmd.Flags().BoolVar(&pprof, "pprof", pprof, "enable profiling")
	cmd.Flags().StringVar(&pprofAddr, "pprof-addr", pprofAddr, "enable profiling")
	cmd.Flags().Uint32Var(&pprofPort, "pprof-port", pprofPort, "enable profiling")
	cmd.Flags().StringVarP(&context, "context", "c", dms.DefaultContextName, "specify a capability context")
	return cmd
}
