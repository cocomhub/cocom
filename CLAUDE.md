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

### API 响应格式

所有 API 返回统一 `{head, body}` 结构，定义在 `pkg/httpwrap/http.go`：

```json
{
    "head": {
        "code": 0,
        "msg": "succ",
        "request_id": "...",
        "time": "2026-06-10T22:54:58.578339+08:00"
    },
    "body": { }
}
```

- 成功：`code=0, msg="succ"`，数据放 body。使用 `httpwrap.ResponseSucc(ctx, w, body)`（net/http handler）或 `httpwrap.GinRespondOK(c, body)`（Gin handler）。
- 失败：错误码 < 0，`httpwrap.ResponseFail(ctx, w, msg)` 或 `httpwrap.GinRespondError(c, httpStatus, errCode, msg)`。标准错误码见 `pkg/httpwrap/errcode.go`（ErrCodeUnknown=-1, ErrCodeInvalid=-2, ErrCodeNotFound=-3, ErrCodeForbidden=-4, ErrCodeInternal=-5）。
- 新增 API 必须遵循此格式，不要手工拼 JSON。

## Web 页面开发约定

- **CSS 修改集中到 `custom/css/styles.css`**：所有覆盖/增强样式写在此文件，不动 vendor CSS（`static.nhentai.net/css/` 下的固定版本文件）。
- **JS 修改集中到 `custom/js/scripts.js`**：所有自定义 JS 逻辑写在此文件，不动 vendor JS（`static.nhentai.net/js/` 下的固定版本文件）。
- **模板不直接嵌入 CSS/JS**：CSS 在 `<head>` 通过 `<link>` 引入，JS 在 `<body>` 末尾通过 `<script>` 引入。Inline style/script 仅限于单页面初始化数据（如 `window._gallery = ...`）。
- **模板引用新资源时**：在 `head.tpl` 中添加，不要逐个修改页面级 tpl。

## 测试约束

- 默认 `make test` 会带 `-race -tags=memory_storage_integration`。涉及 `pkg/storage` / `cmd/server/internal/comic` 等的包有专门走内存存储的集成路径，单跑某个包请加上同样的 tag。
- `cmd/server/settings_integration_test.go`、`graceful_run_test.go`、`pprof_test.go`、`middleware_test.go` 依赖完整 Viper + Gin 初始化，本质上是集成测试，跑前确保未污染全局 `viper` 配置（必要时在用例里 `viper.Reset()`）。
- **Handler 测试现在使用 MemoryStorage 而非 MongoDB**（通过 `internal/comic.SetDefaultStorage` 注入）。`search_test.go` 和 `tags_search_test.go` 仍依赖 MongoDB，需要在有 MongoDB 的环境运行。
- **扩展 Storage 接口时需同时修改**：接口声明（`pkg/comic/storage.go`）、MemoryStorage 实现（同文件）、MongoStorage 占位（`pkg/comic/storage/mongo.go`）、内部桥接（`cmd/server/internal/comic/storage.go` 和 `cmd/server/internal/onecomic/storage.go`），以及 Comic 接口依赖的新增方法（`pkg/comic/comic.go` + `cmd/server/internal/comic/comic.go` + `cmd/server/internal/onecomic/comic.go`）。
- **Comic 接口的 MarshalJSON 递归陷阱**：`ComicImpl.MarshalJSON()` 必须用 `type comicAlias ComicImpl` 技巧切断递归，否则栈溢出。

<!-- superpowers-zh:begin (do not edit between these markers) -->
# Superpowers-ZH 中文增强版

本项目已安装 superpowers-zh 技能框架（20 个 skills）。

## 核心规则

1. **收到任务时，先检查是否有匹配的 skill** — 哪怕只有 1% 的可能性也要检查
2. **设计先于编码** — 收到功能需求时，先用 brainstorming skill 做需求分析
3. **测试先于实现** — 写代码前先写测试（TDD）
4. **验证先于完成** — 声称完成前必须运行验证命令

