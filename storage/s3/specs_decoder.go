package s3

import (
	"fmt"

	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"

	"gitlab.com/nunet/device-management-service/types"
)

type S3InputSource struct {
	Bucket   string
	Key      string
	Filter   string
	Region   string
	Endpoint string
}

func (s S3InputSource) Validate() error {
	if s.Bucket == "" {
		return fmt.Errorf("invalid s3 storage params: bucket cannot be empty")
	}
	return nil
}

func (s S3InputSource) ToMap() map[string]interface{} {
	return structs.Map(s)
}

func DecodeInputSpec(spec *types.SpecConfig) (S3InputSource, error) {
	if !spec.IsType(types.StorageProviderS3) {
		return S3InputSource{}, fmt.Errorf("invalid storage source type. Expected %s but received %s", types.StorageProviderS3, spec.Type)
	}

	inputParams := spec.Params
	if inputParams == nil {
		return S3InputSource{}, fmt.Errorf("invalid storage input source params. cannot be nil")
	}

	var c S3InputSource
	if err := mapstructure.Decode(spec.Params, &c); err != nil {
		return c, err
	}

	return c, c.Validate()
}
