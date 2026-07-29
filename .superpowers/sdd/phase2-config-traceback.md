# Phase 2: 配置链路追踪报告

> 对比 v0.0.57 → 当前 HEAD (chore/cleanup-claude-trae-skills)
> 生成时间: 2026-07-29

## 1. server.* 链路

### server.listen.http.addr
- v0.0.57: `viper.GetString("server.listen.http.addr")` → BuildEngine / graceful.WithAddr
- 当前: `config.Get().Server.Listen.HTTP.Addr` → BuildEngine / graceful.WithAddr
- 传导路径: manager.go SetDefault → config.Init() → Manager.Get() → Unmarshal → Config.Server.Listen.HTTP.Addr → 消费方
- 状态: ✅ 正常迁移

### server.listen.tls.cert / server.listen.tls.key
- v0.0.57: `viper.GetString("server.listen.tls.cert")` → graceful.WithTLS
- 当前: `config.Get().Server.Listen.TLS.Cert` / `.Key`
- 状态: ✅ 正常迁移

### server.listen.unix.path
- v0.0.57: `viper.GetString("server.listen.unix.path")` → graceful.WithUnix
- 当前: `config.Get().Server.Listen.Unix.Path`
- 状态: ✅ 正常迁移

### server.access_log.patterns
- v0.0.57: `viper.GetStringSlice("server.access_log.patterns")` → middlewares.AccessLog
- 当前: `config.Get().Server.AccessLog.Patterns` → `cfg.AccessLog.Patterns`
- 传导路径: server.go BuildEngine → cfg.AccessLog.Patterns
- 状态: ✅ 正常迁移

### server.cors.*
- v0.0.57: `viper.GetBool/GetString` → middlewares.CORS() 内直接读 viper
- 当前: `config.Get().Server.CORS.*` → `cfg.CORS` → middlewares.CORS(cfg.CORS)
- 传导路径: BuildEngine → cfg.CORS → middlewares.CORS(cfg.CORS)
- 状态: ✅ 正常迁移，且从 viper 内聚改为参数注入

### server.gzip.*
- v0.0.57: `viper.GetBool/GetInt` → gzip.Gzip()
- 当前: `cfg.Gzip.Enabled` / `cfg.Gzip.Level`
- 状态: ✅ 正常迁移

### server.ratelimit.*
- v0.0.57: `viper.GetBool/GetInt` → middlewares.RateLimit()
- 当前: `cfg.RateLimit.RPS` / `cfg.RateLimit.Burst`
- 状态: ✅ 正常迁移

### server.admin.token
- v0.0.57: ❌ `viper.GetString("admin.token")` — 键路径不一致（无 `server.` 前缀），且无 SetDefault
- 当前: ✅ `cfg.Admin.Token` — 通过 Manager 结构体获取，有 SetDefault 空字符串
- 状态: ✅ 已修复 — 键路径一致化 + 参数注入

### server.admin.allow_remote
- v0.0.57: 无 SetDefault
- 当前: `v.SetDefault("server.admin.allow_remote", false)` 已添加
- 状态: ✅ 新增默认值

### server.shutdown_timeout
- v0.0.57: ❌ 无 SetDefault，`viper.GetDuration("server.shutdown_timeout")` 可能返回零值
- 当前: ✅ `v.SetDefault("server.shutdown_timeout", "5s")` — 有默认值，通过 `cfg.ShutdownTimeout` 获取
- 状态: ✅ 已修复

### server.scheduler.*
- v0.0.57: 所有子键通过 `viper.Get*` 直接读取
- 当前: `config.Get().Server.Scheduler.*` 结构体字段
- 传导路径: scheduler 各任务通过 `cfg:=config.Get().Server.Scheduler.ProbeComic` 获取
- 状态: ✅ 正常迁移

## 2. mongo.* 链路