## 可用 Skills

Skills 位于 `.claude/skills/` 目录，每个 skill 有独立的 `SKILL.md` 文件。

- **brainstorming**: 在任何创造性工作之前必须使用此技能——创建功能、构建组件、添加功能或修改行为。在实现之前先探索用户意图、需求和设计。
- **chinese-code-review**: 中文 review 沟通参考——话术模板、分级标注（必须修复/建议修改/仅供参考）、国内团队常见反模式应对。仅在用户显式 /chinese-code-review 时调用，不要根据上下文自动触发。
- **chinese-commit-conventions**: 中文 commit 与 changelog 配置参考——Conventional Commits 中文适配、commitlint/husky/commitizen 中文模板、conventional-changelog 中文配置。仅在用户显式 /chinese-commit-conventions 时调用，不要根据上下文自动触发。
- **chinese-documentation**: 中文文档排版参考——中英文空格、全半角标点、术语保留、链接格式、中文文案排版指北约定。仅在用户显式 /chinese-documentation 时调用，不要根据上下文自动触发。
- **chinese-git-workflow**: 国内 Git 平台配置参考——Gitee、Coding.net、极狐 GitLab、CNB 的 SSH/HTTPS/凭据/CI 接入差异与镜像同步配置。仅在用户显式 /chinese-git-workflow 时调用，不要根据上下文自动触发。
- **dispatching-parallel-agents**: 当面对 2 个以上可以独立进行、无共享状态或顺序依赖的任务时使用
- **executing-plans**: 当你有一份书面实现计划需要在单独的会话中执行，并设有审查检查点时使用
- **finishing-a-development-branch**: 当实现完成、所有测试通过、需要决定如何集成工作时使用——通过提供合并、PR 或清理等结构化选项来引导开发工作的收尾
- **mcp-builder**: MCP 服务器构建方法论 — 系统化构建生产级 MCP 工具，让 AI 助手连接外部能力
- **receiving-code-review**: 收到代码审查反馈后、实施建议之前使用，尤其当反馈不明确或技术上有疑问时——需要技术严谨性和验证，而非敷衍附和或盲目执行
- **requesting-code-review**: 完成任务、实现重要功能或合并前使用，用于验证工作成果是否符合要求
- **subagent-driven-development**: 当在当前会话中执行包含独立任务的实现计划时使用
- **systematic-debugging**: 遇到任何 bug、测试失败或异常行为时使用，在提出修复方案之前执行
- **test-driven-development**: 在实现任何功能或修复 bug 时使用，在编写实现代码之前
- **using-git-worktrees**: 当需要开始与当前工作区隔离的功能开发，或在执行实现计划之前使用——通过原生工具或 git worktree 回退机制确保隔离工作区存在
- **using-superpowers**: 在开始任何对话时使用——确立如何查找和使用技能，要求在任何响应（包括澄清性问题）之前调用 Skill 工具
- **verification-before-completion**: 在宣称工作完成、已修复或测试通过之前使用，在提交或创建 PR 之前——必须运行验证命令并确认输出后才能声称成功；始终用证据支撑断言
- **workflow-runner**: 在 Claude Code / OpenClaw / Cursor 中直接运行 agency-orchestrator YAML 工作流——无需 API key，使用当前会话的 LLM 作为执行引擎。当用户提供 .yaml 工作流文件或要求多角色协作完成任务时触发。
- **writing-plans**: 当你有规格说明或需求用于多步骤任务时使用，在动手写代码之前
- **writing-skills**: 当创建新技能、编辑现有技能或在部署前验证技能是否有效时使用

## 如何使用

当任务匹配某个 skill 时，使用 `Skill` 工具加载对应 skill 并严格遵循其流程。绝不要用 Read 工具读取 SKILL.md 文件。

如果你认为哪怕只有 1% 的可能性某个 skill 适用于你正在做的事情，你必须调用该 skill 检查。
<!-- superpowers-zh:end -->
