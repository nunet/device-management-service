// s3/input_source.go
package s3

import (
	"fmt"

	"github.com/fatih/structs"
	"github.com/go-viper/mapstructure/v2"

	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

type InputSource struct {
	Bucket   string
	Key      string
	Filter   string
	Region   string
	Endpoint string
}

func (s InputSource) Validate() error {
	if s.Bucket == "" {
		err := fmt.Errorf("invalid s3 storage params: bucket cannot be empty")
		log.Errorw("s3_input_source_validation_failure", "error", err)
		return err
	}
	return nil
}

func (s InputSource) ToMap() map[string]interface{} {
	return structs.Map(s)
}

func DecodeInputSpec(spec *types.SpecConfig) (InputSource, error) {
	endTrace := observability.StartTrace("decode_input_spec_duration")
	defer endTrace()

	if !spec.IsType(types.StorageProviderS3) {
		err := fmt.Errorf("invalid storage source type. Expected %s but received %s", types.StorageProviderS3, spec.Type)
		log.Errorw("decode_input_spec_invalid_type_failure", "error", err)
		return InputSource{}, err
	}

	inputParams := spec.Params
	if inputParams == nil {
		err := fmt.Errorf("invalid storage input source params. cannot be nil")
		log.Errorw("decode_input_spec_nil_params_failure", "error", err)
		return InputSource{}, err
	}

	var c InputSource
	if err := mapstructure.Decode(spec.Params, &c); err != nil {
		log.Errorw("decode_input_spec_decode_failure", "error", err)
		return c, err
	}

	if err := c.Validate(); err != nil {
		log.Errorw("decode_input_spec_validation_failure", "error", err)
		return c, err
	}

	log.Infow("decode_input_spec_success", "bucket", c.Bucket)
	return c, nil
}
