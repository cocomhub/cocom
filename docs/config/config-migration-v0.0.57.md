# cocom 配置迁移说明文档

> 生成时间: 2026-07-29
> 对比范围: v0.0.57 (3be8caa) → 1bfcc1f
> 方法: 全链路追踪 + 逆向分析

---

## 一、迁移概要

### 迁移前后对比

| 维度 | v0.0.57 | 当前 HEAD | 说明 |
|------|---------|-----------|------|
| 配置存储方式 | 分散在 11 个文件的 `init()` 中 | 集中在 `internal/config/manager.go` 的 `setDefaultsOn()` | 单轨化 |
| 配置读取方式 | `viper.Get*` 直接读取 | `config.Get().Field` 结构体字段 + 参数注入 | 类型安全 |
| 配置定义文件 | 11 个文件 | 1 个文件 (+ 2 个工具覆盖) | 集中管理 |
| 配置键总数 (`SetDefault`) | ~90 个 | ~105 个 | 新增 15 个 |
| 无默认值的消费方 | 5 个 | 0 个 | 全部补齐 |
| 命名空间不匹配 | 1 处 (`http.*`) | 0 处 | 已迁移 |
| 测试覆盖 | 极少 | 48 个用例 (config 包) | 大幅增强 |

### 迁移类型

1. **集中化**: 所有 `viper.SetDefault` 从各包 `init()` 迁移到 `internal/config/manager.go`
2. **结构化**: 所有配置通过 `Config` 结构体 (`internal/config/types.go`) 的 `mapstructure` 标签映射
3. **参数注入**: pkg 层通过构造函数参数接收配置，不再直接调用 `viper.Get`
4. **双路径兼容**: `archive.*` 和 `cocom.archive.*` 双路径保留，确保存量 YAML 兼容

---

## 二、配置键迁移清单

### 2.1 server.* 组

| 配置键 | v0.0.57 来源 | 当前路径 | 迁移方式 | 状态 |
|--------|-------------|---------|---------|------|
| `server.listen.http.addr` | cmd/server/config.go | config.Get().Server.Listen.HTTP.Addr | 结构体 | ✅ |
| `server.listen.tls.cert` | cmd/server/config.go | config.Get().Server.Listen.TLS.Cert | 结构体 | ✅ |
| `server.listen.tls.key` | cmd/server/config.go | config.Get().Server.Listen.TLS.Key | 结构体 | ✅ |
| `server.listen.unix.path` | cmd/server/config.go | config.Get().Server.Listen.Unix.Path | 结构体 | ✅ |
| `server.access_log.patterns` | cmd/server/config.go | config.Get().Server.AccessLog.Patterns | 结构体 | ✅ |
| `server.cors.*` | cmd/server/config.go | config.Get().Server.CORS.* | 结构体→参数注入 | ✅ |
| `server.gzip.*` | cmd/server/config.go | config.Get().Server.Gzip.* | 结构体 | ✅ |
| `server.ratelimit.*` | cmd/server/config.go | config.Get().Server.RateLimit.* | 结构体 | ✅ |
| `server.admin.token` | ❌ 无默认值 + 键路径错误 | config.Get().Server.Admin.Token | 修复 | ✅ |
| `server.admin.allow_remote` | ❌ 无默认值 | config.Get().Server.Admin.AllowRemote | 修复 | ✅ |
| `server.shutdown_timeout` | ❌ 无默认值 | config.Get().Server.ShutdownTimeout | 修复 | ✅ |
| `server.scheduler.*` | cmd/server/config.go | config.Get().Server.Scheduler.* | 结构体 | ✅ |

### 2.2 mongo.* 组

| 配置键 | v0.0.57 来源 | 当前路径 | 迁移方式 | 状态 |
|--------|-------------|---------|---------|------|
| `mongo.*` | pkg/mongowrap/mongo.go | mongowrap.Init(config.Get().Mongo) | 参数注入 | ✅ |

### 2.3 comic.* 组

| 配置键 | v0.0.57 来源 | 当前路径 | 迁移方式 | 状态 |
|--------|-------------|---------|---------|------|
| `comic.mongo.*` | cmd/server/internal/mongo/mongo.go | config.Get().Comic.Mongo.* | 结构体 | ✅ |
| `comic.verify.*` | pkg/comic/config.go | config.Get().Comic.Verify.* | 结构体→参数注入 | ✅ |
| `comic.download.maxDownloadSize` | cmd/server/internal/comic/download.go | comic.Init(maxSize) | 参数注入 | ✅ |

### 2.4 log.* 组

| 配置键 | v0.0.57 来源 | 当前路径 | 迁移方式 | 状态 |
|--------|-------------|---------|---------|------|
| `log.*` (18个) | pkg/logging/config.go | logging.Init(config.Get().Log) | 参数注入 | ✅ |

### 2.5 download.* 组

