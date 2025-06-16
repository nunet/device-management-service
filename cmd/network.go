// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"

	"gitlab.com/nunet/device-management-service/client"
	"gitlab.com/nunet/device-management-service/cmd/cli"
	"gitlab.com/nunet/device-management-service/cmd/utils"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/dms/node"
	dmsUtils "gitlab.com/nunet/device-management-service/utils"
)

type Allocation struct {
	Alloc       string
	PortMapping map[int]int
	DNSName     string
	IP          string
	Status      string
}

type DeploymentNetwork struct {
	ID          string
	Allocations []Allocation
}

func newNetworkCommand(dmsCli *cli.DmsCLI) *cobra.Command {
	gpuCmd := &cobra.Command{
		Use:   "network <cmd>",
		Short: "Network Utility Tool",
		Long: `Available operations:
- ls: List all Networks the DMS is part of
- show: Show details of a specific Network
- attach: Attach to the Network.
`,
	}

	gpuCmd.AddCommand(newNetworkListCommand(dmsCli))
	gpuCmd.AddCommand(newNetworkShowCommand(dmsCli))
	gpuCmd.AddCommand(newNetworkAttachCommand(dmsCli))

	return gpuCmd
}

type networkListOpts struct {
	Context string
	Verbose bool
}

func newNetworkListCommand(dmsCli *cli.DmsCLI) *cobra.Command {
	opts := networkListOpts{}
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List all Networks",
		RunE: func(cmd *cobra.Command, _ []string) error {
			sctx, err := utils.NewSecurityContext(dmsCli, opts.Context)
			if err != nil {
				return fmt.Errorf("could not create security context: %w", err)
			}

			// Now call newClient with the correct arguments
			client, err := dmsCli.NewClient(sctx)
			if err != nil {
				return fmt.Errorf("could not create client: %w", err)
			}
			ids, err := getDeploymentIDs(cmd.Context(), client)
			if err != nil {
				return fmt.Errorf("error getting deployment IDs: %w", err)
			}

			depNets, err := getNetworkList(cmd.Context(), ids, client)
			if err != nil {
				return fmt.Errorf("error getting network list: %w", err)
			}

			if len(depNets) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No Deployment Networks")
				return nil
			}

			fmt.Println("Deployment Networks")
			// TODO Format
			for _, dn := range depNets {
				fmt.Fprintf(cmd.OutOrStdout(), "ID: %s\n", dn.ID)
				fmt.Fprintf(cmd.OutOrStdout(), "    Allocations in Network=%d\n", len(dn.Allocations))
				if opts.Verbose {
					for _, a := range dn.Allocations {
						fmt.Fprintf(cmd.OutOrStdout(), "    Alloc: %s\n", a.Alloc)
						fmt.Fprintf(cmd.OutOrStdout(), "        IP: %s\n", a.IP)
						fmt.Fprintf(cmd.OutOrStdout(), "        Hostname: %s\n", a.DNSName)
						fmt.Fprintf(cmd.OutOrStdout(), "        Ports: %+v\n", a.PortMapping)
						fmt.Fprintf(cmd.OutOrStdout(), "        Status: %+v\n", a.Status)
					}
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "\n")

			return nil
		},
	}
	cmd.Flags().StringVarP(&opts.Context, "context", "c", node.DefaultContextName, "specify a capability context")
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "verbose output")
	err := cmd.MarkFlagRequired("context")
	if err != nil {
		log.Fatalf("unable to mark flag 'context' as required: %v", err)
	}

	return cmd
}

type networkShowOpts struct {
	Context string
	ID      string
}

