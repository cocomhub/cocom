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
	"github.com/cocomhub/cocom/pkg/storage"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	cobra.OnInitialize(
		initConfig,
		config.Init,
		initArchiveManager,
	)

	rootcli.InitRootCmd(rootCmd)
	rootCmd.PersistentFlags().StringVar(&flagOutput, "output", "text", "输出格式：text|json")
	_ = viper.BindPFlag("arctl.output", rootCmd.PersistentFlags().Lookup("output"))

	archivecli.Attach(rootCmd, archivecli.Options{
		OutputMode: outputMode,
		ArchiveSuffix: func() string {
			return "arctla"
		},
	})
}

func initConfig() {
	// config-doc: archive.manager.meta_record_file_list 是否记录文件列表（arctl 默认启用）
	viper.SetDefault("archive.manager.meta_record_file_list", true)
	// config-doc: archive.manager.index.type 索引类型（arctl 默认文件存储）
	viper.SetDefault("archive.manager.index.type", "file")
	// config-doc: storage.backends 附加存储后端列表
	viper.SetDefault("storage.backends", []storage.Config{
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

func initArchiveManager() {
	storage.Clear()
	backends := config.Get().Cocom.Storage.Backends
	if err := storage.SetFromConfigs(backends); err != nil {
		panic(fmt.Errorf("初始化存储失败：%w", err))
	}
	am := config.Get().Archive.Manager
	cfg := manager.Config{
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
	if err := manager.SetFromViper(cfg); err != nil {
		panic(fmt.Errorf("初始化归档管理器失败：%w", err))
	}
}

func outputMode() string {
	if strings.TrimSpace(flagOutput) == "" {
		return "text"
	}
	return flagOutput
}
