// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	"github.com/cocomhub/cocom/cmd/ar"
	"github.com/cocomhub/cocom/cmd/cmv"
	"github.com/cocomhub/cocom/cmd/gallery"
	"github.com/cocomhub/cocom/cmd/genwget"
	"github.com/cocomhub/cocom/cmd/install"
	"github.com/cocomhub/cocom/cmd/verify"
	"github.com/cocomhub/cocom/internal/config"
	"github.com/cocomhub/cocom/internal/rootcli"
	"github.com/cocomhub/cocom/pkg/archive/manager"
	"github.com/cocomhub/cocom/pkg/storage"
	"github.com/cocomhub/cocom/pkg/storage/localfs"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cocom",
	Short: "漫画归档、校验与图片处理 CLI",
	Long: `cocom 是集漫画归档打包、完整性校验、图片处理与 HTTP API 服务于一体的命令行工具。

常用命令：
  cocom server        启动 HTTP API 服务
  cocom ar            归档打包、解包、查询、备份与校验
  cocom gallery       图库管理（合并、比对、移动、生成下载脚本）
  cocom verify        验证漫画图片完整性
  cocom image         图片处理（缩放、裁剪、格式转换、旋转等）
  cocom version       显示版本信息`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(
		rootcli.InitConfig,
		initArchiveManager,
	)
	rootcli.InitRootCmd(rootCmd)
	rootCmd.AddCommand(genwget.Cmd, cmv.Cmd, ar.Cmd, gallery.Cmd, install.Cmd, verify.Cmd)
}

var localfsBackendKeys = []string{
	config.StorageGalleryKey,
	config.StorageArchiveKey,
	config.StorageArchiveTempKey,
}

func initArchiveManager() {
	storage.Clear()
	if err := localfs.SetFromViper(localfsBackendKeys...); err != nil {
		panic(fmt.Errorf("初始化本地存储失败：%w", err))
	}
	if err := storage.SetFromViper(); err != nil {
		panic(fmt.Errorf("初始化存储失败：%w", err))
	}
	if err := manager.SetFromViper(); err != nil {
		panic(fmt.Errorf("初始化归档管理器失败：%w", err))
	}
}