func newNetworkShowCommand(dmsCli *cli.DmsCLI) *cobra.Command {
	opts := networkShowOpts{}

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show details of a specific Network",
		RunE: func(cmd *cobra.Command, _ []string) error {
			sctx, err := utils.NewSecurityContext(dmsCli, opts.Context)
			if err != nil {
				return fmt.Errorf("could not create security context: %w", err)
			}

			// Now call newClient with the correct arguments
			client, err := dmsCli.NewClient(sctx)
			if err != nil {
				return fmt.Errorf("could not create client: %w", err)
			}

			depNet, err := getNetwork(cmd.Context(), opts.ID, client)
			if err != nil {
				return fmt.Errorf("error getting network detail: %w", err)
			}

			// TODO Format
			fmt.Fprintf(cmd.OutOrStdout(), "ID: %s\n", depNet.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "    Allocations in Network\n")
			for _, a := range depNet.Allocations {
				fmt.Fprintf(cmd.OutOrStdout(), "    Alloc: %s\n", a.Alloc)
				fmt.Fprintf(cmd.OutOrStdout(), "        IP: %s\n", a.IP)
				fmt.Fprintf(cmd.OutOrStdout(), "        Hostname: %s\n", a.DNSName)
				fmt.Fprintf(cmd.OutOrStdout(), "        Ports: %+v\n", a.PortMapping)
				fmt.Fprintf(cmd.OutOrStdout(), "        Status: %+v\n", a.Status)
			}

			return nil
		},
	}
	cmd.Flags().StringVarP(&opts.Context, "context", "c", node.DefaultContextName, "Capability Context")
	cmd.Flags().StringVarP(&opts.ID, "id", "i", "", "Deployment ID")
	err := cmd.MarkFlagRequired("context")
	if err != nil {
		log.Fatalf("unable to mark flag 'context' as required: %v", err)
	}
	err = cmd.MarkFlagRequired("id")
	if err != nil {
		log.Fatalf("unable to mark flag 'id' as required: %v", err)
	}
	return cmd
}

type networkAttachOpts struct {
	Context  string
	ID       string
	Alloc    string
	Shell    bool
	Forward  bool
	Username string
	Identity string
	Port     string
	PortMap  string
}

