// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cmd

import (
	"fmt"
	"net/http"
	_ "net/http/pprof" // #nosec
	"os"

	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/cmd/cli"
	"gitlab.com/nunet/device-management-service/dms"
	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/internal"
	"gitlab.com/nunet/device-management-service/lib/did"
)

type RunOptions struct {
	Context  string
	PrismURL string
}

func newRunCmd(
	dmsCli *cli.DmsCLI,
) *cobra.Command {
	var opts RunOptions

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Start the Device Management Service",
		Long: `Start the Device Management Service

The Device Management Service (DMS) is a system application for running a node in the NuNet decentralized network of compute providers.

By default, DMS listens on port 9999. For more information on configuration, see:

  nunet config --help

Or manually create a dms_config.json file and refer to the README for available settings.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			// Configure PRISM resolver URL from flag or environment variable
			prismURL := opts.PrismURL
			if prismURL == "" {
				prismURL = dmsCli.Env().Getenv("PRISM_RESOLVER_URL")
			}
			if prismURL != "" {
				originalConfig := did.GetPRISMResolverConfig()
				did.SetPRISMResolverConfig(did.PRISMResolverConfig{
					ResolverURL:                 prismURL,
					PreferredVerificationMethod: originalConfig.PreferredVerificationMethod,
					HTTPClient:                  originalConfig.HTTPClient,
				})
			}

			passphrase, err := dmsCli.Passphrase(opts.Context)
			if err != nil {
				return fmt.Errorf("get dms passphrase: %w", err)
			}

			cfg, err := dmsCli.Config()
			if err != nil {
				return fmt.Errorf("get dms config: %w", err)
			}

			if cfg.Profiler.Enabled {
				go func() {
					pprofMux := http.DefaultServeMux
					http.DefaultServeMux = http.NewServeMux()

					profilerAddr := fmt.Sprintf("%s:%d", cfg.Profiler.Addr, cfg.Profiler.Port)
					log.Infof("Starting profiler on %s\n", profilerAddr)
					// #nosec
					if err := http.ListenAndServe(profilerAddr, pprofMux); err != nil {
						log.Errorf("Error starting profiler: %v\n", err)
					}
				}()
			}

			dmsInstance, err := dms.NewDMS(dmsCli.FS(), cfg, dmsCli.Env(), passphrase, opts.Context)
			if err != nil {
				return fmt.Errorf("failed to initialize dms: %w", err)
			}

			go func() {
				sig := <-internal.ShutdownChan
				log.Infow("Shutting down after a receiving signal", "sig", sig)

				dmsInstance.Stop()
				os.Exit(0)
			}()

			err = dmsInstance.Run()
			if err != nil {
				return err
			}

			<-internal.ShutdownChan
			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Context, "context", "c", node.DefaultContextName, "specify a capability context")
	cmd.Flags().StringVar(&opts.PrismURL, "prism-url", "", "PRISM resolver URL (e.g., http://localhost:8080). Can also be set via PRISM_RESOLVER_URL environment variable.")
	return cmd
}
