# Phase 4: 新增配置分析

> 对比 v0.0.57 → 当前 HEAD，找出重构过程中新增的配置项

## 新增配置键

### 1. server.admin.token (新增默认值)
- **v0.0.57**: 无 SetDefault，消费方用 `viper.GetString("admin.token")`（键路径不一致）
- **当前**: `v.SetDefault("server.admin.token", "")`，消费方用 `cfg.Admin.Token`
- **用途**: 管理端点鉴权 token
- **必要性**: ✅ 必要 — 修复了缺失默认值 + 键路径不一致
- **默认值**: 空字符串（仅放行 loopback）

### 2. server.admin.allow_remote (新增默认值)
- **v0.0.57**: 无 SetDefault
- **当前**: `v.SetDefault("server.admin.allow_remote", false)`
- **用途**: 是否允许远程访问管理端点
- **必要性**: ✅ 必要 — 新增默认值

### 3. server.shutdown_timeout (新增默认值)
- **v0.0.57**: 无 SetDefault，`viper.GetDuration("server.shutdown_timeout")` 可能返回零值
- **当前**: `v.SetDefault("server.shutdown_timeout", "5s")`
- **用途**: 优雅关闭超时时间
- **必要性**: ✅ 必要 — 修复了缺失默认值

### 4. download.enableProxy (新增)
- **v0.0.57**: 不存在，代理功能走 `http.enable_proxy`（无 SetDefault）
- **当前**: `v.SetDefault("download.enableProxy", false)` — 新键路径
- **用途**: 下载代理开关
- **必要性**: ✅ 必要 — 修复了 http.* 命名空间不匹配问题

### 5. download.proxyURL (新增)
- **v0.0.57**: 不存在，代理功能走 `http.proxy`（无 SetDefault）
- **当前**: `v.SetDefault("download.proxyURL", "")` — 新键路径
- **用途**: 下载代理地址
- **必要性**: ✅ 必要 — 修复了 http.* 命名空间不匹配问题

### 6. archive.root_dir (新增默认值)
- **v0.0.57**: 无 SetDefault，`viper.GetString("archive.root_dir")` 在 archivecli 中
- **当前**: `v.SetDefault("archive.root_dir", "")`
- **用途**: 归档根目录（archivecli 使用）
- **必要性**: ✅ 必要 — 修复了缺失默认值

### 7. client.server_addr (新增默认值)
- **v0.0.57**: 无 SetDefault，`viper.GetString("client.server_addr")` 在 genwget/gallery 中
- **当前**: `v.SetDefault("client.server_addr", "http://localhost:15456")`
- **用途**: 客户端模式的服务端地址
- **必要性**: ✅ 必要 — 修复了缺失默认值

### 8. server.scheduler.* (新增 config-doc 注释)
- 18 个 scheduler 子键新增了 `config-doc` 注释（之前仅 `server.scheduler.enabled` 有注释）
- 不影响功能，仅增强文档完整性
- **必要性**: ✅ 文档增强

## 变更的配置键（非新增）

### archive.archive.* → cocom.archive.* (双路径)
- v0.0.57: `cocom.archive.*` 已有，但 `internal/config/config.go` 中有 fallback 逻辑
- 当前: 双路径继续维护，`manager.go` 中对两个路径都设了 SetDefault
- **用途**: cocom 专属归档配置覆盖

### log.appName 默认值变更
- v0.0.57: `viper.SetDefault("log.appName", AppName)` 其中 `AppName = filepath.Base(os.Args[0])`
- 当前: `v.SetDefault("log.appName", "")` 硬编码空字符串
- **影响**: 行为变更 — 默认值从动态的进程名变为空字符串
- **建议**: 确认此变更是否故意。如进程名作为默认值很重要，应保留动态逻辑

## 新增配置总结

| 配置键 | 类型 | 必要性 | 备注 |
|--------|------|--------|------|
| `server.admin.token` | 新增默认值 | ✅ | 修复缺失 |
| `server.admin.allow_remote` | 新增默认值 | ✅ | 修复缺失 |
| `server.shutdown_timeout` | 新增默认值 | ✅ | 修复缺失 |
| `download.enableProxy` | 新增键 | ✅ | 迁移 http.* |
| `download.proxyURL` | 新增键 | ✅ | 迁移 http.* |
| `archive.root_dir` | 新增默认值 | ✅ | 修复缺失 |
| `client.server_addr` | 新增默认值 | ✅ | 修复缺失 |
| `log.appName` 默认值 | 行为变更 | ⚠️ | 动态→空字符串 |