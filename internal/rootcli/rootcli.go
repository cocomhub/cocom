// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package rootcli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/cocomhub/cocom/pkg/man"
	"github.com/cocomhub/cocom/pkg/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var AppName = ""

var (
	cfgFile string
	dataDir string
	tempDir string
)

func InitRootCmd(rootCmd *cobra.Command) {
	var err error
	cfgFile, err = xdg.ConfigFile(fmt.Sprintf("cocom/%s.yaml", AppName))
	cobra.CheckErr(err)

	dataDirStr, err := DataDir()
	cobra.CheckErr(err)
	tempDirStr, err := TempDir()
	cobra.CheckErr(err)

	// 禁用 help 标志以避免冲突
	rootCmd.PersistentFlags().BoolP("help", "", false, "help for this command")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", cfgFile, "config file")
	rootCmd.PersistentFlags().StringVar(&dataDir, "data-dir", dataDirStr, "data directory")
	rootCmd.PersistentFlags().StringVar(&tempDir, "temp-dir", tempDirStr, "temp directory")

	man.AddManCmd(rootCmd)
	version.AddVersionCmd(rootCmd)
}

func InitConfig() {
	viper.SetConfigFile(cfgFile)
	viper.SetEnvPrefix("COCOM")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		_, _ = fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	} else if errors.Is(err, os.ErrNotExist) {
		err = viper.WriteConfigAs(cfgFile)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "初始化配置文件失败：%v\n", err)
			os.Exit(1)
		}
		_, _ = fmt.Fprintln(os.Stderr, "Created config file:", viper.ConfigFileUsed())
	} else {
		_, _ = fmt.Fprintln(os.Stderr, "Read config file:", viper.ConfigFileUsed(), "failed:", err)
	}
}

func DataDir() (string, error) {
	if dataDir != "" {
		return dataDir, nil
	}
	file, err := xdg.DataFile(fmt.Sprintf("cocom/%s/init", AppName))
	if err != nil {
		return "", fmt.Errorf("获取数据目录失败：%w", err)
	}
	return filepath.Dir(file), nil
}

func TempDir() (string, error) {
	if tempDir != "" {
		return tempDir, nil
	}
	file, err := xdg.CacheFile(fmt.Sprintf("cocom/%s/init", AppName))
	if err != nil {
		return "", fmt.Errorf("获取数据临时目录失败：%w", err)
	}
	return filepath.Dir(file), nil
}
