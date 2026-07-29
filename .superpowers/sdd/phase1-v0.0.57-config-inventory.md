# v0.0.57 配置键清单

> 生成时间: 2026-07-29
> 方法: 扫描 v0.0.57 tag 的所有 `viper.SetDefault` 和 `viper.Get*` 调用

## 配置键分组

### 1. server.* (HTTP 服务配置)

#### server.listen.http.addr
- 默认值: `"0.0.0.0:8080"`
- 定义: `cmd/server/config.go` (init)
- 消费方:
  - `cmd/server/server.go:103`: `viper.GetString("server.listen.http.addr")` → BuildEngine 监听地址
  - `cmd/server/server.go:170`: `viper.GetString("server.listen.http.addr")` → graceful 监听
- 传导路径: server.go BuildEngine → graceful.WithAddr
- 最终用途: HTTP 服务监听地址

#### server.listen.tls.cert / server.listen.tls.key
- 默认值: `""` / `""`
- 定义: `cmd/server/config.go` (init)
- 消费方: `cmd/server/server.go:171-172`: TLS 配置
- 传导路径: server.go → graceful.WithTLS
- 最终用途: HTTPS 证书路径

#### server.listen.unix.path
- 默认值: `""`
- 定义: `cmd/server/config.go` (init)
- 消费方: `cmd/server/server.go:173`: Unix socket 路径
- 传导路径: server.go → graceful.WithUnix
- 最终用途: Unix socket 监听路径

#### server.access_log.patterns
- 默认值: `["/debug", "/api", "/v1", "/v2"]`
- 定义: `cmd/server/config.go:14`
- 消费方: `cmd/server/server.go:41`: `viper.GetStringSlice("server.access_log.patterns")`
- 传导路径: server.go → middlewares.AccessLog
- 最终用途: 访问日志 URL 模式过滤

#### server.cors.*
- 定义: `cmd/server/config.go:16-22`
- 消费方: `cmd/server/server.go:42-43` + `pkg/middlewares/cors.go:15-23`
  - `server.cors.enabled`: `viper.GetBool("server.cors.enabled")` → 开关
  - `server.cors.allow_origins`: `viper.GetString("server.cors.allow_origins")` → CORS 源
  - `server.cors.allow_methods`: `viper.GetString("server.cors.allow_methods")` → CORS 方法
  - `server.cors.allow_headers`: `viper.GetString("server.cors.allow_headers")` → CORS 头
- 传导路径: server.go → middlewares.CORS() → cors.Config
- 最终用途: CORS 跨域配置

#### server.gzip.*
- 定义: `cmd/server/config.go:24-26`
- 消费方: `cmd/server/server.go:45-46`
- 传导路径: server.go → gzip.Gzip()
- 最终用途: Gzip 压缩配置

#### server.ratelimit.*
- 定义: `cmd/server/config.go:28-32`
- 消费方: `cmd/server/server.go:48-50`
- 传导路径: server.go → middlewares.RateLimit(rps, burst)
- 最终用途: 限流配置

#### server.admin.token
- 默认值: `""` (无默认值，零值)
- 定义: 无 SetDefault
- 消费方: `cmd/server/server.go:70`: `viper.GetString("admin.token")`
- ⚠️ **注意**: 消费方用 `"admin.token"` 而非 `"server.admin.token"`，键路径不一致！
- 传导路径: server.go → shutdown handler
- 最终用途: 管理端点鉴权

#### server.shutdown_timeout
- 默认值: `"5s"` (无 SetDefault，string 零值)
- 定义: 无 SetDefault
- 消费方: `cmd/server/server.go:164`: `viper.GetDuration("server.shutdown_timeout")`
- 传导路径: server.go → graceful.WithShutdownTimeout
- 最终用途: 优雅关闭超时

