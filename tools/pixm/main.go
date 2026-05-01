// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/cocomhub/cocom/internal/archivecli"
	"github.com/cocomhub/cocom/pkg/archive/manager"
	"github.com/cocomhub/cocom/pkg/storage"
	"github.com/cocomhub/cocom/pkg/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	flagConfig  string
	flagOutput  string
	flagVerbose bool
)

func init() {
	cobra.OnInitialize(
		initConfig,
		initArchiveManager,
	)
}

func main() {
	root := newRootCmd()
	root.SilenceUsage = true
	root.SilenceErrors = true
	if err := root.Execute(); err != nil {
		archivecli.EmitError(os.Stderr, os.Stdout, outputMode(), err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var pid int
	cmd := &cobra.Command{
		Use:   "pixm",
		Short: "Pixiv图片归档管理命令行工具",
	}
	cmd.PersistentFlags().IntVar(&pid, "pid", 0, "pixiv ID")
	cmd.PersistentFlags().StringVar(&flagConfig, "config", "", "配置文件路径")
	cmd.PersistentFlags().StringVar(&flagOutput, "output", "text", "输出格式：text|json")
	cmd.PersistentFlags().BoolVar(&flagVerbose, "verbose", false, "启用详细日志")
	_ = viper.BindPFlag("pixm.output", cmd.PersistentFlags().Lookup("output"))
	_ = viper.BindPFlag("pixm.verbose", cmd.PersistentFlags().Lookup("verbose"))
	archivecli.Attach(cmd, archivecli.Options{
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
		RootDir: func() string {
			return viper.GetString("pixm.archive.root_dir")
		},
		ArchiveSuffix: func() string {
			return util.FirstNonEmpty(viper.GetString("pixm.archive.archive_suffix"), "pixma")
		},
	})
	return cmd
}

func initConfig() {
	c := manager.DefaultConfig()
	viper.SetDefault("pixm.archive.root_dir", "pixm")
	viper.SetDefault("pixm.archive.archive_suffix", "pixma")
	viper.SetDefault("pixm.archive.manager.algorithm", string(c.Algorithm))
	viper.SetDefault("pixm.archive.manager.meta_record_file_list", true)
	viper.SetDefault("pixm.archive.manager.replicates", c.Replicates)
	viper.SetDefault("pixm.archive.manager.index.type", "file")
	viper.SetDefault("pixm.archive.manager.index.file_store_name", c.Index.FileStoreName)
	viper.SetDefault("pixm.archive.manager.index.file_store_prefix", c.Index.FileStorePrefix)
	viper.SetDefault("pixm.archive.manager.index.mongo_database", c.Index.MongoDatabase)
	viper.SetDefault("pixm.archive.manager.index.mongo_collection", c.Index.MongoCollection)
	viper.SetDefault("pixm.archive.manager.index.mongo_prefix", c.Index.MongoPrefix)
	viper.SetDefault("pixm.archive.manager.index.mongo_id_field", c.Index.MongoIDField)
	viper.SetDefault("pixm.archive.manager.index.mongo_name_field", c.Index.MongoNameField)

	if strings.TrimSpace(flagConfig) != "" {
		viper.SetConfigFile(flagConfig)
	} else {
		viper.SetConfigType("yaml")
	}
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("初始化配置失败：%w", err))
	}
}

func initArchiveManager() {
	storage.Clear()
	if err := storage.SetFromViper(); err != nil {
		panic(fmt.Errorf("初始化存储失败：%w", err))
	}
	if err := manager.SetFromViper("pixm.archive.manager"); err != nil {
		panic(fmt.Errorf("初始化归档管理器失败：%w", err))
	}
}

func outputMode() string {
	if strings.TrimSpace(flagOutput) != "" {
		return flagOutput
	}
	return viper.GetString("pixm.output")
}
