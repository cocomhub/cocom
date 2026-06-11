// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

// config-doc 注释保留以供 config-doc-gen 工具读取。
// SetDefault 已迁移至 internal/config/config.go 的 setDefaults()。
//
// config-doc: server.access_log.patterns 记录访问日志的 URL 模式列表
// Default: [/debug /api /v1 /v2]
// config-doc: server.cors.enabled 是否启用 CORS
// Default: false
// config-doc: server.cors.allow_origins 允许的源
// Default: *
// config-doc: server.cors.allow_methods 允许的 HTTP 方法
// Default: GET,POST,PUT,DELETE,OPTIONS
// config-doc: server.cors.allow_headers 允许的请求头
// Default: *
// config-doc: server.gzip.enabled 是否启用 Gzip 压缩
// Default: false
// config-doc: server.gzip.level Gzip 压缩级别
// Default: 1 (gzip.BestSpeed)
// config-doc: server.ratelimit.enabled 是否启用限流
// Default: false
// config-doc: server.ratelimit.rps 每秒请求数限制
// Default: 10
// config-doc: server.ratelimit.burst 突发请求数
// Default: 20
// 调度器配置
// config-doc: server.scheduler.enabled 是否启用调度器
// Default: false
// config-doc: server.scheduler.timezone 时区
// Default: Local
// config-doc: server.scheduler.probe_comic.enabled 是否启用漫画探测调度
// Default: false
// config-doc: server.scheduler.probe_comic.name 漫画探测任务名称
// Default: ProbeComic
// config-doc: server.scheduler.probe_comic.cron 漫画探测 Cron 表达式
// Default: 0 */10 * * * *
// config-doc: server.scheduler.probe_comic.tags 漫画探测标签列表
// Default: [probe comic]
// config-doc: server.scheduler.archive_status_check.enabled 是否启用存档状态检查
// Default: false
// config-doc: server.scheduler.archive_status_check.name 存档状态检查任务名称
// Default: ArchiveStatusChecker
// config-doc: server.scheduler.archive_status_check.cron 存档状态检查 Cron 表达式
// Default: 0 */30 * * * *
// config-doc: server.scheduler.archive_status_check.tags 存档状态检查标签列表
// Default: [archive check]
// config-doc: server.scheduler.archive_status_check.limit 每次检查数量上限
// Default: 100
// config-doc: server.scheduler.archive_status_check.max_conn 最大并发连接数
// Default: 3
// config-doc: server.scheduler.archive_status_check.backends 要检查的后端列表
// Default: []
// config-doc: server.scheduler.cocoma_archiver.enabled 是否启用 Cocoma 归档调度
// Default: false
// config-doc: server.scheduler.cocoma_archiver.cron Cocoma 归档 Cron 表达式
// Default: * * * * *
// config-doc: server.scheduler.cocoma_archiver.limit 每次处理上限
// Default: 10000
// config-doc: server.scheduler.cocoma_archiver.cid_regex CID 匹配正则
// Default: ^(\d+)\.cocoma$
// config-doc: server.scheduler.cocoma_archiver.scan_dir 扫描目录
// Default: ""
// config-doc: server.scheduler.cocoma_archiver.archive_dir 归档输出目录
// Default: ""
// config-doc: server.scheduler.cocoma_archiver.notmatch_dir 不匹配文件的移动目录
// Default: ""
