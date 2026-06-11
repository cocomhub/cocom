// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var Cmd = &cobra.Command{
	Use:   "server",
	Short: "启动 HTTP API 服务",
	RunE: func(cmd *cobra.Command, args []string) error {
		slog.InfoContext(cmd.Context(), "server called")
		Run()
		return nil
	},
}

func init() {
	Cmd.Flags().Int32P("port", "p", 15456, "server port")
	if err := viper.BindPFlag("port", Cmd.Flags().Lookup("port")); err != nil {
		panic(err)
	}
}
