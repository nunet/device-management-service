package s3

import (
	"context"
	"fmt"

	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"

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
		st.Error(context.Background(), "s3_input_source_validation_failure", nil)
		return err
	}
	return nil
}

func (s InputSource) ToMap() map[string]interface{} {
	return structs.Map(s)
}

func DecodeInputSpec(spec *types.SpecConfig) (InputSource, error) {
	ctx, cancel := st.SpanContext(context.Background(), "s3", "decode_input_spec_duration", "opentelemetry", "log")
	defer cancel()

	if !spec.IsType(types.StorageProviderS3) {
		err := fmt.Errorf("invalid storage source type. Expected %s but received %s", types.StorageProviderS3, spec.Type)
		ctx = context.WithValue(ctx, errorKey, err.Error())
		st.Error(ctx, "decode_input_spec_invalid_type_failure", nil)
		return InputSource{}, err
	}

	inputParams := spec.Params
	if inputParams == nil {
		err := fmt.Errorf("invalid storage input source params. cannot be nil")
		ctx = context.WithValue(ctx, errorKey, err.Error())
		st.Error(ctx, "decode_input_spec_nil_params_failure", nil)
		return InputSource{}, err
	}

	var c InputSource
	if err := mapstructure.Decode(spec.Params, &c); err != nil {
		ctx = context.WithValue(ctx, errorKey, err.Error())
		st.Error(ctx, "decode_input_spec_decode_failure", nil)
		return c, err
	}

	if err := c.Validate(); err != nil {
		ctx = context.WithValue(ctx, errorKey, err.Error())
		st.Error(ctx, "decode_input_spec_validation_failure", nil)
		return c, err
	}

	st.Info(ctx, "decode_input_spec_success", nil)
	return c, nil
}
