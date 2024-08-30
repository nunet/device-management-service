//go:build linux

package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/cmd/backend"
)

const (
	logDir    = "/tmp/nunet-log"
	dmsUnit   = "nunet-dms.service"
	tarGzName = "nunet-log.tar.gz"
)

var logCmd *cobra.Command

func NewLogCmd(net backend.NetworkManager, fs backend.FileSystem, journal backend.Logger) *cobra.Command {
	return &cobra.Command{
		Use:     "log",
		Short:   "Gather all logs into a tarball. COMMAND MUST RUN AS ROOT WITH SUDO",
		PreRunE: isDMSRunning(net),
		RunE: func(cmd *cobra.Command, _ []string) error {
			dmsLogDir := filepath.Join(logDir, "dms-log")

			fmt.Fprintln(cmd.OutOrStdout(), "Collecting logs...")

			err := fs.MkdirAll(dmsLogDir, 0o777)
			if err != nil {
				return fmt.Errorf("cannot create dms-log directory: %w", err)
			}

			if journal == nil {
				return fmt.Errorf("journal service is not initialised")
			}

			// journal is initialized in init.go
			defer journal.Close()

			// filter by service unit name
			match := fmt.Sprintf("_SYSTEMD_UNIT=%s", dmsUnit)

			err = journal.AddMatch(match)
			if err != nil {
				return fmt.Errorf("cannot add unit match: %w", err)
			}

			var count int

			// loop through journal entries
			for {
				c, err := journal.Next()
				if err != nil {
					fmt.Fprintf(cmd.OutOrStderr(), "Error reading next journal entry: %v\n", err)
					continue
				}

				if c == 0 {
					break
				}

				entry, err := journal.GetEntry()
				if err != nil {
					fmt.Fprintf(cmd.OutOrStderr(), "Error getting journal entry %d: %v\n", count, err)
					continue
				}

				msg, ok := entry.Fields["MESSAGE"]
				if !ok {
					fmt.Fprintf(cmd.OutOrStderr(), "Error: no message field in entry %d\n", count)
				}

				logData := fmt.Sprintf("%d: %s\n", entry.RealtimeTimestamp, msg)

				logFilePath := filepath.Join(dmsLogDir, fmt.Sprintf("dms_log.%d", count))

				err = appendToFile(fs, logFilePath, logData)
				if err != nil {
					fmt.Fprintf(cmd.OutOrStderr(), "Error writing log file for boot %d: %v\n", count, err)
				}

				count++
			}

			if count == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No log entries")
				return nil
			}

			tarGzFile := filepath.Join(logDir, tarGzName)

			err = createTar(fs, tarGzFile, dmsLogDir)
			if err != nil {
				return fmt.Errorf("cannot create tar.gz file: %w", err)
			}

			err = fs.RemoveAll(dmsLogDir)
			if err != nil {
				return fmt.Errorf("remove dms-log directory failed: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), tarGzFile)
			return nil
		},
	}
}
