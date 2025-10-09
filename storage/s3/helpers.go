// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

// s3/aws_config.go
package s3

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"

	"gitlab.com/nunet/device-management-service/observability"
)

// GetAWSDefaultConfig returns the default AWS config based on environment variables,
// shared configuration and shared credentials files.
func GetAWSDefaultConfig() (aws.Config, error) {
	endSpan := observability.StartSpan("get_aws_default_config")
	defer endSpan()

	var optFns []func(*config.LoadOptions) error
	cfg, err := config.LoadDefaultConfig(context.Background(), optFns...)
	if err != nil {
		log.Errorw("get_aws_default_config_failure", "error", err)
		return aws.Config{}, err
	}

	log.Infow("get_aws_default_config_success")
	return cfg, nil
}

// hasValidCredentials checks if the provided AWS config has valid credentials.
func hasValidCredentials(config aws.Config) bool {
	endSpan := observability.StartSpan("has_valid_credentials")
	defer endSpan()

	credentials, err := config.Credentials.Retrieve(context.Background())
	if err != nil {
		log.Errorw("has_valid_credentials_failure", "error", err)
		return false
	}

	if !credentials.HasKeys() {
		log.Errorw("has_valid_credentials_failure_no_keys")
		return false
	}

	log.Infow("has_valid_credentials_success")
	return true
}

// sanitizeKey removes trailing spaces and wildcards
func sanitizeKey(key string) string {
	endSpan := observability.StartSpan("sanitize_key")
	defer endSpan()

	sanitizedKey := strings.TrimSuffix(strings.TrimSpace(key), "*")
	log.Infow("sanitize_key_success", "sanitizedKey", sanitizedKey)

	return sanitizedKey
}
