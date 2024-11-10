// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version information set by the build system (see Makefile)
var (
	Version   string
	GoVersion string
	BuildDate string
	Commit    string
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Information about current version",
		Long:  `Display information about the current Device Management Service (DMS) version`,
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println("NuNet Device Management Service")
			fmt.Printf("Version: %s\nCommit: %s\n\nGo Version: %s\nBuild Date: %s\n",
				Version, Commit, GoVersion, BuildDate)
		},
	}
}
