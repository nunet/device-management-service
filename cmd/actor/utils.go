// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package actor

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/dms/jobs/parser"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/lib/env"
)

func displayResponse(w io.Writer, resp any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(resp)
}

func ProcessEnsembleYaml(fs afero.Afero, env env.EnvironmentProvider, path string) (
	*jobtypes.EnsembleConfig, error,
) {
	data, err := fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := &jobtypes.EnsembleConfig{}
	err = parser.Decode(parser.SpecTypeEnsembleV1, data, &cfg, &parser.Options{
		Env:        env,
		Fs:         fs,
		WorkingDir: "",
	})
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// parseDateString parses a date string supporting both relative formats (e.g., "7d", "12h") and absolute formats (RFC3339, common date formats)
func parseDateString(dateStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return time.Time{}, fmt.Errorf("empty date string")
	}

	// Try relative formats first
	if strings.HasSuffix(dateStr, "d") {
		// days is not a standard Go duration; handle explicitly
		daysStr := strings.TrimSuffix(dateStr, "d")
		if daysStr == "" {
			return time.Time{}, fmt.Errorf("invalid date duration: %s", dateStr)
		}
		if nDays, err := strconv.Atoi(daysStr); err == nil && nDays > 0 {
			return time.Now().AddDate(0, 0, -nDays), nil
		}
		return time.Time{}, fmt.Errorf("invalid date duration days: %s", dateStr)
	}

	// Try standard duration formats
	if dur, err := time.ParseDuration(dateStr); err == nil {
		return time.Now().Add(-dur), nil
	}

	// Try datetime formats
	var parseErr error
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		t, err := time.Parse(layout, dateStr)
		if err == nil {
			return t, nil
		}
		parseErr = err
	}

	return time.Time{}, fmt.Errorf("invalid date format: %w", parseErr)
}
