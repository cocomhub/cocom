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
	"github.com/suixibing/cocom/cmd/genwget"
	"github.com/suixibing/cocom/pkg/clog"
	"os"

	"github.com/spf13/cobra"
)

// genWgetCmd represents the genwget command
var genWgetCmd = &cobra.Command{
	Use:   "genwget",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		clog.Debugf(cmd.Context(), "gen wget called")
		err := genwget.NewManager().Handle(cmd.Context())
		if err != nil {
			clog.Errorf(cmd.Context(), "gen comic wget handle failed: %#v", err)
			fmt.Fprintf(os.Stderr, "gen comic wget handle failed: %#v", err)
			return
		}
		clog.Debugf(cmd.Context(), "gen comic wget handle succ")
	},
}

func init() {
	rootCmd.AddCommand(genWgetCmd)

	genWgetCmd.Flags().StringVarP(&genwget.DefaultConfig.Input, "input", "i", `input.txt`, "input source")
	genWgetCmd.Flags().StringVarP(&genwget.DefaultConfig.Output, "output", "o", "genwget.sh", "output source")
	genWgetCmd.Flags().StringVarP(&genwget.DefaultConfig.DstRootPath, "dstRootPath", "d", ".", "comic destination root path")
}
