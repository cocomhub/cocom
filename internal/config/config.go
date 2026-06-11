// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"cmp"
	"fmt"
	"sync"

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

	// config-doc: recommend.limit 各维度推荐漫画数量上限
	viper.SetDefault("recommend.limit", 5)
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

func GetRecommendLimit() int {
	return viper.GetInt("recommend.limit")
}

var (
	globalCfg *Config
	once      sync.Once
)

// Init 聚合全部分散的 SetDefault。
// 应在 viper.ReadInConfig() 之后立即调用。
func Init() {
	setDefaults()
}

// Get 返回懒加载 + 缓存的 Config 实例。
func Get() *Config {
	once.Do(func() {
		cfg := &Config{}
		if err := viper.Unmarshal(cfg); err != nil {
			panic(fmt.Errorf("config unmarshal failed: %w", err))
		}
		globalCfg = cfg
	})
	return globalCfg
}

func setDefaults() {
	// === 从 cmd/server/config.go init() 移入 ===
	viper.SetDefault("server.access_log.patterns", []string{"/debug", "/api", "/v1", "/v2"})
	viper.SetDefault("server.cors.enabled", false)
	viper.SetDefault("server.cors.allow_origins", "*")
	viper.SetDefault("server.cors.allow_methods", "GET,POST,PUT,DELETE,OPTIONS")
	viper.SetDefault("server.cors.allow_headers", "*")
	viper.SetDefault("server.gzip.enabled", false)
	viper.SetDefault("server.gzip.level", 1) // gzip.BestSpeed
	viper.SetDefault("server.ratelimit.enabled", false)
	viper.SetDefault("server.ratelimit.rps", 10)
	viper.SetDefault("server.ratelimit.burst", 20)
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
	viper.SetDefault("server.scheduler.archive_status_check.max_conn", 3)
	viper.SetDefault("server.scheduler.archive_status_check.backends", []string{})
	viper.SetDefault("server.scheduler.cocoma_archiver.enabled", false)
	viper.SetDefault("server.scheduler.cocoma_archiver.cron", "* * * * *")
	viper.SetDefault("server.scheduler.cocoma_archiver.limit", 10000)
	viper.SetDefault("server.scheduler.cocoma_archiver.cid_regex", "^(\\d+)\\.cocoma$")
	viper.SetDefault("server.scheduler.cocoma_archiver.scan_dir", "")
	viper.SetDefault("server.scheduler.cocoma_archiver.archive_dir", "")
	viper.SetDefault("server.scheduler.cocoma_archiver.notmatch_dir", "")
}