func newNetworkAttachCommand(dmsCli *cli.DmsCLI) *cobra.Command {
	opts := networkAttachOpts{}

	cmd := &cobra.Command{
		Use:   "attach",
		Short: "Attach to a specific Network",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			streams := cli.CmdStreams(cmd)

			cfg, err := dmsCli.Config()
			if err != nil {
				return fmt.Errorf("unable to get config: %w", err)
			}

			afs := afero.Afero{Fs: dmsCli.FS()}

			sctx, err := utils.NewSecurityContext(dmsCli, opts.Context)
			if err != nil {
				return fmt.Errorf("could not create security context: %w", err)
			}

			// Now call newClient with the correct arguments
			client, err := dmsCli.NewClient(sctx)
			if err != nil {
				return fmt.Errorf("could not create client: %w", err)
			}

			depNet, err := getNetwork(ctx, opts.ID, client)
			if err != nil {
				return fmt.Errorf("error getting network detail: %w", err)
			}

			targetAlloc := Allocation{}
			for _, a := range depNet.Allocations {
				if a.Alloc == opts.Alloc {
					targetAlloc = a
					break
				}
			}

			switch {
			case opts.Shell:
				key, err := afs.ReadFile(opts.Identity)
				if err != nil {
					return fmt.Errorf("unable to read identity priv key file: %w", err)
				}

				signer, err := ssh.ParsePrivateKey(key)
				if err != nil {
					if _, ok := err.(*ssh.PassphraseMissingError); ok {
						passphrase, err := dmsUtils.PromptForPassphrase(false)
						if err != nil {
							return fmt.Errorf("unable to read passphrase: %w", err)
						}
						signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(passphrase))
						if err != nil {
							return fmt.Errorf("unable to parse identity priv key with passphrase: %w", err)
						}
					} else {
						return fmt.Errorf("unable to parse identity priv key: %w", err)
					}
				}

				hostKeyManager, err := NewHostKeyManager(afs, path.Join(cfg.UserDir, "ssh", "known_hosts"))
				if err != nil {
					return fmt.Errorf("unable to create host key manager: %w", err)
				}

				sshCliConfig := &ssh.ClientConfig{
					User: opts.Username,
					Auth: []ssh.AuthMethod{
						ssh.PublicKeys(signer),
					},
					HostKeyCallback: hostKeyManager.HostKeyCallback(cmd.OutOrStdout()),
				}

				client, err := ssh.Dial("tcp", targetAlloc.IP+":"+opts.Port, sshCliConfig)
				if err != nil {
					return fmt.Errorf("unable to dial: %w", err)
				}

				session, err := client.NewSession()
				if err != nil {
					return fmt.Errorf("unable to create session: %w", err)
				}
				defer session.Close()

				modes := ssh.TerminalModes{
					ssh.ECHO:          1,     // enable echoing
					ssh.TTY_OP_ISPEED: 14400, // input = 14.4kbaud
					ssh.TTY_OP_OSPEED: 14400, // output = 14.4kbaud
				}

				fd := int(os.Stdin.Fd())
				width, height, err := term.GetSize(fd)
				if err != nil {
					return fmt.Errorf("unable to get terminal size: %w", err)
				}

				if err := session.RequestPty("linux", height, width, modes); err != nil {
					return fmt.Errorf("unable to request pseudo terminal: %w", err)
				}
				// set input and output
				session.Stdout = streams.Out
				session.Stdin = streams.In
				session.Stderr = streams.Err

				if err := session.Shell(); err != nil {
					return fmt.Errorf("unable to start shell: %w", err)
				}

				sigWinChChan := make(chan os.Signal, 1)
				signal.Notify(sigWinChChan, syscall.SIGWINCH)
				go func() {
					for range sigWinChChan {
						fd := int(os.Stdin.Fd())
						width, height, err := term.GetSize(fd)
						if err != nil {
							log.Warnf("unable to get terminal size: %v", err)
							continue
						}
						if err := session.WindowChange(height, width); err != nil {
							log.Warnf("unable to change remote window size: %v", err)
						}
					}
				}()

				oState, err := term.MakeRaw(fd)
				if err != nil {
					return fmt.Errorf("unable to make raw terminal: %w", err)
				}
				defer func() {
					err := term.Restore(fd, oState)
					if err != nil {
						log.Errorf("unable to restore terminal: %v", err)
					}
				}()

				err = session.Wait()
				if err != nil {
					return fmt.Errorf("unable to wait: %w", err)
				}

			case opts.Forward:
			// TODO #957
			default:
				fmt.Fprintf(cmd.OutOrStderr(), "unknown action flag")
			}

			return nil
		},
	}
	cmd.Flags().StringVarP(&opts.Context, "context", "c", node.DefaultContextName, "Capability Context")
	cmd.Flags().StringVarP(&opts.ID, "id", "i", "", "Deployment ID")
	cmd.Flags().BoolVar(&opts.Shell, "shell", false, "Attach a Shell")
	cmd.Flags().BoolVar(&opts.Forward, "forward", false, "Attach a Forwarder")
	cmd.Flags().StringVarP(&opts.Alloc, "alloc", "a", "", "Allocation Name")
	cmd.Flags().StringVarP(&opts.Username, "username", "u", "", "Username for SSH Shell")
	cmd.Flags().StringVarP(&opts.Identity, "identity", "I", "", "SSH Private Key for SSH Shell")
	cmd.Flags().StringVarP(&opts.Port, "port", "p", "22", "Port for SSH Shell")
	cmd.Flags().StringVarP(&opts.PortMap, "portmap", "P", "", "Port Mapping <host:alloc> for Forwarder")

	err := cmd.MarkFlagRequired("context")
	if err != nil {
		log.Fatalf("unable to mark flag 'context' as required: %v", err)
	}
	err = cmd.MarkFlagRequired("id")
	if err != nil {
		log.Fatalf("unable to mark flag 'id' as required: %v", err)
	}
	cmd.MarkFlagsMutuallyExclusive("shell", "forward")
	cmd.MarkFlagsMutuallyExclusive("shell", "portmap")
	cmd.MarkFlagsMutuallyExclusive("forward", "username")
	cmd.MarkFlagsMutuallyExclusive("forward", "identity")
	cmd.MarkFlagsMutuallyExclusive("forward", "port")
	cmd.MarkFlagsRequiredTogether("shell", "username", "identity", "port")
	cmd.MarkFlagsRequiredTogether("forward", "portmap")
	// network attach --context dag --shell root@alloc1:22 -i <id>
	return cmd
}

func getDeploymentIDs(ctx context.Context, dmsClient client.DmsClient) ([]string, error) {
	resp, err := dmsClient.DeploymentList(
		ctx,
		node.DeploymentListRequest{},
		client.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("error getting deployment list from client: %w", err)
	}

	ids := make([]string, len(resp.Deployments))

	for r, status := range resp.Deployments {
		// only running deployments
		if status == jobtypes.DeploymentStatusRunning.String() {
			ids = append(ids, r)
		}
	}

	return ids, nil
}

