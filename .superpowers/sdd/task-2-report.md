# Task 2: pkg 层 Viper 消除深度审查报告

## 审查范围

审查了以下 pkg 层的 Viper 消除情况：

| 文件 | 状态 | 备注 |
|------|------|------|
| `pkg/mongowrap/mongo.go` | ✅ 通过 | 无 `init()` 残留，无 `viper.Get` 调用，`Client()` 使用 `atomic.Bool` 保护 |
| `pkg/mongowrap/config.go` | ✅ 通过 | Config 结构体完整，字段完整 |
| `pkg/middlewares/cors.go` | ✅ 通过 | 通过 `internal/config.CORS` 结构体注入，无 viper 依赖 |
| `pkg/middlewares/ratelimit.go` | ✅ 通过 | 参数 `rps`/`burst` 直接传入，无 viper 依赖 |
| `pkg/middlewares/localguard.go` | ✅ 通过 | 参数 `allowRemote` 直接传入，无 viper 依赖 |
| `pkg/logging/config.go` | ✅ 通过 | 注释确认无 `init()` 残留，仅有 Config 结构体定义 |
| `pkg/archive/manager/config.go` | ✅ 通过 | `SetFromViper` 接受 Config 结构体，正确解耦 |
| `pkg/storage/config.go` | ✅ 通过 | `SetFromConfigs` 接受 `[]Config`，无 viper 依赖 |
| `pkg/comic/config.go` | ✅ 通过 | 仅为注释文件，无代码 |
| `pkg/download/config.go` | ✅ 已修复 | 新增 `EnableProxy`/`ProxyURL` 字段 |
| `pkg/download/downloader.go` | ✅ 已修复 | 新增 `SetEnableProxy`/`SetProxyURL` 方法 |
| `pkg/imaging/` | ✅ 通过 | 无 viper 依赖，无 config.go 文件 |

## 发现的问题及修复

### 问题 1: `pkg/download/config.go` 缺少 `EnableProxy`/`ProxyURL` 字段

**发现**：`pkg/download/downloader.go` 的 `DownloaderConfig` 结构体中有 `EnableProxy`/`ProxyURL` 字段，但 `pkg/download/config.go` 的 `Config` 结构体缺少这两个字段，导致 `NewInitConfig` 无法从 Config 注入代理配置。

**修复**：
- `pkg/download/config.go`: 在 `Config` 结构体中添加 `EnableProxy bool` 和 `ProxyURL string` 字段
- `pkg/download/downloader.go`: 添加 `SetEnableProxy`/`SetProxyURL` 方法，`NewInitConfig` 中注入代理配置
- `internal/config/manager.go`: 添加 `download.enableProxy`/`download.proxyURL` 默认值
- `internal/config/config_keys_test.go`: 添加对应测试用例
- `cmd/server/handler/init.go`: `download.Init` 调用中注入 `EnableProxy`/`ProxyURL`

### 问题 2: `http.enable_proxy`/`http.proxy` 死默认值

**发现**：`internal/config/manager.go` 中 `http.enable_proxy`/`http.proxy` 的 SetDefault 未被任何代码消费（无代码通过 `viper.Get("http.enable_proxy")` 读取）。

**修复**：迁移到 `download.enableProxy`/`download.proxyURL`，通过 Config 结构体注入。

### 问题 3: `pkg/archive/manager/README.md` 过时注释

**发现**：README 中 "默认从 viper 读取" 的注释已过时。

**修复**：更新为 "传入 IndexStore，默认使用内存索引"。

## 未通过测试

以下测试失败与本次审查无关（均为环境缺失）：

- `pkg/archive/archiver_test.go` — 需要 `7z` 命令（Windows 未安装）
- `pkg/imaging/image_test.go` — 需要 WebP 工具（`cocom install webp` 未运行）

## 验证结果

- `go build ./pkg/...` ✅ 通过
- `go vet ./pkg/...` ✅ 通过
- `go test ./internal/config/...` ✅ 通过
- `go test ./pkg/download/...` ✅ 通过
- `go test ./pkg/middlewares/...` ✅ 通过
- `go test ./pkg/mongowrap/...` ✅ 通过
- `go build ./cmd/server/handler/...` ✅ 通过

## 结论

pkg 层已无 `viper.Get` 调用残留，所有 `init()` 残留已清理，`config.go` 文件字段完整。`EnableProxy`/`ProxyURL` 已从 Config 结构体正确注入。