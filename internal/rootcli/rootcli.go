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
		// 铁律 1：配置存在但不可读/格式错误 → 显式报错退出，不回退默认静默启动。
		// viper 对「文件存在但解析失败」与「其他 I/O 错误」都返回非 ErrNotExist 错误。
		// 注意：cobra.OnInitialize 钩子无法安全 panic/os.Exit（在 Run 之前调用），
		// 因此这里记录哨兵错误，由 config.Init 的失败路径承担最终 fail-fast。
		// （config.Init 依赖 viper.ConfigFileUsed()，若此处未合并，config.Init 拿到
		// 的文件值也就不可用——两处共同保证配置错误不被静默吞掉。）
		_, _ = fmt.Fprintf(os.Stderr, "读取配置文件失败，终止启动（配置存在但不可读/格式错误）：%v\n", err)
		initConfigErr = err
		return
	}
}

// initConfigErr 记录配置文件加载失败（rootcli.InitConfig 为 cobra.OnInitialize 钩子
// 无法直接 fail-fast，由后续 initLogging 检查并 os.Exit(1)，铁律 1）。
var initConfigErr error

// ConfigLoadError 返回配置文件加载错误；无错误时返回 nil。
func ConfigLoadError() error { return initConfigErr }

// ConfigFile 返回当前生效的配置文件路径（供 config migrate 等工具使用）。
func ConfigFile() string { return cfgFile }

// SetConfigFileForTest 仅在测试中注入配置文件路径（生产路径由 --config flag 驱动）。
func SetConfigFileForTest(path string) { cfgFile = path }

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
