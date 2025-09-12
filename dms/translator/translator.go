package translator

import (
	"fmt"

	"gitlab.com/nunet/device-management-service/dms/translator/types"
)

type SpecType string

const (
	SpecTypeDockerCompose SpecType = "docker-compose"
)

func Translate(specType SpecType, input []byte) (*types.Translation, error) {
	translator, found := registry.GetTranslator(specType)
	if !found {
		return nil, fmt.Errorf("translator for spec type %s not found", specType)
	}
	return translator.Translate(input)
}
