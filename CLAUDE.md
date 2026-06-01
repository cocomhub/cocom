# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

> 仓库根目录 `../CLAUDE.md` 描述了 cocomhub 工作区的三个子项目以及跨子项目通用约定（中文回复、UTF-8、SPDX 头、构建产物位置、命名习惯等）。本文件只补充 **cocom 子项目** 专属的命令、架构与陷阱；与根 CLAUDE.md 冲突时以本文件为准，其余沿用。

## 子项目定位

- Module: `github.com/cocomhub/cocom`，Go 1.26。
- 单一二进制 `cocom`，同时是 Cobra CLI（漫画归档 / 校验 / 图片处理）和 Gin API Server。
- 依赖 MongoDB（元数据）+ 本地 FS（图片 / 归档）。

## 常用命令

均假设已在 `cocom/` 目录下。

```bash
make build              # fmt + 构建到 build/cocom（同时生成 shell 补全与 manpage）
make test               # go test -race -tags=memory_storage_integration -timeout 5m -coverprofile
make lint               # golangci-lint run
make fmt                # gofmt + gofumpt + addlicense + go fix
make cover              # 覆盖率 HTML 到 build/cover.html（先跑 nocover 校验）
make run-server         # build 后运行 ./build/cocom server --config ./build/conf/cocom.yaml
make install            # build 后拷贝二进制到 ~/bin，并安装 zsh 补全
make build-sub-tools    # 构建 tools/*/main.go 下所有子工具（arctl / pixcover / pixm）
make release-snapshot   # goreleaser snapshot 构建
```

单测：`go test -run TestName ./pkg/path/...`。若被测包内带有 `//go:build memory_storage_integration` 文件，须加 `-tags=memory_storage_integration`，否则会得到 “no tests to run” 假阴性。

`make nocover` 通过 `scripts/check-test-files.sh` 校验所有包都有测试文件——新增包时若暂无测试，覆盖率目标会失败。

## 架构要点（需读多文件才能掌握的部分）

### 启动与配置链
- `main.go` → `cmd.Execute()`（Cobra）。一级命令直接落在 `cmd/` 下：`ar.go`、`gallery*.go`、`image.go`、`verify.go`、`install.go`，以及子包 `cmd/cmv/`、`cmd/genwget/`、`cmd/server/`。
- 配置基于 Viper。`cmd/root.go` 注册了两个 `cobra.OnInitialize` 钩子，**每条命令执行前都会运行**：
  1. `rootcli.InitConfig`（`internal/rootcli/`）：加载配置文件、日志等；
  2. `initArchiveManager`：见下条。

### 存储注册表是全局状态
- 抽象在 `pkg/storage`，本地实现在 `pkg/storage/localfs`。三个命名 key 由 `internal/config` 提供：`StorageGalleryKey`、`StorageArchiveKey`、`StorageArchiveTempKey`。
- `initArchiveManager` 的固定顺序是：`storage.Clear()` → `localfs.SetFromViper(localfsBackendKeys...)` → `storage.SetFromViper()` → `manager.SetFromViper()`。
- **改动存储相关代码或测试时必须沿用 `Clear() + SetFromViper(...)` 模式**，否则会和全局注册表残留状态打架，出现”跨用例污染”一类的诡异失败。

### 存储抽象的两层架构

cocom 有两套存储抽象，职责不同、相互独立：

1. **`pkg/storage.Storage`（FS 层）** — 文件/对象存储接口
   - 方法: Put/Get/Stat/List/Delete/Copy/Move
   - 主要实现: localfs（本地文件系统）, baidupcs（百度网盘）
   - 用途: 漫画图片文件、存档文件的存取

2. **`pkg/comic.Storage`（业务数据层）** — 漫画元数据 CRUD 接口
   - 方法: Get/Update/Find/FindTotal/FindChannel/ArchiveByID/RestoreByID/SaveVerifyResult
   - 主要实现: MongoDB 各集合
   - 用途: 漫画信息、归档记录的查询和修改
   - 多个 MongoDB 实现位于 `cmd/server/internal/{comic,onecomic}/storage.go` 和 `pkg/comic/storage/mongo.go`

新增存储实现时，请根据职责选择正确的抽象层。`FindChannel` 通用分页循环已提取到 `pkg/comic/storage/base.go` 的 `FindChannelHelper`，新实现可直接复用。

### HTTP Server
- 位于 `cmd/server/`（`server.go`、`api/`、`handler/`、`view/`、`internal/`），基于 **Gin**（不是 `.cursorrules` 里写的 `net/http`，以代码为准）。
- 中间件链通过 Viper 配置开关：`server.cors.enabled`、`server.gzip.enabled`、`server.ratelimit.enabled`，访问日志走 `middlewares.AccessLog` + `server.access_log.patterns`。
- `/debug/pprof` 受 `middlewares.LocalGuard("debug.allow_remote")` 守护；`/admin/server/shutdown` 要么校验 `X-Admin-Token == admin.token`，要么仅放行 loopback。
- 集成 `gin-contrib/graceful` 做优雅停机，关闭信号通过 `shutdownCh` 传入。
- 旧版 `/api` 与 `/debug` 由 `handler.Init` 桥接到 net/http Mux（迁移期的双栈结构，新增端点请走 Gin）。

### 版本信息与脏构建
- 通过 `-ldflags` 注入 `pkg/version` 的 `Version`、`BuiltAt`、`CommitID`、`Branch`、`ReleaseURL`，**不要手工改这些常量**。
- `make prepare` 把当前 uncommitted diff 写入 `pkg/version/build/dirty_info.txt`，会被打包到二进制；`make build` 隐式依赖 `fmt → prepare`，本地脏改动会进入产物。

### 子工具
- `tools/<name>/main.go` 是独立的小 CLI（`arctl`、`pixcover`、`pixm`），由根 Makefile 的 `$(BuildDir)/%: tools/%/main.go` 规则统一构建到 `build/<name>`，共享同一 `pkg/version` 注入。

## cocom 专属编码风格

- 日志统一使用标准库 **`log/slog`**，通过 `pkg/logging` 初始化（内部使用 zap 引擎 + zapslog bridge 适配到 slog API）。
- 所有业务代码都应使用 `slog.InfoContext` / `slog.ErrorContext` 等标准 API。
- **不要直接 import** `go.uber.org/zap`，zap 已被封装在 `pkg/logging` 内部。
- 日志配置见 `docs/config.md` 的 `log.*` Viper 键（以 `log.` 为前缀，不是 `logging.`）。
- HTTP server 端用 **Gin**（不要按 `.cursorrules` 写的 `net/http + ServeMux`）。
- 允许保留 TODO / 占位符；新增功能时一并维护 `README.md` 与 `CHANGELOG.md`。
- 错误响应避免把原始 error 直接抛给客户端，做输入校验 + 合适的 HTTP 状态码 + 统一 JSON 格式。

## 测试约束

- 默认 `make test` 会带 `-race -tags=memory_storage_integration`。涉及 `pkg/storage` / `cmd/server/internal/comic` 等的包有专门走内存存储的集成路径，单跑某个包请加上同样的 tag。
- `cmd/server/settings_integration_test.go`、`graceful_run_test.go`、`pprof_test.go`、`middleware_test.go` 依赖完整 Viper + Gin 初始化，本质上是集成测试，跑前确保未污染全局 `viper` 配置（必要时在用例里 `viper.Reset()`）。
