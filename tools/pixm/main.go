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
	Use:           "pixm",
	Short:         "Pixiv图片归档管理命令行工具",
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
	_ = viper.BindPFlag("pixm.output", rootCmd.PersistentFlags().Lookup("output"))

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
	viper.SetDefault("archive.manager.meta_record_file_list", true)
	viper.SetDefault("archive.manager.index.type", "file")
	viper.SetDefault("storage.backends", []storage.Config{
		{
			Name: "archive-manager-index",
			Type: "localfs",
			MetaData: map[string]any{
				"root": filepath.Join(rootcli.DataDir(), "storage", "archive-manager-index"),
			},
		},
	})
	rootcli.InitConfig()
}

func initArchiveManager() {
	storage.Clear()
	if err := storage.SetFromViper(); err != nil {
		panic(fmt.Errorf("初始化存储失败：%w", err))
	}
	if err := manager.SetFromViper(); err != nil {
		panic(fmt.Errorf("初始化归档管理器失败：%w", err))
	}
}

func outputMode() string {
	if strings.TrimSpace(flagOutput) != "" {
		return flagOutput
	}
	return viper.GetString("pixm.output")
}