### mongo.*
- v0.0.57: `viper.GetString` 在 mongowrap.Init() 中直接读取 5 个字段
- 当前: `mongowrap.Init(cfg Config)` 参数注入
- 传导路径: `handler/init.go` → `mongowrap.Init(config.Get().Mongo)`
- 状态: ✅ 正常迁移，从 viper 内聚改为参数注入

## 3. comic.* 链路

### comic.mongo.*
- v0.0.57: `viper.GetString` 在 mongo.go init() 中直接读取
- 当前: `config.Get().Comic.Mongo.*` 结构体字段
- 路径: `cmd/server/internal/mongo/mongo.go:47` → `config.Get().Comic.Mongo.Database`
- 状态: ✅ 正常迁移

### comic.verify.*
- v0.0.57: `viper.SetDefault` 在 `pkg/comic/config.go` 中，但无 `viper.Get` 消费方
- 当前: `v.SetDefault` 在 `manager.go` 中，消费方通过 `comic.VerifyConfig` 结构体传递
- 传导路径: 通过 `comic.Init(ctx, cfg.Comic.Verify.Concurrent, ...)` 参数注入
- 状态: ✅ 正常迁移（但消费方间接 — 通过 comic.Init 参数）

### comic.download.maxDownloadSize
- v0.0.57: `viper.GetInt32` 在 `cmd/server/internal/comic/download.go` 中
- 当前: 同上（但 viper 实例已改为 Manager 的 viper）
- 传导路径: 通过 Manager 的 viper 获取，未使用 Config 结构体
- 状态: ⚠️ 部分迁移 — 键路径在 manager.go 有 SetDefault 但消费方仍用 `viper.GetInt32`（全局 viper 还是 Manager 的 viper？）

## 4. log.* 链路

### log.*
- v0.0.57: `viper.GetBool/GetString/GetInt` 在 `pkg/logging/config.go` 中 18 个调用
- 当前: `logging.Config` 结构体直接从 `config.Get().Log` 注入
- 传导路径: `cmd/root.go` → `logging.Init(config.Get().Log)`
- 状态: ✅ 正常迁移，从 viper 内聚改为参数注入

## 5. download.* 链路

### download.maxRunning / download.downloadDir
- v0.0.57: `viper.GetString/GetInt` 在 `downloader.go` 中
- 当前: `download.Config` 结构体从 `config.Get().Download` 注入
- 传导路径: `handler/init.go` → `download.Init(download.Config{...})`
- 状态: ✅ 正常迁移

### download.enableProxy / download.proxyURL
- v0.0.57: ❌ **不存在** — 代理配置走 `http.*` 命名空间
- 当前: ✅ 新添加，从 `download.Config` 结构体注入
- 传导路径: `handler/init.go` → `download.Init(cfg)` → `NewInitConfig(cfg)` → `SetEnableProxy/SetProxyURL`
- 状态: ✅ 新增（修复了 http.* 断链问题）

## 6. http.* 链路（已移除）

### http.enable_proxy / http.proxy
- v0.0.57: ❌ `viper.GetBool("http.enable_proxy")` + `viper.GetString("http.proxy")` 在 `downloader.go` 中直接读取
- 当前: ❌ 已移除，功能迁移到 `download.enableProxy` / `download.proxyURL`
- 状态: ✅ 已迁移

## 7. archive.* 链路

### archive.cmd / archive.password / archive.replicate
- v0.0.57: `viper.GetString` 在 `internal/config/config.go` 中
- 当前: `config.Get().Archive.*` 和 `config.Get().Cocom.Archive.*` 双路径（fallback 链）
- 传导路径: `cmd/root.go` 中 `storage.SetFromConfigs` 使用 `Cocom.Archive.*`，但 `cmd/server/internal/comic/storage.go` 中 `config.Get().Cocom.Archive.*`
- 状态: ✅ 双路径正常维护

