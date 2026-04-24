// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"compress/gzip"

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
	viper.SetDefault("cocom.archive.password", "")
	viper.SetDefault("cocom.archive.cmd", "7z")
	viper.SetDefault("cocom.archive.replicate", false)
	viper.SetDefault("archive.algorithm.single.concurrency", 4)
	viper.SetDefault("archive.algorithm.double.concurrency", 4)
	viper.SetDefault("server.access_log.patterns", []string{"/debug", "/api", "/v1", "/v2"})
	// server 中间件配置默认值（默认关闭）
	viper.SetDefault("server.cors.enabled", false)
	viper.SetDefault("server.cors.allow_origins", "*")
	viper.SetDefault("server.cors.allow_methods", "GET,POST,PUT,DELETE,OPTIONS")
	viper.SetDefault("server.cors.allow_headers", "*")
	viper.SetDefault("server.gzip.enabled", false)
	viper.SetDefault("server.gzip.level", gzip.BestSpeed)
	viper.SetDefault("server.ratelimit.enabled", false)
	viper.SetDefault("server.ratelimit.rps", 10)
	viper.SetDefault("server.ratelimit.burst", 20)
	// 调度器配置
	viper.SetDefault("server.scheduler.enabled", false)
	viper.SetDefault("server.scheduler.timezone", "Local")
	viper.SetDefault("server.scheduler.probe_comic.enabled", false)
	viper.SetDefault("server.scheduler.probe_comic.name", "ProbeComic")
	viper.SetDefault("server.scheduler.probe_comic.cron", "0 */10 * * * *")
	viper.SetDefault("server.scheduler.probe_comic.tags", []string{"probe", "comic"})
	viper.SetDefault("server.scheduler.archive_status_check.enabled", false)
	viper.SetDefault("server.scheduler.archive_status_check.name", "ArchiveStatusChecker")
	viper.SetDefault("server.scheduler.archive_status_check.cron", "0 */30 * * * *")
	viper.SetDefault("server.scheduler.archive_status_check.tags", []string{"archive", "check"})
	viper.SetDefault("server.scheduler.archive_status_check.limit", 100)
	viper.SetDefault("server.scheduler.archive_status_check.targets", []map[string]any{})
	viper.SetDefault("server.scheduler.cocoma_archiver.enabled", false)
	viper.SetDefault("server.scheduler.cocoma_archiver.cron", "* * * * *")
	viper.SetDefault("server.scheduler.cocoma_archiver.limit", 10000)
	viper.SetDefault("server.scheduler.cocoma_archiver.cid_regex", "^(\\d+)\\.cocoma$")
	viper.SetDefault("server.scheduler.cocoma_archiver.scan_dir", "")
	viper.SetDefault("server.scheduler.cocoma_archiver.archive_dir", "")
	viper.SetDefault("server.scheduler.cocoma_archiver.notmatch_dir", "")
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
	return viper.GetString("cocom.archive.password")
}

func GetArchiveCmd() string {
	return viper.GetString("cocom.archive.cmd")
}

func GetArchiveReplicate() bool {
	return viper.GetBool("cocom.archive.replicate")
}
