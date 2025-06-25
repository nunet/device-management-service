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
	_ "net/http/pprof" //#nosec
	"os"

	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/dms"
	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/internal"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/env"
)

func newRunCmd(
	env env.EnvironmentProvider, gcfg *config.Config,
) *cobra.Command {
	var context string
	pprof := gcfg.Profiler.Enabled
	pprofAddr := gcfg.Profiler.Addr
	pprofPort := gcfg.Profiler.Port

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Start the Device Management Service",
		Long: `Start the Device Management Service

The Device Management Service (DMS) is a system application for running a node in the NuNet decentralized network of compute providers.

By default, DMS listens on port 9999. For more information on configuration, see:

  nunet config --help

Or manually create a dms_config.json file and refer to the README for available settings.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			passphrase, err := utils.GetDMSPassphrase(env, false)
			if err != nil {
				return fmt.Errorf("get dms passphrase: %w", err)
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

			dmsInstance, err := dms.NewDMS(gcfg, passphrase, context)
			if err != nil {
				return fmt.Errorf("failed to initialize dms: %w", err)
			}

			go func() {
				sig := <-internal.ShutdownChan
				log.Infof("Shutting down after receiving %v...\n", sig)

				dmsInstance.Stop()
				os.Exit(1)
			}()

			err = dmsInstance.Run()
			if err != nil {
				return err
			}

			<-internal.ShutdownChan
			return nil
		},
	}
	cmd.Flags().BoolVar(&pprof, "pprof", pprof, "enable profiling")
	cmd.Flags().StringVar(&pprofAddr, "pprof-addr", pprofAddr, "enable profiling")
	cmd.Flags().Uint32Var(&pprofPort, "pprof-port", pprofPort, "enable profiling")
	cmd.Flags().StringVarP(&context, "context", "c", node.DefaultContextName, "specify a capability context")
	return cmd
}
