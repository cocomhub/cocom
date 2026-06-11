// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cmv

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "cmv",
	Short: "移动漫画图库到目标目录",
	RunE: func(cmd *cobra.Command, args []string) error {
		slog.DebugContext(cmd.Context(), "cmv called")
		manager := NewComicMoveManager()
		if err := manager.Handle(cmd.Context()); err != nil {
			slog.ErrorContext(cmd.Context(), "comic move manager handle failed", slog.String("err", err.Error()))
			fmt.Fprintf(os.Stderr, "comic move manager handle failed: %#v", err)
			return err
		}
		slog.DebugContext(cmd.Context(), "comic move manager handle succ")
		return nil
	},
}

func init() {
	Cmd.Flags().StringVarP(&DefaultConfig.ComicRegexRuleRaw, "regexRule", "r", `^\[(\d+)\].*$`, "comic regex rule")
	Cmd.Flags().StringVarP(&DefaultConfig.SrcPath, "srcPath", "s", ".", "comic source path")
	Cmd.Flags().StringVarP(&DefaultConfig.DstRootPath, "dstRootPath", "d", ".", "comic destination root path")
	Cmd.Flags().StringVarP(&DefaultConfig.Output, "output", "o", "cmv.sh", "output shell script")
	Cmd.Flags().StringSliceVar(&DefaultConfig.SkipDirs, "skipDirs", []string{}, "skip not matched directory list")
	Cmd.Flags().BoolVarP(&DefaultConfig.IgnoreNotMatch, "ignoreNotMatch", "i", false, "ignore not matched directory")
	Cmd.Flags().BoolVar(&DefaultConfig.SkipFail, "skipFail", true, "skip failed directory")
}
