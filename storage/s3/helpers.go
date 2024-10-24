// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package s3

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// GetAWSDefaultConfig returns the default AWS config based on environment variables,
// shared configuration and shared credentials files.
func GetAWSDefaultConfig() (aws.Config, error) {
	ctx, cancel := st.SpanContext(context.Background(), "s3", "get_aws_default_config_duration", "opentelemetry", "log")
	defer cancel()

	var optFns []func(*config.LoadOptions) error
	cfg, err := config.LoadDefaultConfig(ctx, optFns...)
	if err != nil {
		ctx = context.WithValue(ctx, errorKey, err.Error())
		st.Error(ctx, "get_aws_default_config_failure", nil)
		return aws.Config{}, err
	}

	st.Info(ctx, "get_aws_default_config_success", nil)
	return cfg, nil
}

// hasValidCredentials checks if the provided AWS config has valid credentials.
func hasValidCredentials(config aws.Config) bool {
	ctx, cancel := st.SpanContext(context.Background(), "s3", "has_valid_credentials_duration", "opentelemetry", "log")
	defer cancel()

	credentials, err := config.Credentials.Retrieve(ctx)
	if err != nil {
		ctx = context.WithValue(ctx, errorKey, err.Error())
		st.Error(ctx, "has_valid_credentials_failure", nil)
		return false
	}

	if !credentials.HasKeys() {
		st.Error(ctx, "has_valid_credentials_failure_no_keys", nil)
		return false
	}

	st.Info(ctx, "has_valid_credentials_success", nil)
	return true
}

// sanitizeKey removes trailing spaces and wildcards
func sanitizeKey(key string) string {
	ctx, cancel := st.SpanContext(context.Background(), "s3", "sanitize_key_duration", "opentelemetry", "log")
	defer cancel()

	sanitizedKey := strings.TrimSuffix(strings.TrimSpace(key), "*")
	ctx = context.WithValue(ctx, sanitizedKeyContext, sanitizedKey)
	st.Info(ctx, "sanitize_key_success", nil)

	return sanitizedKey
}
