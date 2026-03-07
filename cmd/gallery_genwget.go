// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	"github.com/cocomhub/cocom/cmd/genwget"
	"github.com/cocomhub/cocom/pkg/clog"

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

// gallery variant
var galleryGenWgetCmd = &cobra.Command{
	Use:   "genwget",
	Short: "生成图库下载脚本",
	Long:  "根据输入源生成 wget 下载脚本，用于漫画图库的批量下载。",
	Run: func(cmd *cobra.Command, args []string) {
		clog.Debugf(cmd.Context(), "gallery genwget called")
		err := genwget.NewManager().Handle(cmd.Context())
		if err != nil {
			clog.Errorf(cmd.Context(), "gallery genwget handle failed: %#v", err)
			fmt.Fprintf(os.Stderr, "gallery genwget handle failed: %#v", err)
			return
		}
		clog.Debugf(cmd.Context(), "gallery genwget handle succ")
	},
}

func init() {
	rootCmd.AddCommand(genWgetCmd)
	galleryCmd.AddCommand(galleryGenWgetCmd)

	genWgetCmd.Flags().StringVarP(&genwget.DefaultConfig.Input, "input", "i", `input.txt`, "input source")
	genWgetCmd.Flags().StringVarP(&genwget.DefaultConfig.Output, "output", "o", "genwget.sh", "output source")
	genWgetCmd.Flags().StringVarP(&genwget.DefaultConfig.DstRootPath, "dstRootPath", "d", ".", "comic destination root path")

	// reuse flags for gallery subcommand
	galleryGenWgetCmd.Flags().StringVarP(&genwget.DefaultConfig.Input, "input", "i", `input.txt`, "input source")
	galleryGenWgetCmd.Flags().StringVarP(&genwget.DefaultConfig.Output, "output", "o", "genwget.sh", "output source")
	galleryGenWgetCmd.Flags().StringVarP(&genwget.DefaultConfig.DstRootPath, "dstRootPath", "d", ".", "comic destination root path")
}
