// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package gallery

import (
	"github.com/cocomhub/cocom/internal/config"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "gallery",
	Short: "图库相关工具",
	Long:  "提供对漫画图库的管理、合并与比对等功能。",
}

func init() {
	// root registration handled in cmd/root.go
}

func serverAddr() string {
	return config.Get().Client.ServerAddr
}
