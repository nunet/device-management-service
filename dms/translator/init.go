package translator

import (
	"gitlab.com/nunet/device-management-service/dms/translator/dockercompose"
	"gitlab.com/nunet/device-management-service/dms/translator/types"
)

var registry *Registry

func init() {
	registry = &Registry{
		translators: make(map[SpecType]types.Translator),
	}

	registry.RegisterTranslator(SpecTypeDockerCompose, dockercompose.NewDockerComposeTranslator())
}
