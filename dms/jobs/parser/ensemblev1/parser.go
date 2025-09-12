package ensemblev1

import "gitlab.com/nunet/device-management-service/dms/jobs/parser/types"

func NewEnsemblev1Parser() types.BasicParser {
	return types.NewBasicParser(
		"yaml",
		resolvePlaceholders,
		NewEnsemblev1Decoder(),
		NewEnsemblev1Encoder(),
		NewEnsembleV1Validator(),
	)
}
