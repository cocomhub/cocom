// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config

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

// Init 重新注册所有 SetDefault，供 cobra.OnInitialize 调用。
func Init() {
	global.SetDefaults()
}

// Reset 清空配置缓存，使下一次 Get() 重新解析。
func Reset() { global.Reset() }

// Get 返回全局配置（懒加载 + 缓存）。
func Get() *Config { return global.Get() }

// Parse 定义在 manager.go 中，从任意 viper.Viper 解析 Config。