| 配置键 | v0.0.57 来源 | 当前路径 | 迁移方式 | 状态 |
|--------|-------------|---------|---------|------|
| `download.maxRunning` | pkg/download/downloader.go | download.Init(config.Get().Download) | 参数注入 | ✅ |
| `download.downloadDir` | pkg/download/downloader.go | download.Init(config.Get().Download) | 参数注入 | ✅ |
| `download.enableProxy` | 🔄 从 `http.enable_proxy` 迁移 | download.Init(config.Get().Download) | 新增路径 | ✅ |
| `download.proxyURL` | 🔄 从 `http.proxy` 迁移 | download.Init(config.Get().Download) | 新增路径 | ✅ |

### 2.6 http.* 组（已移除）

| 配置键 | 处理方式 | 迁移目标 |
|--------|---------|---------|
| `http.enable_proxy` | 🔄 已迁移 | `download.enableProxy` |
| `http.proxy` | 🔄 已迁移 | `download.proxyURL` |

### 2.7 archive.* 组

| 配置键 | v0.0.57 来源 | 当前路径 | 迁移方式 | 状态 |
|--------|-------------|---------|---------|------|
| `archive.password` | internal/config/config.go | config.Get().Archive.Password | 双路径 | ✅ |
| `archive.cmd` | internal/config/config.go | config.Get().Archive.Cmd | 双路径 | ✅ |
| `archive.replicate` | internal/config/config.go | config.Get().Archive.Replicate | 双路径 | ✅ |
| `archive.algorithm.*` | internal/config/config.go | archive.InitConcurrency(...) | 参数注入 | ✅ |
| `archive.root_dir` | ❌ 无默认值 | config.Get().Archive.RootDir | 修复 | ✅ |
| `archive.manager.*` | pkg/archive/manager/config.go | manager.SetFromViper(config.Get().Archive.Manager) | 结构体 | ✅ |

### 2.8 cocom.* 组

| 配置键 | v0.0.57 来源 | 当前路径 | 迁移方式 | 状态 |
|--------|-------------|---------|---------|------|
| `cocom.storage.path` | internal/config/config.go | config.Get().Cocom.Storage.Path | 结构体 | ✅ |
| `cocom.archive.path` | internal/config/config.go | config.Get().Cocom.Archive.Path | 结构体 | ✅ |
| `cocom.archive.temp_path` | internal/config/config.go | config.Get().Cocom.Archive.TempPath | 结构体 | ✅ |
| `cocom.archive.password` | internal/config/config.go | config.Get().Cocom.Archive.Password | 双路径 | ✅ |
| `cocom.archive.cmd` | internal/config/config.go | config.Get().Cocom.Archive.Cmd | 双路径 | ✅ |
| `cocom.archive.replicate` | internal/config/config.go | config.Get().Cocom.Archive.Replicate | 双路径 | ✅ |
| `cocom.archive.algorithm.*` | internal/config/config.go | archive.InitConcurrency(...) | 参数注入 | ✅ |
| `cocom.cache.*` | cmd/server/internal/cache/cache.go | cache.Init(evictionInterval, cleanInterval) | 参数注入 | ✅ |

### 2.9 client.* 组

| 配置键 | v0.0.57 来源 | 当前路径 | 迁移方式 | 状态 |
|--------|-------------|---------|---------|------|
| `client.server_addr` | ❌ 无默认值 | config.Get().Client.ServerAddr | 修复 | ✅ |

### 2.10 recommend.* 组

| 配置键 | v0.0.57 来源 | 当前路径 | 迁移方式 | 状态 |
|--------|-------------|---------|---------|------|
| `recommend.limit` | internal/config/config.go | config.Get().Recommend.Limit | 结构体 | ✅ |

---

## 三、发现并修复的问题

### 3.1 已修复的问题（共 6 个）

| # | 问题 | 严重度 | 修复方式 | 修复提交 |
|---|------|--------|---------|---------|
| 1 | `http.enable_proxy`/`http.proxy` 死键 — 命名空间不匹配，`http.*` 键无任何代码消费 | 高 | 迁移到 `download.enableProxy`/`download.proxyURL` | `cdf1f0d` |
| 2 | `download.Config` 缺少 `EnableProxy`/`ProxyURL` 字段 → 代理配置无法注入 | 高 | 补全 Config 结构体 + 注入链 | `cdf1f0d` |
| 3 | `server.admin.token` 消费方用 `"admin.token"` 而非 `"server.admin.token"`，键路径不一致 | 中 | 统一为 `Config.Server.Admin.Token` | 重构中修复 |
| 4 | `server.shutdown_timeout` / `archive.root_dir` / `client.server_addr` 无 SetDefault 默认值 | 中 | 全部补齐 | 重构中修复 |
| 5 | `types.go` 中残留未使用的 `Log`/`Mongo`/`Download` 死类型 | 低 | 移除 | `8ac90d3` |
| 6 | `pkg/logging/config.go` 和 `cmd/server/internal/mongo/mongo.go` 残留空 `init()` | 低 | 移除 | `f2e9954`, `aeb6647` |

