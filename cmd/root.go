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
	"github.com/cocomhub/cocom/cmd/image"
	"github.com/cocomhub/cocom/cmd/install"
	"github.com/cocomhub/cocom/cmd/server"
	"github.com/cocomhub/cocom/cmd/verify"
	"github.com/cocomhub/cocom/internal/config"
	"github.com/cocomhub/cocom/internal/rootcli"
	"github.com/cocomhub/cocom/pkg/archive"
	"github.com/cocomhub/cocom/pkg/archive/manager"
	"github.com/cocomhub/cocom/pkg/logging"
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
		config.Init,
		initLogging,
		initArchiveManager,
	)
	rootcli.InitRootCmd(rootCmd)
	rootCmd.AddCommand(genwget.Cmd, cmv.Cmd, ar.Cmd, gallery.Cmd, install.Cmd, verify.Cmd, image.Cmd, server.Cmd)
}

func initLogging() {
	logging.Init(config.Get().Log)
}

func initArchiveManager() {
	storage.Clear()
	if err := localfs.SetFromMap(map[string]string{
		config.StorageGalleryKey:     config.Get().Cocom.Storage.Path,
		config.StorageArchiveKey:     config.Get().Cocom.Archive.Path,
		config.StorageArchiveTempKey: config.Get().Cocom.Archive.TempPath,
	}); err != nil {
		panic(fmt.Errorf("初始化本地存储失败：%w", err))
	}
	if err := storage.SetFromConfigs(config.Get().Cocom.Storage.Backends); err != nil {
		panic(fmt.Errorf("初始化存储失败：%w", err))
	}

	archive.InitConcurrency(
		config.Get().Cocom.Archive.Algorithm.Single.Concurrency,
		config.Get().Cocom.Archive.Algorithm.Double.Concurrency,
	)

	am := config.Get().Archive.Manager
	if err := manager.SetFromViper(manager.Config{
		Algorithm:          archive.Type(am.Algorithm),
		MetaRecordFileList: am.MetaRecordFileList,
		Replicates:         am.Replicates,
		Index: manager.IndexConfig{
			Type:            am.Index.Type,
			FileStoreName:   am.Index.FileStoreName,
			FileStorePrefix: am.Index.FileStorePrefix,
			MongoDatabase:   am.Index.MongoDatabase,
			MongoCollection: am.Index.MongoCollection,
			MongoPrefix:     am.Index.MongoPrefix,
			MongoIDField:    am.Index.MongoIDField,
			MongoNameField:  am.Index.MongoNameField,
		},
	}); err != nil {
		panic(fmt.Errorf("初始化归档管理器失败：%w", err))
	}
}
