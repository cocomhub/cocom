// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	"github.com/cocomhub/cocom/cmd/cmv"
	"github.com/cocomhub/cocom/pkg/clog"

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

// galleryMvCmd represents the gallery move subcommand
var galleryMvCmd = &cobra.Command{
	Use:   "mv",
	Short: "移动漫画图库到目标目录",
	Long:  "根据规则将漫画图库移动到目标保存目录。",
	Run: func(cmd *cobra.Command, args []string) {
		clog.Debugf(cmd.Context(), "gallery move called")
		manager := cmv.NewComicMoveManager()
		err := manager.Handle(cmd.Context())
		if err != nil {
			clog.Errorf(cmd.Context(), "gallery move handle failed: %#v", err)
			fmt.Fprintf(os.Stderr, "gallery move handle failed: %#v", err)
			return
		}
		clog.Debugf(cmd.Context(), "gallery move handle succ")
	},
}

func init() {
	rootCmd.AddCommand(cmvCmd)
	galleryCmd.AddCommand(galleryMvCmd)

	cmvCmd.Flags().StringVarP(&cmv.DefaultConfig.ComicRegexRuleRaw, "regexRule", "r", `^\[(\d+)\].*$`, "comic regex rule")
	cmvCmd.Flags().StringVarP(&cmv.DefaultConfig.SrcPath, "srcPath", "s", ".", "comic source path")
	cmvCmd.Flags().StringVarP(&cmv.DefaultConfig.DstRootPath, "dstRootPath", "d", ".", "comic destination root path")
	cmvCmd.Flags().StringVarP(&cmv.DefaultConfig.Output, "output", "o", "cmv.sh", "output shell script")
	cmvCmd.Flags().StringSliceVar(&cmv.DefaultConfig.SkipDirs, "skipDirs", []string{}, "skip not matched directory list")
	cmvCmd.Flags().BoolVarP(&cmv.DefaultConfig.IgnoreNotMatch, "ignoreNotMatch", "i", false, "ignore not matched directory")
	cmvCmd.Flags().BoolVar(&cmv.DefaultConfig.SkipFail, "skipFail", true, "skip failed directory")

	// reuse the same flags on gallery move
	galleryMvCmd.Flags().StringVarP(&cmv.DefaultConfig.ComicRegexRuleRaw, "regexRule", "r", `^\[(\d+)\].*$`, "comic regex rule")
	galleryMvCmd.Flags().StringVarP(&cmv.DefaultConfig.SrcPath, "srcPath", "s", ".", "comic source path")
	galleryMvCmd.Flags().StringVarP(&cmv.DefaultConfig.DstRootPath, "dstRootPath", "d", ".", "comic destination root path")
	galleryMvCmd.Flags().StringVarP(&cmv.DefaultConfig.Output, "output", "o", "cmv.sh", "output shell script")
	galleryMvCmd.Flags().StringSliceVar(&cmv.DefaultConfig.SkipDirs, "skipDirs", []string{}, "skip not matched directory list")
	galleryMvCmd.Flags().BoolVarP(&cmv.DefaultConfig.IgnoreNotMatch, "ignoreNotMatch", "i", false, "ignore not matched directory")
	galleryMvCmd.Flags().BoolVar(&cmv.DefaultConfig.SkipFail, "skipFail", true, "skip failed directory")
}
