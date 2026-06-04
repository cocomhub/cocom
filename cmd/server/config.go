// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"compress/gzip"

	"github.com/spf13/viper"
)

func init() {
	// config-doc: server.access_log.patterns 记录访问日志的 URL 模式列表
	viper.SetDefault("server.access_log.patterns", []string{"/debug", "/api", "/v1", "/v2"})
	// config-doc: server.cors.enabled 是否启用 CORS
	viper.SetDefault("server.cors.enabled", false)
	// config-doc: server.cors.allow_origins 允许的源
	viper.SetDefault("server.cors.allow_origins", "*")
	// config-doc: server.cors.allow_methods 允许的 HTTP 方法
	viper.SetDefault("server.cors.allow_methods", "GET,POST,PUT,DELETE,OPTIONS")
	// config-doc: server.cors.allow_headers 允许的请求头
	viper.SetDefault("server.cors.allow_headers", "*")
	// config-doc: server.gzip.enabled 是否启用 Gzip 压缩
	viper.SetDefault("server.gzip.enabled", false)
	// config-doc: server.gzip.level Gzip 压缩级别
	viper.SetDefault("server.gzip.level", gzip.BestSpeed)
	// config-doc: server.ratelimit.enabled 是否启用限流
	viper.SetDefault("server.ratelimit.enabled", false)
	// config-doc: server.ratelimit.rps 每秒请求数限制
	viper.SetDefault("server.ratelimit.rps", 10)
	// config-doc: server.ratelimit.burst 突发请求数
	viper.SetDefault("server.ratelimit.burst", 20)
	// 调度器配置
	// config-doc: server.scheduler.enabled 是否启用调度器
	viper.SetDefault("server.scheduler.enabled", false)
	// config-doc: server.scheduler.timezone 时区
	viper.SetDefault("server.scheduler.timezone", "Local")
	// config-doc: server.scheduler.probe_comic.enabled 是否启用漫画探测调度
	viper.SetDefault("server.scheduler.probe_comic.enabled", false)
	// config-doc: server.scheduler.probe_comic.name 漫画探测任务名称
	viper.SetDefault("server.scheduler.probe_comic.name", "ProbeComic")
	// config-doc: server.scheduler.probe_comic.cron 漫画探测 Cron 表达式
	viper.SetDefault("server.scheduler.probe_comic.cron", "0 */10 * * * *")
	// config-doc: server.scheduler.probe_comic.tags 漫画探测标签列表
	viper.SetDefault("server.scheduler.probe_comic.tags", []string{"probe", "comic"})
	// config-doc: server.scheduler.archive_status_check.enabled 是否启用存档状态检查
	viper.SetDefault("server.scheduler.archive_status_check.enabled", false)
	// config-doc: server.scheduler.archive_status_check.name 存档状态检查任务名称
	viper.SetDefault("server.scheduler.archive_status_check.name", "ArchiveStatusChecker")
	// config-doc: server.scheduler.archive_status_check.cron 存档状态检查 Cron 表达式
	viper.SetDefault("server.scheduler.archive_status_check.cron", "0 */30 * * * *")
	// config-doc: server.scheduler.archive_status_check.tags 存档状态检查标签列表
	viper.SetDefault("server.scheduler.archive_status_check.tags", []string{"archive", "check"})
	// config-doc: server.scheduler.archive_status_check.limit 每次检查数量上限
	viper.SetDefault("server.scheduler.archive_status_check.limit", 100)
	// config-doc: server.scheduler.archive_status_check.max_conn 最大并发连接数
	viper.SetDefault("server.scheduler.archive_status_check.max_conn", 3)
	// config-doc: server.scheduler.archive_status_check.backends 要检查的后端列表
	viper.SetDefault("server.scheduler.archive_status_check.backends", []string{})
	// config-doc: server.scheduler.cocoma_archiver.enabled 是否启用 Cocoma 归档调度
	viper.SetDefault("server.scheduler.cocoma_archiver.enabled", false)
	// config-doc: server.scheduler.cocoma_archiver.cron Cocoma 归档 Cron 表达式
	viper.SetDefault("server.scheduler.cocoma_archiver.cron", "* * * * *")
	// config-doc: server.scheduler.cocoma_archiver.limit 每次处理上限
	viper.SetDefault("server.scheduler.cocoma_archiver.limit", 10000)
	// config-doc: server.scheduler.cocoma_archiver.cid_regex CID 匹配正则
	viper.SetDefault("server.scheduler.cocoma_archiver.cid_regex", "^(\\d+)\\.cocoma$")
	// config-doc: server.scheduler.cocoma_archiver.scan_dir 扫描目录
	viper.SetDefault("server.scheduler.cocoma_archiver.scan_dir", "")
	// config-doc: server.scheduler.cocoma_archiver.archive_dir 归档输出目录
	viper.SetDefault("server.scheduler.cocoma_archiver.archive_dir", "")
	// config-doc: server.scheduler.cocoma_archiver.notmatch_dir 不匹配文件的移动目录
	viper.SetDefault("server.scheduler.cocoma_archiver.notmatch_dir", "")
}