#### server.scheduler.*
- 定义: `cmd/server/config.go:35-73`
- 消费方:
  - `cmd/server/server.go:132`: `viper.GetBool("server.scheduler.enabled")`
  - `cmd/server/internal/scheduler/scheduler.go:21`: `viper.GetString("server.scheduler.timezone")`
  - `cmd/server/internal/scheduler/probe_comic.go:23-33`: 多个 GetBool/GetString/GetStringSlice
  - `cmd/server/internal/scheduler/archive_status_check.go:121-127`: 7 个 Get*
  - `cmd/server/internal/scheduler/cocoma_archiver.go:24-53`: 多个 Get*
  - `pkg/cocomaarchiver/archiver.go:42,64`: `cid_regex` 和 `limit`
- 传导路径: server.go → scheduler.New() → 各任务注册
- 最终用途: 调度器及子任务配置

### 2. mongo.* (MongoDB 连接配置)

#### mongo.*
- 定义: `pkg/mongowrap/mongo.go:28-36` (init)
- 消费方:
  - `pkg/mongowrap/mongo.go:40-44`: buildMongoDBURI 中 5 个 GetString
  - `pkg/mongowrap/mongo.go:64-66`: slog 日志中 3 个 GetString
- 传导路径: mongowrap.Init() → buildMongoDBURI → mongo.Connect
- 最终用途: MongoDB 连接参数

### 3. comic.* (漫画业务配置)

#### comic.mongo.*
- 定义: `cmd/server/internal/mongo/mongo.go:44-58` (init)
- 消费方:
  - `cmd/server/internal/mongo/mongo.go:64`: DB 数据库名
  - `cmd/server/internal/mongo/mongo.go:74-140`: 7 个 Collection 名
  - `cmd/ar.go:71-79`: 命令行工具中直接使用
- 传导路径: mongo.go init() → DB() → Collection()
- 最终用途: 漫画 MongoDB 集合配置

#### comic.verify.*
- 定义: `pkg/comic/config.go:10-12` (init)
- 消费方: 搜索无直接 `viper.Get` 引用（可能通过 config.Get 或参数传递）
- 最终用途: 漫画校验并发配置

#### comic.download.maxDownloadSize
- 定义: `cmd/server/internal/comic/download.go:31` (init)
- 消费方: `cmd/server/internal/comic/download.go:35`: `viper.GetInt32("comic.download.maxDownloadSize")`
- 传导路径: download.go Init() → maxDownloadSize 全局变量
- 最终用途: 下载并发限制

### 4. log.* (日志配置)

#### log.*
- 定义: `pkg/logging/config.go:17-51` (init)
- 消费方: `pkg/logging/config.go:56-73`: `GetConfigByViper()` 中 18 个 Get*
- 传导路径: logging.GetConfigByViper() → Config 结构体 → NewLogger
- 最终用途: 日志系统配置

### 5. download.* (下载配置)

#### download.maxRunning / download.downloadDir
- 定义: `pkg/download/downloader.go:30-32` (init)
- 消费方:
  - `pkg/download/downloader.go:37-38`: Init() 中 SetDownloadDir/SetMaxRunning
  - `pkg/comic/verify.go:519`: `viper.GetString("download.downloadDir")`
- 传导路径: download.Init() → DownloaderConfig → NewDownloader
- 最终用途: 下载参数配置

### 6. http.* (HTTP 代理配置)

#### http.enable_proxy / http.proxy
- 定义: 无 SetDefault
- 消费方: `pkg/download/downloader.go:150-151`: `viper.GetBool("http.enable_proxy")` + `viper.GetString("http.proxy")`
- 传导路径: downloader.go NewDownloader → 读取 viper → 设置 HTTP 代理
- 最终用途: 下载代理配置
- ⚠️ **注意**: 命名空间与功能不匹配 - 代理功能属于 download 包但键在 http.* 下

### 7. archive.* (归档配置)

#### archive.cmd / archive.password / archive.replicate
- 定义: `internal/config/config.go:34-38`
- 消费方: `internal/config/config.go:62-77`: 获取归档参数
- 传导路径: config.GetArchivePassword() → archive.Archiver
- 最终用途: 归档命令和密码

