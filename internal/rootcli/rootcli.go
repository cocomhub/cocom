// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package rootcli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/cocomhub/cocom/pkg/man"
	"github.com/cocomhub/cocom/pkg/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// AppName 用于拼接默认配置/数据/临时目录名。
// 默认取进程 basename（主程序即 cocom），工具可显式覆盖（arctl/pixm）。
var AppName = filepath.Base(os.Args[0])

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
	// 环境变量替换规则：COCOM_SERVER_LISTEN_HTTP_ADDR → server.listen.http.addr。
	// 与 internal/config.Init 保持一致，避免两处 viper 实例的 AutomaticEnv 键规则不同步。
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		_, _ = fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	} else if errors.Is(err, os.ErrNotExist) {
		// 首启不自动写盘：避免把环境变量或内部默认值落盘成隐藏点文件。
		// 引导用户通过 --config 指定或拷贝参考配置 cocom-gen.yaml。
		_, _ = fmt.Fprintln(os.Stderr, "未找到配置文件，使用内置默认配置；可用 --config 指定，或拷贝 cocom-gen.yaml")
	} else {
		_, _ = fmt.Fprintln(os.Stderr, "Read config file:", viper.ConfigFileUsed(), "failed:", err)
	}
}

// ConfigFile 返回当前生效的配置文件路径（供 config migrate 等工具使用）。
func ConfigFile() string { return cfgFile }

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
