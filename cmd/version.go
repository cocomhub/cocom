/*
Copyright © 2023 suixibing <suixibing@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/suixibing/cocom/pkg/version"

	"github.com/spf13/cobra"
)

var (
	outputFile string
	outputJSON bool
	format     string
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Version for command build",
	Long: `Version for command build:

    version, commit sha, latest version, date
`,
	Run: func(cmd *cobra.Command, args []string) {
		var w io.Writer = os.Stdout
		if outputFile != "" {
			f, err := os.Create(outputFile)
			if err != nil {
				fmt.Println("Failed to create file:", err)
				return
			}
			defer f.Close()
			w = f
		}

		if outputJSON {
			version.PrintVersionJSON(w)
		} else {
			version.PrintVersion(w, format)
		}
	},
}

var dirtyInfoCmd = &cobra.Command{
	Use:   "dirty-info",
	Short: "Show the differences from the last commit",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.DirtyInfo)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	versionCmd.AddCommand(dirtyInfoCmd)
	versionCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output the version information to a file")
	versionCmd.Flags().BoolVarP(&outputJSON, "json", "j", false, "Output version information in JSON format")
	versionCmd.Flags().StringVarP(&format, "format", "f", "", "Format the output")
}
