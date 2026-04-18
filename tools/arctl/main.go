// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/cocomhub/cocom/internal/archivecli"
	"github.com/cocomhub/cocom/pkg/archive/manager"
	"github.com/cocomhub/cocom/pkg/storage"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	flagConfig  string
	flagOutput  string
	flagVerbose bool
)

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
	cmd := &cobra.Command{
		Use:   "arctl",
		Short: "归档管理命令行工具",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := initConfig(); err != nil {
				return fmt.Errorf("初始化配置失败: %w", err)
			}
			if err := initArchiveManager(); err != nil {
				return fmt.Errorf("初始化归档管理器失败: %w", err)
			}
			return nil
		},
	}
	cmd.PersistentFlags().StringVar(&flagConfig, "config", "", "配置文件路径")
	cmd.PersistentFlags().StringVar(&flagOutput, "output", "text", "输出格式：text|json")
	cmd.PersistentFlags().BoolVar(&flagVerbose, "verbose", false, "启用详细日志")
	_ = viper.BindPFlag("arctl.output", cmd.PersistentFlags().Lookup("output"))
	_ = viper.BindPFlag("arctl.verbose", cmd.PersistentFlags().Lookup("verbose"))
	archivecli.Attach(cmd, archivecli.Options{
		OutputMode: outputMode,
	})
	return cmd
}

func initConfig() error {
	c := manager.DefaultConfig()
	viper.SetDefault("arctl.archive.manager.algorithm", string(c.Algorithm))
	viper.SetDefault("arctl.archive.manager.replicates", c.Replicates)
	viper.SetDefault("arctl.archive.manager.index.type", "file")
	viper.SetDefault("arctl.archive.manager.index.fileStoreName", c.Index.FileStoreName)
	viper.SetDefault("arctl.archive.manager.index.fileStorePrefix", c.Index.FileStorePrefix)
	viper.SetDefault("arctl.archive.manager.index.mongoDatabase", c.Index.MongoDatabase)
	viper.SetDefault("arctl.archive.manager.index.mongoCollection", c.Index.MongoCollection)

	if strings.TrimSpace(flagConfig) != "" {
		viper.SetConfigFile(flagConfig)
	} else {
		viper.SetConfigType("yaml")
	}
	viper.AutomaticEnv()
	return viper.ReadInConfig()
}

func initArchiveManager() error {
	if err := storage.SetFromViper(); err != nil {
		return err
	}
	if err := manager.SetFromViper("arctl.archive.manager"); err != nil {
		return err
	}
	return nil
}

func outputMode() string {
	if strings.TrimSpace(flagOutput) != "" {
		return flagOutput
	}
	return viper.GetString("arctl.output")
}
