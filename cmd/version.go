// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/cocomhub/cocom/pkg/version"

	"github.com/spf13/cobra"
)

type versionFlags struct {
	outputFile string
	outputJSON bool
	format     string
}

var versionFlag versionFlags

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Version for command build",
	Long: `Version for command build:

    version, commit sha, latest version, date
`,
	Run: func(cmd *cobra.Command, args []string) {
		var w io.Writer = os.Stdout
		if versionFlag.outputFile != "" {
			f, err := os.Create(versionFlag.outputFile)
			if err != nil {
				fmt.Println("Failed to create file:", err)
				return
			}
			defer f.Close()
			w = f
		}

		if versionFlag.outputJSON {
			version.PrintVersionJSON(w)
		} else {
			version.PrintVersion(w, versionFlag.format)
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
	versionCmd.Flags().StringVarP(&versionFlag.outputFile, "output", "o", "", "Output the version information to a file")
	versionCmd.Flags().BoolVarP(&versionFlag.outputJSON, "json", "j", false, "Output version information in JSON format")
	versionCmd.Flags().StringVarP(&versionFlag.format, "format", "f", "", "Format the output")
}
