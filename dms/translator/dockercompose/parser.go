package dockercompose

import (
	"context"

	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/types"
)

// Parse takes the content of a docker-compose.yml file and returns a parsed Project object.
func Parse(content []byte) (*types.Project, error) {
	configDetails := types.ConfigDetails{
		ConfigFiles: []types.ConfigFile{
			{
				Content: content,
			},
		},
	}

	// The loader.LoadWithContext function handles parsing and validation of the compose file.
	// NOTE: should we add context to the Parse function and pass it here?
	project, err := loader.LoadWithContext(context.Background(), configDetails, func(opts *loader.Options) { opts.SetProjectName("project", true) })
	if err != nil {
		return nil, err
	}

	return project, nil
}
