// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"cmp"
	"fmt"
	"sync"
	"time"

	"github.com/cocomhub/cocom/pkg/archive"
	"github.com/spf13/viper"
)

const (
	StorageGalleryKey     = "cocom.storage.path"
	StorageArchiveKey     = "cocom.archive.path"
	StorageArchiveTempKey = "cocom.archive.temp_path"
)

func init() {
	// init() 确保在包加载时就设置了 defaults（被 pkg/storage/localfs 等依赖）。
	// 同步给 setDefaults()，使 viper.Reset() + Init() 也能恢复全部默认值。
	setDefaults()
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
	mu        sync.Mutex
)

// Init 聚合全部分散的 SetDefault。
// 应在 viper.ReadInConfig() 之后立即调用。
func Init() {
	setDefaults()
}

// Reset 清空缓存，使下一次 Get() 重新 Unmarshal 当前 Viper 状态。
// 在运行时修改 Viper 配置后（如 CLI flag 处理）必须调用 Reset() 再 Get()。
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	globalCfg = nil
}

// Get 返回懒加载 + 缓存的 Config 实例。
// 调用 Reset() 可使下一次 Get() 重新 Unmarshal。
func Get() *Config {
	mu.Lock()
	defer mu.Unlock()
	if globalCfg != nil {
		return globalCfg
	}
	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		panic(fmt.Errorf("config unmarshal failed: %w", err))
	}
	globalCfg = cfg
	return globalCfg
}