func getNetworkList(ctx context.Context, ids []string, dmsClient client.DmsClient) ([]DeploymentNetwork, error) {
	depNet := make([]DeploymentNetwork, 0)
	for _, i := range ids {
		resp, err := dmsClient.DeploymentManifest(
			ctx,
			node.DeploymentManifestRequest{
				ID: i,
			},
			client.WithTimeout(5*time.Second),
		)
		if err != nil {
			return nil, fmt.Errorf("unable to get deployment manifest(id=%s) from client: %w", i, err)
		}
		if resp.Manifest.Subnet.Join {
			// "network" is only if orchestator joined the subnet
			allocs := make([]Allocation, 0)
			for allocID, alloc := range resp.Manifest.Allocations {
				allocs = append(allocs, Allocation{
					Alloc:       allocID,
					DNSName:     alloc.DNSName,
					IP:          alloc.PrivAddr,
					PortMapping: alloc.Ports,
					Status:      string(alloc.Status),
				})
			}
			depNet = append(depNet, DeploymentNetwork{
				ID:          resp.Manifest.ID,
				Allocations: allocs,
			})
		}
	}

	return depNet, nil
}

func getNetwork(ctx context.Context, id string, dmsClient client.DmsClient) (DeploymentNetwork, error) {
	resp, err := dmsClient.DeploymentManifest(
		ctx,
		node.DeploymentManifestRequest{
			ID: id,
		},
		client.WithTimeout(5*time.Second),
	)
	if err != nil {
		return DeploymentNetwork{}, fmt.Errorf("unable to get deployment manifest(id=%s) from client: %w", id, err)
	}

	if resp.Manifest.Subnet.Join {
		// "network" is only if orchestator joined the subnet
		allocs := make([]Allocation, 0)
		for allocID, alloc := range resp.Manifest.Allocations {
			allocs = append(allocs, Allocation{
				Alloc:       allocID,
				DNSName:     alloc.DNSName,
				IP:          alloc.PrivAddr,
				PortMapping: alloc.Ports,
				Status:      string(alloc.Status),
			})
		}

		return DeploymentNetwork{
			ID:          resp.Manifest.ID,
			Allocations: allocs,
		}, nil
	}

	return DeploymentNetwork{}, fmt.Errorf("the deployment does not have an accessible network")
}

// Hostkey manager
type HostKeyManager struct {
	knownHostsPath string
	keys           map[string]string
}

func NewHostKeyManager(afs afero.Afero, knownHostsPath string) (*HostKeyManager, error) {
	hostKeyManager := &HostKeyManager{
		knownHostsPath: knownHostsPath,
		keys:           make(map[string]string),
	}

	// create dir+file if not exists
	if _, err := afs.Stat(knownHostsPath); os.IsNotExist(err) {
		if err := afs.MkdirAll(path.Dir(knownHostsPath), 0o700); err != nil {
			return nil, fmt.Errorf("unable to create known_hosts dir: %w", err)
		}
		if _, err := afs.Create(knownHostsPath); err != nil {
			log.Fatalf("unable to create known_hosts file: %v", err)
		}
	}

	// read known_hosts file
	data, err := os.ReadFile(knownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read known_hosts file: %w", err)
	}

	records := strings.Split(string(data), "\n")
	for _, record := range records {
		record = strings.TrimSpace(record)
		if record == "" {
			continue
		}

		fields := strings.Fields(record)
		if len(fields) >= 2 {
			if err != nil {
				return nil, fmt.Errorf("unable to parse known_hosts key: %w", err)
			}
			hostKeyManager.keys[fields[0]] = fields[1]
		}
	}

	return hostKeyManager, nil
}

func (h *HostKeyManager) saveRecord(hostname string, key string) error {
	// write to file
	f, err := os.OpenFile(h.knownHostsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("unable to open known_hosts file: %w", err)
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "%s %s\n", hostname, key)
	if err != nil {
		return fmt.Errorf("unable to write known_hosts file: %w", err)
	}

	return nil
}

func (h *HostKeyManager) HostKeyCallback(out io.Writer) ssh.HostKeyCallback {
	return func(hostname string, _ net.Addr, key ssh.PublicKey) error {
		// encode key to string
		keyStr := base64.StdEncoding.EncodeToString(key.Marshal())

		stored, ok := h.keys[hostname]
		if !ok {
			fmt.Fprintf(
				out,
				"Unknown host key for %s\nFingerprint: %s\n\n",
				hostname,
				keyStr,
			)
			yes, err := dmsUtils.PromptYesNo(os.Stdin, out, "Are you sure you want to proceed?")
			if err != nil {
				return fmt.Errorf("unable to prompt for host key verification: %w", err)
			}
			if !yes {
				return fmt.Errorf("host key verification failed")
			}

			// save the key
			if err := h.saveRecord(hostname, keyStr); err != nil {
				return fmt.Errorf("unable to save host key: %w", err)
			}

			return nil
		}

		// if stored key exists
		if stored != keyStr {
			return fmt.Errorf("host key verification failed")
		}

		return nil
	}
}
