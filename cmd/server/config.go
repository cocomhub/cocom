// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"compress/gzip"

	"github.com/spf13/viper"
)

func init() {
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