### 3.2 潜在问题

| # | 问题 | 说明 | 建议 |
|---|------|------|------|
| 1 | `log.appName` 默认值从动态 `filepath.Base(os.Args[0])` 变为空字符串 | 如果依赖进程名作为日志标识，此变更可能影响 | 确认是否需要恢复动态默认值 |
| 2 | `pkg/download/download_test.go` 是空壳测试 | 仅 `t.Log()`，不验证任何逻辑 | 后续补充实际测试 |
| 3 | `pkg/archive/archiver.go` 中 `archive.algorithm.*` 仍通过 `cmp.Or(viper.GetInt(...), 1)` 读取 | 但 `InitConcurrency` 已通过参数设置，该路径可能在运行时被覆盖 | 确认是否需移除 viper 后备逻辑 |

---

## 四、架构图

### 配置初始化流程

```
main.go
  └─ cmd.Execute()
       └─ cobra.OnInitialize (顺序执行)
            ├─ 1. rootcli.InitConfig()     ← 加载 YAML 文件 + 环境变量
            │    └─ viper.SetConfigFile(cfgFile)
            │    └─ viper.ReadInConfig()
            │    └─ viper.SetEnvPrefix("COCOM")
            │    └─ viper.AutomaticEnv()
            │
            ├─ 2. config.Init()            ← 同步到 Manager 实例
            │    └─ global.SetDefaults()
            │    └─ global.v.MergeInConfig()  ← 合并 YAML 值
            │    └─ global.v.SetEnvPrefix("COCOM")
            │    └─ global.v.AutomaticEnv()
            │    └─ global.Reset()
            │
            ├─ 3. initLogging()            ← 初始化日志
            │    └─ logging.Init(config.Get().Log)
            │
            └─ 4. initArchiveManager()     ← 初始化归档管理器
                 └─ storage.Clear()
                 └─ localfs.SetFromMap(...)
                 └─ storage.SetFromConfigs(...)
                 └─ archive.InitConcurrency(...)
                 └─ manager.SetFromViper(...)
```

### 配置消费路径

```
v.SetDefault(key, value)                          ← 定义默认值
  │
  ▼
Manager.SetDefaults() → Manager.Get() → Unmarshal → Config 结构体
  │                                                    │
  │  (通过 config.Init() 同步)                          │
  ▼                                                    ▼
config.Get().Field                               ← 消费方读取
  │
  ├─ 直接使用: server.go, mongo.go, scheduler 等
  ├─ 参数注入: logging.Init(cfg), download.Init(cfg), cache.Init(...)
  └─ 结构体传递: manager.SetFromViper(cfg), middlewares.CORS(cfg)
```

---

## 五、cocom-gen.yaml 配置参考

`cocom-gen.yaml` 是新生成的完整配置参考文件，包含所有配置项及环境变量覆盖路径注释。

### 配置结构

```yaml
cocom:       # 项目专属配置
  ├─ storage:     # 存储路径
  ├─ archive:     # 归档参数（cocom 专属覆盖）
  └─ cache:       # 缓存配置
archive:     # 可复用归档基础设施
  ├─ root_dir     # 归档根目录
  ├─ manager:     # 归档管理器
  │   ├─ algorithm
  │   ├─ replicates
  │   └─ index:   # 索引配置
  └─ ...          # 算法并发
mongo:       # MongoDB 连接
comic:       # 漫画业务
  ├─ verify:      # 校验参数
  ├─ download:    # 下载限制
  └─ mongo:       # 漫画 MongoDB 集合
server:      # HTTP 服务
  ├─ listen:      # 监听地址 (HTTP/TLS/Unix)
  ├─ admin:       # 管理端点
  ├─ scheduler:   # 调度器及子任务
  ├─ cors:        # CORS 配置
  ├─ gzip:        # 压缩配置
  └─ ratelimit:   # 限流配置
log:         # 日志配置
download:    # 下载配置
recommend:   # 推荐配置
client:      # 客户端模式
```

---

## 六、升级指南

### 从 v0.0.57 升级

1. **YAML 配置文件**: `cocom.yaml` 中的旧键仍然有效（`archive.*` 双路径兼容），但建议迁移到 `cocom-gen.yaml` 的新结构
2. **环境变量前缀**: 保持 `COCOM_` 前缀不变
3. **关键变化**:
   - `http.enable_proxy`/`http.proxy` → `download.enableProxy`/`download.proxyURL`
   - `server.admin.token` 键路径不变（无需改 YAML）
   - `server.shutdown_timeout` 新增默认值 `"5s"`
4. **行为变化**:
   - `log.appName` 默认值从进程名变为空字符串
   - 如有依赖 `log.appName` 自动填充，需在 YAML 中显式配置