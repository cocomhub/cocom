// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"

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
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
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
