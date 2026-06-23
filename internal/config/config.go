// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

const (
	StorageGalleryKey     = "cocom.storage.path"
	StorageArchiveKey     = "cocom.archive.path"
	StorageArchiveTempKey = "cocom.archive.temp_path"
)

// global 是全局 Manager 实例。
// 生产代码通过 G().*() 访问；测试代码创建独立 Manager 实例隔离。
var global *Manager

func init() {
	global = New()
}

// G 返回全局 Manager 实例。
func G() *Manager { return global }

// Init 重新注册所有 SetDefault，并从全局 viper 同步配置文件和环境变量。
// 供 cobra.OnInitialize 调用（在 rootcli.InitConfig 之后执行）。
func Init() {
	global.SetDefaults()

	// 将全局 viper（rootcli.InitConfig 已加载）的配置源同步到 Manager 的 viper。
	// 全局 viper 已通过 SetConfigFile + ReadInConfig 加载了 YAML，并配置了
	// SetEnvPrefix("COCOM") + AutomaticEnv()，但 Manager 的 viper 是独立实例
	// 对这些一无所知——不同步的话 config.Get() 只返回硬编码默认值。
	if cfgFile := viper.ConfigFileUsed(); cfgFile != "" {
		global.v.SetConfigFile(cfgFile)
		// MergeInConfig 将文件值合并到已有默认值之上（文件值优先级高于 SetDefault）
		if err := global.v.MergeInConfig(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "config: merge config file %s: %v\n", cfgFile, err)
		}
	}
	global.v.SetEnvPrefix("COCOM")
	global.v.AutomaticEnv()

	// Reset 缓存，下次 Get() 重新 Unmarshal（此时默认值 + YAML + env 均已到位）
	global.Reset()
}

// Reset 清空配置缓存，使下一次 Get() 重新解析。
func Reset() { global.Reset() }

// Get 返回全局配置（懒加载 + 缓存）。
func Get() *Config { return global.Get() }

// Parse 定义在 manager.go 中，从任意 viper.Viper 解析 Config。
