// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package rootcli

import (
	"fmt"
	"os"

	"github.com/cocomhub/cocom/pkg/logging"
	"github.com/cocomhub/cocom/pkg/man"
	"github.com/cocomhub/cocom/pkg/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	cfgPath string
)

func InitRootCmd(rootCmd *cobra.Command) {
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

	man.AddManCmd(rootCmd)
	version.AddVersionCmd(rootCmd)
}

func InitConfig() {
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
