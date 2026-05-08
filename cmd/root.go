// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	"github.com/cocomhub/cocom/cmd/server/config"
	"github.com/cocomhub/cocom/pkg/archive/manager"
	"github.com/cocomhub/cocom/pkg/logging"
	"github.com/cocomhub/cocom/pkg/storage"
	"github.com/cocomhub/cocom/pkg/storage/localfs"
	"github.com/cocomhub/cocom/pkg/version"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	cfgPath string
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
		initConfig,
		initArchiveManager,
	)

	home, err := os.UserHomeDir()
	cobra.CheckErr(err)
	cfgPath = home + "/.cocom"
	cfgFile = cfgPath + "/cocom.yaml"

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// 禁用 help 标志以避免冲突
	rootCmd.PersistentFlags().BoolP("help", "", false, "help for this command")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", cfgFile, "config file")
	rootCmd.PersistentFlags().StringVar(&cfgPath, "configPath", cfgPath, "config file path")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	version.AddVersionCmd(rootCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		if cfgPath == "" {
			// Find home directory.
			home, err := os.UserHomeDir()
			cobra.CheckErr(err)
			cfgPath = home + "/.cocom"
		}

		// Search config in home directory with name ".cocom" (without extension).
		viper.AddConfigPath(cfgPath)
		viper.SetConfigType("yaml")
		viper.SetConfigName("cocom")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		_, _ = fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	} else {
		_, _ = fmt.Fprintln(os.Stderr, "Read config file:", viper.ConfigFileUsed(), "failed:", err)
	}

	logging.Init()
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
