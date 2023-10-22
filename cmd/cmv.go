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
	"os"

	"github.com/suixibing/cocom/cmd/cmv"
	"github.com/suixibing/cocom/pkg/clog"

	"github.com/spf13/cobra"
)

// cmvCmd represents the cmv command
var cmvCmd = &cobra.Command{
	Use:   "cmv",
	Short: "Move comic gallery to save directory",
	Long:  `Move comic gallery to save directory.`,
	Run: func(cmd *cobra.Command, args []string) {
		clog.Debugf(cmd.Context(), "cmv called")
		manager := cmv.NewComicMoveManager()
		err := manager.Handle(cmd.Context())
		if err != nil {
			clog.Errorf(cmd.Context(), "comic move manager handle failed: %#v", err)
			fmt.Fprintf(os.Stderr, "comic move manager handle failed: %#v", err)
			return
		}
		clog.Debugf(cmd.Context(), "comic move manager handle succ")
	},
}

func init() {
	rootCmd.AddCommand(cmvCmd)

	cmvCmd.Flags().StringVarP(&cmv.DefaultConfig.ComicRegexRuleRaw, "regexRule", "r", `^\[(\d+)\].*$`, "comic regex rule")
	cmvCmd.Flags().StringVarP(&cmv.DefaultConfig.SrcPath, "srcPath", "s", ".", "comic source path")
	cmvCmd.Flags().StringVarP(&cmv.DefaultConfig.DstRootPath, "dstRootPath", "d", ".", "comic destination root path")
	cmvCmd.Flags().StringVarP(&cmv.DefaultConfig.Output, "output", "o", "cmv.sh", "output shell script")
	cmvCmd.Flags().StringSliceVar(&cmv.DefaultConfig.SkipDirs, "skipDirs", []string{}, "skip not matched directory list")
	cmvCmd.Flags().BoolVarP(&cmv.DefaultConfig.IgnoreNotMatch, "ignoreNotMatch", "i", false, "ignore not matched directory")
	cmvCmd.Flags().BoolVar(&cmv.DefaultConfig.SkipFail, "skipFail", true, "skip failed directory")
}
