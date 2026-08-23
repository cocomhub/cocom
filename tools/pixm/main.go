// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cocomhub/cocom/internal/archivecli"
	"github.com/cocomhub/cocom/internal/config"
	"github.com/cocomhub/cocom/internal/rootcli"
	"github.com/cocomhub/cocom/pkg/archive"
	"github.com/cocomhub/cocom/pkg/archive/manager"
	"github.com/cocomhub/cocom/pkg/storage"
	"github.com/cocomhub/cocom/pkg/util"
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
	Use:           "pixm",
	Short:         "Pixiv图片归档管理命令行工具",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	// 工具使用独立配置名（pixm.yaml），避免与主程序 cocom.yaml 互相污染。
	rootcli.AppName = "pixm"

	cobra.OnInitialize(
		initConfig,
		config.Init,
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

	var pid int
	rootCmd.PersistentFlags().IntVar(&pid, "pid", 0, "pixiv ID")

	archivecli.Attach(rootCmd, archivecli.Options{
		GetArchiveID: func(id int) (int, error) {
			if id > 0 && pid > 0 && id != pid/1000 {
				return 0, errors.New("归档ID与pixiv ID不匹配")
			} else if id > 0 {
				return id, nil
			} else if pid > 0 {
				return pid / 1000, nil
			}
			return 0, errors.New("缺少必要参数：--id 或 --pid")
		},
		OutputMode: outputMode,
		ReplicatePrefix: func(id int) string {
			return strings.Join(util.SplitStrRightBySize(fmt.Sprintf("%03d", id/1000), 3), "/")
		},
		ArchiveSuffix: func() string {
			return "pixma"
		},
	})
}

func initConfig() {
	// config-doc: archive.manager.meta_record_file_list 是否记录文件列表（pixm 默认启用）
	// 注意：写入 Manager 的 viper（config.G().Viper()），而非全局 viper——全局 viper 与 Manager viper 相互隔离。
	config.G().Viper().SetDefault("archive.manager.meta_record_file_list", true)
	// config-doc: archive.manager.index.type 索引类型（pixm 默认文件存储）
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

func initArchiveManager() error {
	cfg := config.Get()
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
