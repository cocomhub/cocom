// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"cmp"

	"github.com/spf13/viper"
)

const (
	StorageGalleryKey     = "cocom.storage.path"
	StorageArchiveKey     = "cocom.archive.path"
	StorageArchiveTempKey = "cocom.archive.temp_path"
)

func init() {
	// config-doc: cocom.storage.path 画廊数据存储路径
	viper.SetDefault(StorageGalleryKey, "/data/cocom/data/gallery")
	// config-doc: cocom.archive.path 归档文件存储路径
	viper.SetDefault(StorageArchiveKey, "/data/cocom/data/archive")
	// config-doc: cocom.archive.temp_path 归档临时文件路径
	viper.SetDefault(StorageArchiveTempKey, "/data/cocom/data/archive-temp")
	// Deprecated
	// config-doc: cocom.archive.password (已废弃) 请改用 archive.password
	viper.SetDefault("cocom.archive.password", "")
	// config-doc: cocom.archive.cmd (已废弃) 请改用 archive.cmd
	viper.SetDefault("cocom.archive.cmd", "")
	// config-doc: cocom.archive.replicate (已废弃) 请改用 archive.replicate
	viper.SetDefault("cocom.archive.replicate", false)

	// config-doc: archive.password 存档加密密码
	viper.SetDefault("archive.password", "archive@123456")
	// config-doc: archive.cmd 7z 命令路径
	viper.SetDefault("archive.cmd", "7z")
	// config-doc: archive.replicate 是否默认复制到远端存储
	viper.SetDefault("archive.replicate", false)
	// config-doc: archive.algorithm.single.concurrency 单线程算法并发数
	viper.SetDefault("archive.algorithm.single.concurrency", 4)
	// config-doc: archive.algorithm.double.concurrency 双线程算法并发数
	viper.SetDefault("archive.algorithm.double.concurrency", 4)
}

func GetSaveRoot() string {
	return viper.GetString(StorageGalleryKey)
}

func GetArchiveRoot() string {
	return viper.GetString(StorageArchiveKey)
}

func GetArchiveTempRoot() string {
	return viper.GetString(StorageArchiveTempKey)
}

func GetArchivePassword() string {
	return cmp.Or(
		viper.GetString("cocom.archive.password"),
		viper.GetString("archive.password"),
	)
}

func GetArchiveCmd() string {
	return cmp.Or(
		viper.GetString("cocom.archive.cmd"),
		viper.GetString("archive.cmd"),
	)
}

func GetArchiveReplicate() bool {
	return cmp.Or(
		viper.GetBool("cocom.archive.replicate"),
		viper.GetBool("archive.replicate"),
	)
}
