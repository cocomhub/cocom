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
	"github.com/cocomhub/cocom/pkg/mongowrap"
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
	)
	rootcli.InitRootCmd(rootCmd)
	rootCmd.AddCommand(genwget.Cmd, cmv.Cmd, ar.Cmd, gallery.Cmd, install.Cmd, verify.Cmd, image.Cmd, server.Cmd)

	// 存储/归档管理器初始化下沉到真正需要它的命令（server、ar 及子命令）。
	// 根命令不再统一 OnInitialize 初始化，version/help/completion/man 不触碰存储与 MongoDB 依赖链。
	// （此前用 rootCmd.CalledAs() 判断不可靠——CalledAs() 只在被 Find() 命中的命令上置位，根命令恒为 ""。）
	ar.Cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return initArchiveManager()
	}
	server.Cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return initArchiveManager()
	}
}

func initLogging() {
	cfg, err := config.GetE()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "配置解析失败: %v\n", err)
		os.Exit(1)
	}
	if err := rootcli.ConfigLoadError(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "配置加载失败（配置存在但不可读/格式错误），终止启动：%v\n", err)
		os.Exit(1)
	}
	logging.Init(cfg.Log)
}

// initArchiveManager 初始化存储注册表与归档管理器。
// 仅在 server/ar 命令的 PersistentPreRunE 中调用，返回错误由 cobra 处理（fail-fast），不 panic。
func initArchiveManager() error {
	cfg, err := config.GetE()
	if err != nil {
		return err
	}

	// 语义校验集中在此（而非 config.Init）：config migrate 等只读配置的工具仍需在旧版/非法配置下运行。
	if err := cfg.Validate(); err != nil {
		return err
	}

	storage.Clear()
	if err := localfs.SetFromMap(map[string]string{
		config.StorageGalleryKey:     cfg.Cocom.Storage.Path,
		config.StorageArchiveKey:     cfg.Cocom.Archive.Path,
		config.StorageArchiveTempKey: cfg.Cocom.Archive.TempPath,
	}); err != nil {
		return fmt.Errorf("初始化本地存储失败：%w", err)
	}
	if err := storage.SetFromConfigs(cfg.Cocom.Storage.Backends); err != nil {
		return fmt.Errorf("初始化存储失败：%w", err)
	}

	archive.InitConcurrency(
		config.ArchiveInt(cfg.Cocom.Archive.Algorithm.Single.Concurrency, cfg.Archive.Algorithm.Single.Concurrency, "algorithm.single.concurrency"),
		config.ArchiveInt(cfg.Cocom.Archive.Algorithm.Double.Concurrency, cfg.Archive.Algorithm.Double.Concurrency, "algorithm.double.concurrency"),
	)
	// 归档错误/日志的 7z 命令行密码脱敏开关（默认 true，可显式关闭便于调试）
	archive.RedactCmd = cfg.Cocom.Archive.RedactCmd

	am := cfg.Archive.Manager
	// mongo 系索引需要先初始化 MongoDB 连接（sync.Once 幂等，与 server handler.Init 中的 Init 共存）。
	if config.IsMongoIndexType(am.Index.Type) {
		if err := mongowrap.Init(cfg.Mongo); err != nil {
			return fmt.Errorf("初始化 MongoDB 连接失败：%w", err)
		}
	}
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
		return fmt.Errorf("初始化归档管理器失败：%w", err)
	}
	return nil
}
