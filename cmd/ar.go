// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/cocomhub/cocom/internal/archivecli"
	"github.com/spf13/cobra"
)

var arOutput string

var arCmd = &cobra.Command{
	Use:   "ar",
	Short: "对单个 cid 执行归档打包、解包、查询、备份与校验",
}

func init() {
	arCmd.PersistentFlags().StringVar(&arOutput, "output", "text", "输出格式：text|json")
	archivecli.Attach(arCmd, archivecli.Options{
		OutputMode: func() string { return arOutput },
	})
	rootCmd.AddCommand(arCmd)
}
