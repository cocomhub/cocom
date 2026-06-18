// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package download

// Config 下载模块配置。
// mapstructure 标签对齐 viper 键路径（如 download.downloadDir → Config.DownloadDir）。
type Config struct {
	MaxRunning  int    `mapstructure:"maxRunning" json:"maxRunning"`
	DownloadDir string `mapstructure:"downloadDir" json:"downloadDir"`
}
