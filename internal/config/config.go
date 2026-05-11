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
	viper.SetDefault(StorageGalleryKey, "/data/cocom/data/gallery")
	viper.SetDefault(StorageArchiveKey, "/data/cocom/data/archive")
	viper.SetDefault(StorageArchiveTempKey, "/data/cocom/data/archive-temp")
	// Deprecated
	viper.SetDefault("cocom.archive.password", "")
	viper.SetDefault("cocom.archive.cmd", "")
	viper.SetDefault("cocom.archive.replicate", false)

	viper.SetDefault("archive.password", "archive@123456")
	viper.SetDefault("archive.cmd", "7z")
	viper.SetDefault("archive.replicate", false)
	viper.SetDefault("archive.algorithm.single.concurrency", 4)
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
