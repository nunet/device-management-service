//go:build linux

package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/cmd/backend"
)

const (
	logDir    = "/tmp/nunet-log"
	dmsUnit   = "nunet.service"
	tarGzName = "nunet-log.tar.gz"
)

// NewLogCmd is a constructor for `log` command
func newLogCmd(afs afero.Afero, loggerArg interface{}) *cobra.Command {
	return &cobra.Command{
		Use:   "log",
		Short: "Gather all logs into a tarball. COMMAND MUST RUN AS ROOT WITH SUDO",
		RunE: func(cmd *cobra.Command, _ []string) error {
			dmsLogDir := filepath.Join(logDir, "dms-log")

			fmt.Fprintln(cmd.OutOrStdout(), "Collecting logs...")

			err := afs.MkdirAll(dmsLogDir, 0o777)
			if err != nil {
				return fmt.Errorf("cannot create dms-log directory: %w", err)
			}

			logger, ok := loggerArg.(backend.Logger)
			if !ok {
				return fmt.Errorf("unknown logger: %v", loggerArg)
			}

			defer logger.Close()

			// filter by service unit name
			match := fmt.Sprintf("_SYSTEMD_UNIT=%s", dmsUnit)

			err = logger.AddMatch(match)
			if err != nil {
				return fmt.Errorf("cannot add unit match: %w", err)
			}

			var counter int
			for {
				count, err := logger.Next()
				if err != nil {
					fmt.Fprintf(cmd.OutOrStderr(), "Error reading next logger entry: %v\n", err)
					continue
				}

				if count == 0 {
					break
				}

				logEntry, err := logger.GetEntry()
				if err != nil {
					fmt.Fprintf(cmd.OutOrStderr(), "Error getting logger entry %d: %v\n", count, err)
					continue
				}

				msg, ok := logEntry.Fields["MESSAGE"]
				if !ok {
					fmt.Fprintf(cmd.OutOrStderr(), "Error: no message field in entry %d\n", count)
				}

				logData := fmt.Sprintf("%d: %s\n", logEntry.RealtimeTimestamp, msg)

				logFilePath := filepath.Join(dmsLogDir, fmt.Sprintf("dms_log.%d", count))

				err = appendToFile(afs, logFilePath, logData)
				if err != nil {
					fmt.Fprintf(cmd.OutOrStderr(), "Error writing log file for boot %d: %v\n", count, err)
				}
				counter++
			}

			if counter == 0 {
				return fmt.Errorf("no log entries for %s", dmsUnit)
			}

			tarGzFile := filepath.Join(logDir, tarGzName)

			err = createTar(afs, tarGzFile, dmsLogDir)
			if err != nil {
				return fmt.Errorf("cannot create tar.gz file: %w", err)
			}

			err = afs.RemoveAll(dmsLogDir)
			if err != nil {
				return fmt.Errorf("remove dms-log directory failed: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), tarGzFile)
			return nil
		},
	}
}
