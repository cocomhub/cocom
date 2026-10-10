// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cocomhub/cocom/internal/archivecli"
	"github.com/cocomhub/cocom/internal/config"
	"github.com/cocomhub/cocom/internal/rootcli"
	"github.com/cocomhub/cocom/pkg/archive"
	"github.com/cocomhub/cocom/pkg/archive/manager"
	"github.com/cocomhub/cocom/pkg/logging"
	"github.com/cocomhub/cocom/pkg/storage"
	"github.com/spf13/cobra"
)

var flagOutput string

func main() {
	if err := rootCmd.Execute(); err != nil {
		archivecli.EmitError(os.Stderr, os.Stdout, outputMode(), err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:           "arctl",
	Short:         "归档管理命令行工具",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	// 工具使用独立配置名（arctl.yaml），避免与主程序 cocom.yaml 互相污染。
	rootcli.AppName = "arctl"

	cobra.OnInitialize(
		initConfig,
		config.Init,
		initLogging,
	)

	rootcli.InitRootCmd(rootCmd)
	rootCmd.PersistentFlags().StringVar(&flagOutput, "output", "text", "输出格式：text|json")
	// 归档/存储初始化下沉到 PersistentPreRunE：version/help/completion/man 不再触碰存储与 MongoDB。
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		switch cmd.Name() {
		case "version", "help", "completion", "man":
			return nil
		}
		return initArchiveManager()
	}

	archivecli.Attach(rootCmd, archivecli.Options{
		OutputMode: outputMode,
		ArchiveSuffix: func() string {
			return "arctla"
		},
	})
}

func initConfig() {
	// config-doc: archive.manager.meta_record_file_list 是否记录文件列表（arctl 默认启用）
	// 注意：写入 Manager 的 viper（config.G().Viper()），而非全局 viper——全局 viper 与 Manager viper 相互隔离。
	config.G().Viper().SetDefault("archive.manager.meta_record_file_list", true)
	// config-doc: archive.manager.index.type 索引类型（arctl 默认文件存储）
	config.G().Viper().SetDefault("archive.manager.index.type", "file")
	// config-doc: cocom.storage.backends 附加存储后端列表（与 root.go initArchiveManager 同一装配源）
	config.G().Viper().SetDefault("cocom.storage.backends", []storage.Config{
		{
			Name: "archive-manager-index",
			Type: "localfs",
			MetaData: map[string]any{
				"root": filepath.Join(func() string {
					d, err := rootcli.DataDir()
					if err != nil {
						panic(err)
					}
					return d
				}(), "storage", "archive-manager-index"),
			},
		},
	})

	rootcli.InitConfig()
}

// initLogging 按已加载的配置初始化日志。
//
// 必须注册在 config.Init 之后：变量配置的默认值与配置文件合并都在那一步完成，
// 在此之前读到的 log.* 是未合并的值。
//
// 回归背景：rootcli.InitConfig 末尾的 logging.Init 被移除后，子工具没有任何地方
// 初始化日志（全仓 logging.Init 只命中 cmd/root.go），于是 log.enableFile / log.level /
// log.encoding 对 arctl/pixm/pixcover 完全无效，一直输出到默认 TextHandler。
func initLogging() {
	if err := logging.Init(config.Get().Log); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "日志初始化失败: %v\n", err)
	}
}

func initArchiveManager() error {
	cfg, err := config.GetE()
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	storage.Clear()
	backends := cfg.Cocom.Storage.Backends
	if err := storage.SetFromConfigs(backends); err != nil {
		return fmt.Errorf("初始化存储失败：%w", err)
	}
	am := cfg.Archive.Manager
	cfg2 := manager.Config{
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
	}
	if err := manager.SetFromViper(cfg2); err != nil {
		return fmt.Errorf("初始化归档管理器失败：%w", err)
	}
	return nil
}

func outputMode() string {
	if strings.TrimSpace(flagOutput) == "" {
		return "text"
	}
	return flagOutput
}
