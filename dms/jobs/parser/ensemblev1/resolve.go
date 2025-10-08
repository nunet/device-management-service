// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package ensemblev1

import (
	"gitlab.com/nunet/device-management-service/dms/jobs/parser/resolve"
	"gitlab.com/nunet/device-management-service/dms/jobs/parser/tree"
	"gitlab.com/nunet/device-management-service/dms/jobs/parser/types"
)

func resolvePlaceholders(data *any, options *types.Options) error {
	resolver := resolve.NewResolver(
		map[string]resolve.Handler{
			"env":  resolve.NewEnvResolver(options.Env),
			"file": resolve.NewFileResolver(options.Fs, options.WorkingDir),
		},
		nil,
	)
	return tree.Walk(data, tree.NewPath(), func(node *any, _ tree.Path) error {
		if strVal, ok := (*node).(string); ok {
			interpolated, err := resolver.Process(strVal)
			if err != nil {
				return err
			}
			*node = interpolated
		}
		return nil
	})
}
