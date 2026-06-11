// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package genwget

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "genwget",
	Short: "生成 wget 下载脚本",
	RunE: func(cmd *cobra.Command, args []string) error {
		slog.DebugContext(cmd.Context(), "gen wget called")
		if err := NewManager().Handle(cmd.Context()); err != nil {
			slog.ErrorContext(cmd.Context(), "gen comic wget handle failed", slog.String("err", err.Error()))
			fmt.Fprintf(os.Stderr, "gen comic wget handle failed: %#v", err)
			return err
		}
		slog.DebugContext(cmd.Context(), "gen comic wget handle succ")
		return nil
	},
}

func init() {
	Cmd.Flags().StringVarP(&DefaultConfig.Input, "input", "i", "input.txt", "input source")
	Cmd.Flags().StringVarP(&DefaultConfig.Output, "output", "o", "genwget.sh", "output source")
	Cmd.Flags().StringVarP(&DefaultConfig.DstRootPath, "dstRootPath", "d", ".", "comic destination root path")
}