### archive.algorithm.*
- v0.0.57: `viper.GetInt` 在 `pkg/archive/archiver.go` 中直接读取
- 当前: 通过 `archive.InitConcurrency(single, double)` 参数注入（从 `config.Get().Cocom.Archive.Algorithm.*`）
- 传导路径: `cmd/root.go` → `archive.InitConcurrency(config.Get().Cocom.Archive.Algorithm.Single.Concurrency, ...)`
- 状态: ✅ 正常迁移

### archive.root_dir
- v0.0.57: 无 SetDefault，`viper.GetString("archive.root_dir")` 在 `archivecli/commands.go`
- 当前: ✅ `v.SetDefault("archive.root_dir", "")` 已添加
- 状态: ✅ 新增默认值

### archive.manager.*
- v0.0.57: `viper.Get*("archive.manager.*")` 在 `manager/config.go` 中
- 当前: `manager.Config` 结构体从 `config.Get().Archive.Manager` 构造
- 传导路径: `cmd/root.go` → `manager.SetFromViper(manager.Config{...})`
- 状态: ✅ 正常迁移

## 8. cocom.* 链路

### cocom.storage.* / cocom.archive.* / cocom.cache.*
- v0.0.57: 通过 `internal/config/config.go` 中的 Get* 函数获取
- 当前: 通过 `config.Get().Cocom.*` 结构体字段获取
- 状态: ✅ 正常迁移

## 9. client.* 链路

### client.server_addr
- v0.0.57: ❌ 无 SetDefault，`viper.GetString("client.server_addr")` 在 genwget/gallery 中
- 当前: ✅ `v.SetDefault("client.server_addr", "http://localhost:15456")` 已添加，`config.Get().Client.ServerAddr` 结构体字段
- 状态: ✅ 新增默认值 + 结构体化

## 10. recommend.* 链路

### recommend.limit
- v0.0.57: `viper.GetInt("recommend.limit")` 在 `internal/config/config.go` 中
- 当前: `config.Get().Recommend.Limit` 结构体字段
- 传导路径: `cmd/server/handler/recommend.go` → `config.Get().Recommend.Limit`
- 状态: ✅ 正常迁移

## 异常汇总

| 异常类型 | 数量 | 详情 |
|---------|------|------|
| 已修复（v0.0.57 问题→当前已解决） | 6 | `server.admin.token` 键路径、`server.shutdown_timeout` 无默认值、`http.*` 命名空间、`archive.root_dir` 无默认值、`client.server_addr` 无默认值、`comic.download.maxDownloadSize` 与 `cocom.cache.*` 参数注入 |
| 待确认 | 0 | — |

## 详细说明

### comic.download.maxDownloadSize — 已完全迁移

**v0.0.57**: `cmd/server/internal/comic/download.go:35` 中 `viper.GetInt32("comic.download.maxDownloadSize")` 从全局 viper 读取，`init()` 中设 `viper.SetDefault`

**当前**: `cmd/server/internal/comic/download.go:29` 中 `func Init(ctx context.Context, maxSize int32)` 参数注入

传导路径: `handler/init.go` → `comic.Init(ctx, cfg.Comic.Download.MaxDownloadSize)` → `config.Get().Comic.Download.MaxDownloadSize`

**状态: ✅ 已完全迁移 — 通过参数注入**

### cocom.cache.* — 已完全迁移

**v0.0.57**: `cache.go:28-29` 中 `viper.GetDuration("cocom.cache.evictionInterval")` 和 `viper.GetDuration("cocom.cache.cleanInterval")`

**当前**: `cache.go:20` 中 `func Init(ctx context.Context, evictionInterval, cleanInterval time.Duration)` 参数注入

传导路径: `handler/init.go` → `cache.Init(ctx, cfg.Cocom.Cache.EvictionInterval, cfg.Cocom.Cache.CleanInterval)` → `config.Get().Cocom.Cache.*`

**状态: ✅ 已完全迁移 — 通过参数注入**