package types

import (
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
)

// Translation represents the result of a translation, including the configuration
// and any warnings about unsupported features.
type Translation struct {
	Config   *jobtypes.EnsembleConfig
	Warnings []string
}

// Translator defines the interface for converting a source configuration file
// into a NuNet EnsembleConfig.
type Translator interface {
	Translate(input []byte) (*Translation, error)
}
