// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"fmt"
	"log/slog"
	"net"
	"strconv"

	"github.com/cocomhub/cocom/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var Cmd = &cobra.Command{
	Use:   "server",
	Short: "启动 HTTP API 服务",
	RunE: func(cmd *cobra.Command, args []string) error {
		slog.InfoContext(cmd.Context(), "server called")

		// 如果 -p / --port 被显式设置，解析 server.listen.http.addr 并替换端口号
		if cmd.Flags().Changed("port") {
			port, err := strconv.Atoi(cmd.Flags().Lookup("port").Value.String())
			if err != nil {
				return fmt.Errorf("invalid port value: %w", err)
			}
			addr := viper.GetString("server.listen.http.addr")
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				// addr 格式异常时，直接用 host 整体拼
				host = addr
			}
			viper.Set("server.listen.http.addr", fmt.Sprintf("%s:%d", host, port))
		}

		// CLI flag 或环境变量修改后需要 Reset 再 Init 确保 config.Get() 拿到新值
		config.Reset()
		config.Init()

		Run()
		return nil
	},
}

func init() {
	Cmd.Flags().Int32P("port", "p", 0, "server port (overrides port in server.listen.http.addr)")
	// 不再 BindPFlag — 在 RunE 中手动替换端口
}
