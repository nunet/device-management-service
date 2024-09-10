package dms

import (
	"os"

	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/internal/config"
)

func init() {
	fs := afero.NewOsFs()

	workDir := config.GetConfig().WorkDir
	if workDir != "" {
		err := fs.MkdirAll(workDir, os.FileMode(0o700))
		if err != nil {
			log.Warnf("unable to create work directory: %v", err)
		}
	}

	dataDir := config.GetConfig().DataDir
	if dataDir != "" {
		err := fs.MkdirAll(dataDir, os.FileMode(0o700))
		if err != nil {
			log.Warnf("unable to create data directory: %v", err)
		}
	}

	userDir := config.GetConfig().UserDir
	if userDir != "" {
		err := fs.MkdirAll(userDir, os.FileMode(0o700))
		if err != nil {
			log.Warnf("unable to create user directory: %v", err)
		}
	}
}
