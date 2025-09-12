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
