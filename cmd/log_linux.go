//go:build linux

package cmd

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

func createTar(afs afero.Afero, tarGzPath string, sourceDir string) error {
	tarGzFile, err := afs.Create(tarGzPath)
	if err != nil {
		return fmt.Errorf("create %s file failed: %w", tarGzPath, err)
	}
	defer tarGzFile.Close()

	gzWriter := gzip.NewWriter(tarGzFile)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	return afs.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == tarGzPath {
			return nil
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		header.Name = strings.TrimPrefix(path, sourceDir)
		if header.Name == "" || header.Name == "/" {
			return nil
		}

		err = tarWriter.WriteHeader(header)
		if err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			data, err := afs.ReadFile(path)
			if err != nil {
				return err
			}

			_, err = tarWriter.Write(data)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// appendToFile opens filename and write string data to it
func appendToFile(afs afero.Afero, filename, data string) error {
	f, err := afs.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open %s file failed: %w", filename, err)
	}
	defer f.Close()

	_, err = f.WriteString(data)
	if err != nil {
		return fmt.Errorf("write string data to file %s failed: %w", filename, err)
	}

	return nil
}