func setDefaults() {
	// 核心存储路径（必须在最前面，被 pkg/storage/localfs 引用）
	// config-doc: cocom.storage.path 画廊数据存储路径
	viper.SetDefault(StorageGalleryKey, "/data/cocom/data/gallery")
	// config-doc: cocom.archive.path 归档文件存储路径
	viper.SetDefault(StorageArchiveKey, "/data/cocom/data/archive")
	// config-doc: cocom.archive.temp_path 归档临时文件路径
	viper.SetDefault(StorageArchiveTempKey, "/data/cocom/data/archive-temp")

	// archive.* 旧版兼容键
	// config-doc: archive.password 存档加密密码
	viper.SetDefault("archive.password", "archive@123456")
	// config-doc: archive.cmd 7z 命令路径
	viper.SetDefault("archive.cmd", "7z")
	// config-doc: archive.replicate 是否默认复制到远端存储
	viper.SetDefault("archive.replicate", false)

	// archive.algorithm.*
	// config-doc: archive.algorithm.single.concurrency 单层加密算法并发数
	viper.SetDefault("archive.algorithm.single.concurrency", 4)
	// config-doc: archive.algorithm.double.concurrency 双层加密算法并发数
	viper.SetDefault("archive.algorithm.double.concurrency", 4)

	// config-doc: recommend.limit 各维度推荐漫画数量上限
	viper.SetDefault("recommend.limit", 5)

	// === 从 cmd/server/config.go init() 移入 ===
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
	viper.SetDefault("server.gzip.level", 1) // gzip.BestSpeed

	// config-doc: server.ratelimit.enabled 是否启用限流
	viper.SetDefault("server.ratelimit.enabled", false)
	// config-doc: server.ratelimit.rps 每秒请求数限制
	viper.SetDefault("server.ratelimit.rps", 10)
	// config-doc: server.ratelimit.burst 限流突发大小
	viper.SetDefault("server.ratelimit.burst", 20)

	// config-doc: server.listen.http.addr HTTP 监听地址（host:port）
	viper.SetDefault("server.listen.http.addr", "0.0.0.0:8080")
	// config-doc: server.listen.tls.cert TLS 证书路径
	viper.SetDefault("server.listen.tls.cert", "")
	// config-doc: server.listen.tls.key TLS 私钥路径
	viper.SetDefault("server.listen.tls.key", "")
	// config-doc: server.listen.unix.path Unix 套接字路径
	viper.SetDefault("server.listen.unix.path", "")

	// config-doc: server.admin.token 管理端点鉴权 token（为空则仅放行 localhost）
	viper.SetDefault("server.admin.token", "")
	// config-doc: server.admin.allow_remote 是否允许远程访问管理端点
	viper.SetDefault("server.admin.allow_remote", false)
	// config-doc: server.shutdown_timeout 优雅关闭超时时间
	viper.SetDefault("server.shutdown_timeout", "5s")

	// config-doc: server.scheduler.enabled 是否启用调度器
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

	// === 从 pkg/logging/config.go init() 移入（统一文档记录） ===
	// config-doc: log.enableFile 是否启用文件日志
	viper.SetDefault("log.enableFile", false)
	// config-doc: log.filename 日志文件名
	viper.SetDefault("log.filename", "app.log")
	// config-doc: log.maxSize 日志文件最大大小（MB）
	viper.SetDefault("log.maxSize", 256)
	// config-doc: log.maxAge 日志文件保留天数
	viper.SetDefault("log.maxAge", 30)
	// config-doc: log.maxBackups 保留的旧日志文件最大数量
	viper.SetDefault("log.maxBackups", 5)
	// config-doc: log.localtime 是否使用本地时间
	viper.SetDefault("log.localtime", true)
	// config-doc: log.compress 是否压缩旧日志
	viper.SetDefault("log.compress", true)
	// config-doc: log.enableConsole 是否启用控制台日志
	viper.SetDefault("log.enableConsole", true)
	// config-doc: log.enableCaller 是否记录调用者信息
	viper.SetDefault("log.enableCaller", true)
	// config-doc: log.enableSourceIP 是否记录来源 IP
	viper.SetDefault("log.enableSourceIP", false)
	// config-doc: log.enablePID 是否记录进程 ID
	viper.SetDefault("log.enablePID", true)
	// config-doc: log.fileLevel 文件日志级别
	viper.SetDefault("log.fileLevel", "info")
	// config-doc: log.consoleLevel 控制台日志级别
	viper.SetDefault("log.consoleLevel", "debug")
	// config-doc: log.fileEncoding 文件日志编码格式
	viper.SetDefault("log.fileEncoding", "json")
	// config-doc: log.consoleEncoding 控制台日志编码格式
	viper.SetDefault("log.consoleEncoding", "console")
	// config-doc: log.appName 应用名称
	viper.SetDefault("log.appName", "")
	// config-doc: log.sourceEth 来源网卡
	viper.SetDefault("log.sourceEth", "eth3")
	// config-doc: log.disableTraceID 是否禁用 Trace ID
	viper.SetDefault("log.disableTraceID", false)

	// === 从 pkg/mongowrap/mongo.go init() 移入 ===
	// config-doc: mongo.user MongoDB 用户名
	viper.SetDefault("mongo.user", "cocom")
	// config-doc: mongo.password MongoDB 密码
	viper.SetDefault("mongo.password", "cocom123")
	// config-doc: mongo.host MongoDB 服务器地址
	viper.SetDefault("mongo.host", "localhost:27017")
	// config-doc: mongo.database MongoDB 数据库名
	viper.SetDefault("mongo.database", "cocom")
	// config-doc: mongo.authSource MongoDB 认证数据库
	viper.SetDefault("mongo.authSource", "cocom")

	// === 从 pkg/comic/config.go init() 移入 ===
	// config-doc: comic.verify.concurrent 校验并发数
	viper.SetDefault("comic.verify.concurrent", 10)
	// config-doc: comic.verify.task_buffer_size 校验任务缓冲区大小
	viper.SetDefault("comic.verify.task_buffer_size", 100)

	// === 从 pkg/download/downloader.go init() 移入 ===
	// config-doc: download.maxRunning 最大下载任务数
	viper.SetDefault("download.maxRunning", 10)
	// config-doc: download.downloadDir 下载目录
	viper.SetDefault("download.downloadDir", "Downloads")

	// === 从 pkg/archive/manager/config.go init() 移入 ===
	// config-doc: archive.manager.algorithm 归档算法 (single/double)
	viper.SetDefault("archive.manager.algorithm", string(archive.TypeDouble))
	// config-doc: archive.manager.meta_record_file_list 是否在元数据中记录文件列表
	viper.SetDefault("archive.manager.meta_record_file_list", false)
	// config-doc: archive.manager.replicates 远端复制目标列表
	viper.SetDefault("archive.manager.replicates", []string{})
	// config-doc: archive.manager.index.type 索引类型 (memory/file/mongo)
	viper.SetDefault("archive.manager.index.type", "memory")
	// config-doc: archive.manager.index.file_store_name 文件索引存储名称
	viper.SetDefault("archive.manager.index.file_store_name", "archive-manager-index")
	// config-doc: archive.manager.index.file_store_prefix 文件索引存储前缀
	viper.SetDefault("archive.manager.index.file_store_prefix", "archive/index")
	// config-doc: archive.manager.index.mongo_database MongoDB 索引数据库
	viper.SetDefault("archive.manager.index.mongo_database", "archiveManager")
	// config-doc: archive.manager.index.mongo_collection MongoDB 索引集合
	viper.SetDefault("archive.manager.index.mongo_collection", "archiveInfo")
	// config-doc: archive.manager.index.mongo_prefix MongoDB 索引键前缀
	viper.SetDefault("archive.manager.index.mongo_prefix", "")
	// config-doc: archive.manager.index.mongo_id_field MongoDB 索引 ID 字段名
	viper.SetDefault("archive.manager.index.mongo_id_field", "id")
	// config-doc: archive.manager.index.mongo_name_field MongoDB 索引名称字段名
	viper.SetDefault("archive.manager.index.mongo_name_field", "name")

	// === 从 cmd/server/internal/cache/cache.go init() 移入 ===
	// config-doc: cocom.cache.cleanInterval 缓存清理间隔
	viper.SetDefault("cocom.cache.cleanInterval", 1*time.Minute)
	// config-doc: cocom.cache.evictionInterval 缓存淘汰间隔
	viper.SetDefault("cocom.cache.evictionInterval", 10*time.Minute)

	// === 从 cmd/server/internal/comic/download.go init() 移入 ===
	// config-doc: comic.download.maxDownloadSize 最大并发下载数
	viper.SetDefault("comic.download.maxDownloadSize", 5)

	// === 从 cmd/server/internal/mongo/mongo.go init() 移入 ===
	// config-doc: comic.mongo.database 漫画 MongoDB 数据库名
	viper.SetDefault("comic.mongo.database", "cocom")
	// config-doc: comic.mongo.collections.comicInfo 漫画信息集合名
	viper.SetDefault("comic.mongo.collections.comicInfo", "comicInfo")
	// config-doc: comic.mongo.collections.oneComicInfo 单卷漫画信息集合名
	viper.SetDefault("comic.mongo.collections.oneComicInfo", "oneComicInfo")
	// config-doc: comic.mongo.collections.videoInfo 视频信息集合名
	viper.SetDefault("comic.mongo.collections.videoInfo", "videoInfo")
	// config-doc: comic.mongo.collections.settings 设置集合名
	viper.SetDefault("comic.mongo.collections.settings", "settings")
	// config-doc: comic.mongo.collections.custom 自定义集合名
	viper.SetDefault("comic.mongo.collections.custom", "custom")
	// config-doc: comic.mongo.collections.comicTag 漫画标签集合名
	viper.SetDefault("comic.mongo.collections.comicTag", "comicTag")
	// config-doc: comic.mongo.collections.tagRelation 标签关系集合名
	viper.SetDefault("comic.mongo.collections.tagRelation", "tagRelation")

	// === client / http ===
	// config-doc: client.server_addr 客户端请求的服务端地址
	viper.SetDefault("client.server_addr", "http://localhost:15456")
	// config-doc: http.enable_proxy 是否启用 HTTP 代理
	viper.SetDefault("http.enable_proxy", false)
	// config-doc: http.proxy HTTP 代理地址
	viper.SetDefault("http.proxy", "")
}