#### archive.algorithm.single.concurrency / archive.algorithm.double.concurrency
- 定义: `internal/config/config.go:40-42`
- 消费方: `pkg/archive/archiver.go:70,140`: `viper.GetInt(...)`
- 传导路径: archiver.go → NewSingle/NewDouble → 并发 channel
- 最终用途: 归档算法并发数

#### archive.root_dir
- 定义: 无 SetDefault
- 消费方: `internal/archivecli/commands.go:52`: `viper.GetString("archive.root_dir")`
- 最终用途: 归档根目录

#### archive.manager.*
- 定义: `pkg/archive/manager/config.go:14-34` (init)
- 消费方: `pkg/archive/manager/config.go:75-86`: SetFromViper 中 GetAll
- 传导路径: manager.SetFromViper() → Config 结构体 → 归档管理器
- 最终用途: 归档管理器配置

### 8. cocom.* (项目专属配置)

#### cocom.storage.path / cocom.archive.path / cocom.archive.temp_path
- 定义: `internal/config/config.go:20-24`
- 消费方: `internal/config/config.go:49-57`: GetGalleryStoragePath/GetArchiveStoragePath/GetArchiveTempStoragePath
- 传导路径: config.Get*() → storage/local 路径
- 最终用途: 存储路径配置

#### cocom.archive.password / cocom.archive.cmd / cocom.archive.replicate
- 定义: `internal/config/config.go:27-31`
- 消费方: `internal/config/config.go:62-77`: 与 archive.* 联合使用（fallback 链）
- 传导路径: 同 archive.*
- 最终用途: 归档参数（cocom 专属覆盖）

#### cocom.cache.cleanInterval / cocom.cache.evictionInterval
- 定义: `cmd/server/internal/cache/cache.go:22-24` (init)
- 消费方: `cmd/server/internal/cache/cache.go:28-29`: GetDuration
- 传导路径: cache.Init() → bigcache 配置
- 最终用途: 缓存配置

### 9. client.* (客户端配置)

#### client.server_addr
- 定义: 无 SetDefault
- 消费方:
  - `cmd/gallery.go:22`: `viper.GetString("client.server_addr")`
  - `cmd/genwget/genwget.go:129`: `viper.GetString("client.server_addr")`
- 最终用途: 客户端模式的服务端地址

### 10. recommend.* (推荐配置)

#### recommend.limit
- 定义: `internal/config/config.go:45`
- 消费方: `internal/config/config.go:82`: `viper.GetInt("recommend.limit")`
- 最终用途: 推荐数量上限

### 11. tools 工具配置

#### archive.manager.meta_record_file_list
- 工具特定覆盖: `tools/arctl/main.go:56` (默认 true), `tools/pixm/main.go:74` (默认 true)
- 用途: arctl/pixm 工具默认启用文件列表记录

#### archive.manager.index.type
- 工具特定覆盖: `tools/arctl/main.go:58` (file), `tools/pixm/main.go:76` (file)
- 用途: arctl/pixm 工具默认使用文件索引

#### storage.backends
- 工具特定覆盖: `tools/arctl/main.go:60`, `tools/pixm/main.go:78`
- 用途: 附加存储后端配置

#### arctl.output / pixm.output
- 消费方: `tools/arctl/main.go:93`, `tools/pixm/main.go:110`
- 用途: 输出格式控制

## 汇总统计

- 配置键总数 (SetDefault): ~90 个
- 配置键总数 (Get*): ~120 个
- 定义位置: 11 个文件中的 init() 函数
- 消费方: 分散在 20+ 文件中

## 已知问题

1. **`server.admin.token` 键路径不一致**: SetDefault 无定义，消费方用 `"admin.token"` 而非 `"server.admin.token"`
2. **`http.enable_proxy`/`http.proxy` 无 SetDefault**: 消费方在 downloader.go 中，但键路径不下 download.* 命名空间
3. **`server.shutdown_timeout` 无 SetDefault**: 消费方用 GetDuration，零值字符串可能导致解析失败
4. **`archive.root_dir` 无 SetDefault**: 消费方在 archivecli 中
5. **`client.server_addr` 无 SetDefault**: 消费方在 gallery.go 和 genwget.go 中