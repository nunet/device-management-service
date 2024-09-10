package dms

import (
	"os"

	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/telemetry/logger"
)

var zlog *logger.Logger

func init() {
	zlog = logger.New("dms")

	fs := afero.NewOsFs()
	userDir := config.GetConfig().UserDir
	if userDir != "" {
		err := fs.MkdirAll(userDir, os.FileMode(0o700))
		if err != nil {
			log.Warnf("unable to create user directory: %v", err)
		}
	}
}
