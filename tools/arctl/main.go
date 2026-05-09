// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/cocomhub/cocom/internal/archivecli"
	"github.com/cocomhub/cocom/internal/rootcli"
	"github.com/cocomhub/cocom/pkg/archive/manager"
	"github.com/cocomhub/cocom/pkg/storage"
	"github.com/cocomhub/cocom/pkg/util"
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
		initArchiveManager,
	)

	rootcli.InitRootCmd(rootCmd)
	rootCmd.PersistentFlags().StringVar(&flagOutput, "output", "text", "输出格式：text|json")
	_ = viper.BindPFlag("arctl.output", rootCmd.PersistentFlags().Lookup("output"))

	archivecli.Attach(rootCmd, archivecli.Options{
		OutputMode: outputMode,
		RootDir: func() string {
			return viper.GetString("arctl.archive.root_dir")
		},
		ArchiveSuffix: func() string {
			return util.FirstNonEmpty(viper.GetString("arctl.archive.archive_suffix"), "arctla")
		},
	})
}

func initConfig() {
	c := manager.DefaultConfig()
	viper.SetDefault("arctl.archive.root_dir", "arctl")
	viper.SetDefault("arctl.archive.archive_suffix", "arctla")
	viper.SetDefault("arctl.archive.manager.algorithm", string(c.Algorithm))
	viper.SetDefault("arctl.archive.manager.meta_record_file_list", true)
	viper.SetDefault("arctl.archive.manager.replicates", c.Replicates)
	viper.SetDefault("arctl.archive.manager.index.type", "file")
	viper.SetDefault("arctl.archive.manager.index.file_store_name", c.Index.FileStoreName)
	viper.SetDefault("arctl.archive.manager.index.file_store_prefix", c.Index.FileStorePrefix)
	viper.SetDefault("arctl.archive.manager.index.mongo_database", c.Index.MongoDatabase)
	viper.SetDefault("arctl.archive.manager.index.mongo_collection", c.Index.MongoCollection)
	viper.SetDefault("arctl.archive.manager.index.mongo_prefix", c.Index.MongoPrefix)
	viper.SetDefault("arctl.archive.manager.index.mongo_id_field", c.Index.MongoIDField)
	viper.SetDefault("arctl.archive.manager.index.mongo_name_field", c.Index.MongoNameField)

	rootcli.InitConfig()
}

func initArchiveManager() {
	storage.Clear()
	if err := storage.SetFromViper(); err != nil {
		panic(fmt.Errorf("初始化存储失败：%w", err))
	}
	if err := manager.SetFromViper("arctl.archive.manager"); err != nil {
		panic(fmt.Errorf("初始化归档管理器失败：%w", err))
	}
}

func outputMode() string {
	if strings.TrimSpace(flagOutput) != "" {
		return flagOutput
	}
	return viper.GetString("arctl.output")
}
