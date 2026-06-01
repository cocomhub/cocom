// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"log/slog"
	"runtime/debug"

	"github.com/cocomhub/cocom/cmd/server"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "启动 HTTP API 服务",
	Long: `启动 cocom 的 Gin HTTP API 服务，提供漫画查询、管理、校验、归档等接口。

启动示例：
  cocom server                     使用默认配置启动
  cocom server --port 8080         指定端口
  cocom server --config ./conf.yaml 指定配置文件`,
	Run: func(cmd *cobra.Command, args []string) {
		defer func() {
			if err := recover(); err != nil {
				slog.ErrorContext(cmd.Context(), "server panic", slog.Any("err", err), slog.String("stack", string(debug.Stack())))
			}
		}()
		slog.InfoContext(cmd.Context(), "server called")
		server.Run()
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.Flags().Int32P("port", "p", 15456, "server port")
	err := viper.BindPFlag("port", serverCmd.Flags().Lookup("port"))
	if err != nil {
		panic(any(err))
	}
}
