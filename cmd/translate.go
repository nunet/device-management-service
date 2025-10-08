// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/cmd/cli"
	"gitlab.com/nunet/device-management-service/dms/jobs/parser"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/dms/translator"
)

type TranslateOptions struct {
	InputFile  string
	FromFormat string
	OutputFile string
}

func newTranslateCmd(
	dmsCli *cli.DmsCLI,
) *cobra.Command {
	var opts TranslateOptions

	cmd := &cobra.Command{
		Use:   "translate <input-file>",
		Short: "Translate a foreign specification to a NuNet Ensemble configuration.",
		Long: `Translate a foreign specification, such as a Docker Compose file,
into a native NuNet DMS Ensemble configuration file.

This allows you to leverage existing development files and easily onboard them
onto the NuNet platform.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.InputFile = args[0]

			return runTranslateCmd(cmd.Context(), dmsCli, opts, cli.CmdStreams(cmd))
		},
	}
	cmd.Flags().StringVarP(&opts.FromFormat, "from", "f", "docker-compose", "The source format of the input file (e.g., 'docker-compose').")
	cmd.Flags().StringVarP(&opts.OutputFile, "output", "o", "", "Path to the output NuNet Ensemble file.")
	return cmd
}

func runTranslateCmd(_ context.Context, dmsCli *cli.DmsCLI, opts TranslateOptions, streams cli.Streams) error {
	fs := afero.Afero{Fs: dmsCli.FS()}

	// Read the input file content
	inputBytes, err := fs.ReadFile(opts.InputFile)
	if err != nil {
		return fmt.Errorf("could not read input file '%s': %w", opts.InputFile, err)
	}

	// Perform the translation
	translation, err := translator.Translate(translator.SpecType(opts.FromFormat), inputBytes)
	if err != nil {
		return fmt.Errorf("translation failed: %w", err)
	}

	data, err := parser.Encode(parser.SpecTypeEnsembleV1, translation.Config)
	if err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	if opts.OutputFile == "" {
		fmt.Fprintln(streams.Out, string(data))
	} else {
		if err := fs.WriteFile(opts.OutputFile, data, 0o644); err != nil {
			return fmt.Errorf("failed to write output file '%s': %w", opts.OutputFile, err)
		}
	}

	// Print a success message and any warnings to stderr
	if opts.OutputFile != "" {
		fmt.Fprintf(streams.Out, "Successfully translated '%s' to '%s'.\n", opts.InputFile, opts.OutputFile)
	}
	if len(translation.Warnings) > 0 {
		fmt.Fprintln(streams.Err, "\nPlease review the following warnings (also included as comments in the output file):")
		for _, warning := range translation.Warnings {
			fmt.Fprintf(streams.Err, " - %s\n", warning)
		}
	}
	return nil
}

type ValidateOpts struct {
	InputFile string
}

func newValidateCmd(
	dmsCli *cli.DmsCLI,
) *cobra.Command {
	var opts ValidateOpts

	cmd := &cobra.Command{
		Use:   "validate <input-file>",
		Short: "Validate a NuNet Ensemble configuration.",
		Long: `Parses a NuNet Ensemble configuration file and validate if the configuration is valid.
	
Example:
  nunet validate ensemble.yaml
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.InputFile = args[0]
			return runValidateCmd(cmd.Context(), dmsCli, opts, cli.CmdStreams(cmd))
		},
	}
	return cmd
}

func runValidateCmd(_ context.Context, dmsCli *cli.DmsCLI, opts ValidateOpts, streams cli.Streams) error {
	fs := afero.Afero{Fs: dmsCli.FS()}

	// Read the input file content
	inputBytes, err := fs.ReadFile(opts.InputFile)
	if err != nil {
		return fmt.Errorf("could not read input file '%s': %w", opts.InputFile, err)
	}

	var cfg jobtypes.EnsembleConfig
	err = parser.Decode(parser.SpecTypeEnsembleV1, inputBytes, &cfg, &parser.Options{
		Env:        dmsCli.Env(),
		Fs:         fs,
		WorkingDir: "",
	})
	if err != nil {
		return err
	}
	err = cfg.Validate()
	if err != nil {
		return err
	}
	fmt.Fprintln(streams.Out, "Configuration is valid.")
	return nil
}
