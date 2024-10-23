package jobs

import (
	"errors"
)

var (
	ErrProvisioningFailed = errors.New("failed to provision the ensemble")
	ErrDeploymentFailed   = errors.New("failed to create deployment")
	ErrTODO               = errors.New("TODO")
)
