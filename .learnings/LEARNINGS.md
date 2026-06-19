# Learnings

Corrections, insights, and knowledge gaps captured during development.

**Categories**: correction | insight | knowledge_gap | best_practice

---

## [LRN-20260614-001] independent-test-module-internal-import

**Logged**: 2026-06-14T17:00:00+08:00
**Priority**: high
**Status**: resolved
**Area**: tests

### Summary
独立 Go module 的测试包无法 import `internal` 目录下的包，必须通过桥接函数暴露。

### Details
`tests/e2e/` 是独立 Go module（有自己 `go.mod`），通过 `replace` 指向 `../../`。
Go 工具链禁止独立 module import 另一个 module 的 `internal/` 目录。
本项目中 `cmd/server/internal/comic`、`cmd/server/internal/onecomic` 等均不能被 `tests/e2e` 引用。

### Suggested Action
已在 `cmd/server/handler/e2e_storage.go` 创建 `InitE2EStorage()` 桥接函数，导出需要的 MemoryStorage 实例。
**模式**：在 handler 包（`cmd/server/handler/`）中创建 bridge 函数，该包不属于 `internal` 可被外部 import。

### Metadata
- Source: error
- Related Files: tests/e2e/main_test.go, cmd/server/handler/e2e_storage.go

---

## [LRN-20260614-002] go-test-subdirectory-package

**Logged**: 2026-06-14T17:00:00+08:00
**Priority**: high
**Status**: resolved
**Area**: tests

### Summary
Go 测试文件若放在子目录且 `package` 声明与父目录不同，会导致编译错误无法访问父包符号。

### Details
`tests/e2e/tests/*_test.go` 声明 `package main`，但 `tests/e2e/main_test.go` 也是 `package main`。
Go 不允许跨目录的同包名访问未导出符号 —— `newPage`、`testServer` 在子目录中全部 `undefined`。

### Suggested Action
将测试文件直接放在 `tests/e2e/` 目录下，不要用 `tests/e2e/tests/` 子目录。
所有测试文件在同一个 `package main` 中即可正常互访。

### Metadata
- Source: error
- Related Files: tests/e2e/main_test.go, tests/e2e/compare_test.go
- Pattern-Key: tests.subdirectory.package

---
## [LRN-20260614-003] png-extension-mismatch

**Logged**: 2026-06-14T17:00:00+08:00
**Priority**: medium
**Status**: resolved
**Area**: tests

### Summary
使用 `png.Encode` 编码的图片但用了 `.jpg` 扩展名，服务端按扩展名设置 Content-Type 会导致解码失败。

### Details
`tests/e2e/fixtures/seed.go` 中 `generateMockImage` 使用 `image/png` 编码，但文件名是 `1.jpg`、`2.jpg`。
如果服务端用文件扩展名决定 Content-Type，浏览器收到 PNG 字节但标记为 image/jpeg 会解码出错。

### Suggested Action
将文件扩展名改为 `.png`，保持与编码格式一致。

### Metadata
- Source: code_review
- Related Files: tests/e2e/fixtures/seed.go

---
## [LRN-20260614-004] playwright-evaluate-args

**Logged**: 2026-06-14T17:00:00+08:00
**Priority**: high
**Status**: resolved
**Area**: tests

### Summary
Playwright-go v0.5700 的 `Locator.Evaluate()` 需要 `args` 参数（至少传 `nil`），不能省略。

### Details
`page.Locator(selector).Evaluate("js code")` 在旧版可用，但新版签名变为
`Evaluate(expression string, args ...any)`，且要求至少传 `nil` 作为第二个参数。
省略时编译报 `not enough arguments in call`。

### Suggested Action
所有 `Evaluate` 调用改为 `Evaluate("js code", nil)`。

### Metadata
- Source: error
- Related Files: tests/e2e/navigation_test.go, tests/e2e/gallery_detail_test.go

---
## [LRN-20260614-005] env-var-test-leakage

**Logged**: 2026-06-14T17:00:00+08:00
**Priority**: medium
**Status**: resolved
**Area**: tests

### Summary
TestMain 中 `os.Setenv` 修改环境变量后未恢复，可能泄漏影响其他测试。

### Details
`COCOM_STORAGE_GALLERY`、`COCOM_STORAGE_ARCHIVE`、`COCOM_STORAGE_ARCHIVE_TEMP`
被设置为临时目录路径，但只在 `TestMain` 结束前才恢复。
如果测试中途 panic 或跳过 cleanup，环境变量不会被恢复。

### Suggested Action
在设置前保存原值，在 `defer` 中恢复，并在 `TestMain` 末尾也做恢复操作。

### Metadata
- Source: code_review
- Related Files: tests/e2e/main_test.go

---
## [LRN-20260614-006] handler-init-mongo-panic

**Logged**: 2026-06-14T17:00:00+08:00
**Priority**: high
**Status**: resolved
**Area**: tests

### Summary
`handler.Init()` 会调用 `mongowrap.Init()` 尝试连接 MongoDB，在测试环境执行会 panic。

### Details
测试环境不应依赖 MongoDB，但 `handler.Init()` 内部有 `mongowrap.Init()` 调用。
必须跳过 `handler.Init()`，改为手动逐条注册 Gin 路由。

### Suggested Action
使用 `gin.WrapF(handlerFunc)` 逐条注册需要的路由，不调用 `handler.Init()`。
需要先在 handler 包中找到正确的函数签名（如 `CompareComics` 而非 `ComparePost`）。

### Metadata
- Source: error
- Related Files: tests/e2e/main_test.go, cmd/server/handler/init.go

---
## [LRN-20260614-007] testutil-factory-type-mismatch

**Logged**: 2026-06-14T17:00:00+08:00
**Priority**: high
**Status**: resolved
**Area**: tests

### Summary
`testutil.MockComicInfo` 工厂函数和测试数据构造中使用了不存在的类型（`api.ComicTitle`、`api.Page`、`api.ImageUrls`），与实际 `api.ComicInfo` 结构不匹配。

### Details
- `api.ComicTitle` 不存在 —— ComicInfo 的 title 字段是匿名 struct
- `api.Page` 不存在 —— 图片用 `api.PicInfo`
- `MediaID` → 实际是 `MediaId`
- `WithPages` 的 Options 签名与实际不匹配

### Suggested Action
编写测试数据前先阅读 `api.ComicInfo` 的实际 Go 结构定义，不要假设字段名和类型。

### Metadata
- Source: error
- Related Files: cmd/server/internal/testutil/factory.go, pkg/comic/api/type.go

---
## [LRN-20260614-008] subagent-worktree-cleanup

**Logged**: 2026-06-14T17:00:00+08:00
**Priority**: medium
**Status**: resolved
**Area**: infra

### Summary
Subagent 创建的工作树在实现完成后可能残留，需手动清理。

### Details
多个子代理（subagent-driven-development）各自创建隔离 worktree，
技能完成后不会自动清理 worktree 目录和分支。
这些残留 worktree 会在 `git worktree list` 显示，占用磁盘空间。

### Suggested Action
定期执行：
- `git worktree prune` 清理过期注册
- 手动删除 `.claude/worktrees/` 下不再使用的目录
- 删除对应的分支

### Metadata
- Source: error
- Related Files: .claude/worktrees/

---
## [LRN-20260614-009] playwright-test-screenshot-dir

**Logged**: 2026-06-14T17:00:00+08:00
**Priority**: medium
**Status**: resolved
**Area**: tests

### Summary
截图函数写入目录前需要确保目录存在，否则静默失败。

### Details
`TakeScreenshot` 函数接受 `ScreenshotDir` 常量路径但从未创建该目录。
`page.Screenshot()` 在目录不存在时会报错，但该错误仅用 `t.Logf` 记录，
不会传播到测试失败信号中，造成失败截图静默丢失。

### Suggested Action
在 `TakeScreenshot` 中自动创建父目录，或在 TestMain 中预先创建。

### Metadata
- Source: code_review
- Related Files: tests/e2e/helpers/playwright.go

---
## [LRN-20260614-010] gin-WrapF-handler-signature

**Logged**: 2026-06-14T17:00:00+08:00
**Priority**: medium
**Status**: resolved
**Area**: backend

### Summary
`gin.WrapF` 只兼容 `func(http.ResponseWriter, *http.Request)` 签名，
handler 包中所有旧版 net/http handler 都符合此签名可以直接包装。

### Details
handler 包（`cmd/server/handler/`）使用 `func(w http.ResponseWriter, req *http.Request)` 签名。
`gin.WrapF` 可以将这些函数包装为 Gin HandlerFunc。
Gin handler（`func(c *gin.Context)`）则直接用 `r.POST("/path", handlerFunc)` 注册。

### Metadata
- Source: insight
- Related Files: tests/e2e/main_test.go, cmd/server/handler/*.go

---

## [LRN-20260614-011] playwright-windows-cgo

**Logged**: 2026-06-14T17:00:00+08:00
**Priority**: medium
**Status**: pending
**Area**: infra

### Summary
Playwright-go 在 Windows 上需要 CGO_ENABLED=1 以及正确的 Chromium 安装路径。

### Details
`CGO_ENABLED=1` 已在 Makefile 的 `test-e2e` 目标中设置。
`test-e2e-install` 目标使用 `go run` 安装 Chromium，但 POSIX 路径语法在 Windows 上可能不兼容。
Windows 用户可能需手动 `playwright install chromium`。

### Metadata
- Source: code_review
- Related Files: Makefile

---

## [LRN-20260614-012] free-port-race-httptest

**Logged**: 2026-06-14T17:00:00+08:00
**Priority**: low
**Status**: pending
**Area**: tests

### Summary
`httptest.NewServer` 自动选择随机端口，E2E 测试中与 Playwright 集成时，
需要在 `page.Goto` 中使用 `testServer.URL`，不能硬编码端口。

### Details
每个 Gin TestServer 实例的端口不同，`httptest.Server.URL` 提供了完整 URL。
所有测试文件通过全局 `testServer` 变量引用 URL，不需要知道具体端口号。

### Metadata
- Source: insight
- Related Files: tests/e2e/main_test.go

---
## [LRN-20260614-013] v2-api-route-not-registered

**Logged**: 2026-06-14T17:00:00+08:00
**Priority**: medium
**Status**: pending
**Area**: tests

### Summary
E2E 测试未注册 `/v2/api/nhcomic/:cid/archive` 等路由（需要 Service 实例），
因此 gallery_detail_test.go 中 like/archive/restore 的点击验证仅限 DOM 可见性。

### Details
前端 `gallery-actions.js` 中 like/archive/restore 的 XHR 请求走 `/v2/api/nhcomic/` 路由，
由 `pkg/comic.Handler` 注册（需要 `Service` 实例和 `gin.RouterGroup`）。
当前 TestMain 没有创建 Service 实例，这些按钮的点击不会触发真正的 API 调用。

### Metadata
- Source: insight
- Related Files: tests/e2e/main_test.go, cmd/server/view/static/custom/js/modules/gallery-actions.js

---
## [LRN-20260614-014] e2e-test-go-mod-tidy-order

**Logged**: 2026-06-14T17:00:00+08:00
**Priority**: medium
**Status**: resolved
**Area**: tests

### Summary
在 `tests/e2e/` 独立 module 中，必须先 `cd` 到该目录再执行 `go mod tidy`，
且首次添加测试文件之前 `go mod tidy` 可能因无代码而生成不完整的 `go.sum`。

### Details
- 独立 module 需要有自己的 `replace` 指令指向主 module
- 首次 `go mod tidy` 在没有 import 任何包时不会生成有用的 go.sum
- 添加至少一个引用主 module 的测试文件后，再 `go mod tidy` 才能正确填充依赖

### Metadata
- Source: error
- Related Files: tests/e2e/go.mod
- Pattern-Key: tests.independent.module.setup

---

## [LRN-20260615-015] registerAPIRoutes-gin-IRouter-interface

**Logged**: 2026-06-15T03:30:00+08:00
**Priority**: high
**Status**: resolved
**Area**: backend

### Summary
`registerAPIRoutes` 的形参必须使用 `gin.IRouter` 接口类型，不能用 `*gin.RouterGroup` 指针类型，因为 `*gin.Engine` 值嵌入 `RouterGroup` 无法隐式转换为指针。

### Details
`*gin.Engine` 通过值嵌入（value-embedding）包含 `RouterGroup`，不能隐式转换为 `*gin.RouterGroup`。两者都实现 `gin.IRouter` 接口。所以 `registerAPIRoutes(r gin.IRouter)` 可以接受 `*gin.Engine`（生产环境）和 `*gin.RouterGroup`（E2E 环境）。

### Suggested Action
使用 `gin.IRouter` 作为参数类型，不要用 `*gin.Engine` 或 `*gin.RouterGroup`。

### Metadata
- Source: error
- Related Files: cmd/server/handler/init.go, cmd/server/handler/e2e_storage.go
- Pattern-Key: backend.gin.irouter

---

## [LRN-20260615-016] e2e-dialog-accept-correct-value

**Logged**: 2026-06-15T03:30:00+08:00
**Priority**: high
**Status**: resolved
**Area**: tests

### Summary
E2E 测试中处理 `window.prompt()` 对话框时，`dialog.Accept()` 必须传入 JS 预期的有效值，不能随意传字符串。

### Details
`page-manager.js` 的 `openDeleteConfirm()` 使用 `parseInt(input.trim(), 10) === cidNum` 验证用户输入。测试传入 `dialog.Accept("delete")` 永远不会通过验证，删除请求从未发送。改为 `dialog.Accept("3003")`（正确的 CID 数字）。

### Suggested Action
测试 prompt 对话框前先阅读对应 JS 的验证逻辑，传入能通过验证的真实值。

### Metadata
- Source: code_review
- Related Files: tests/e2e/gallery_detail_test.go, cmd/server/view/static/custom/js/modules/page-manager.js

---

## [LRN-20260615-017] e2e-test-assert-direction

**Logged**: 2026-06-15T03:30:00+08:00
**Priority**: medium
**Status**: resolved
**Area**: tests

### Summary
E2E 测试中的断言方向容易写反——检查"不应该是什么"而不是"应该是什么"。

### Details
ZoomReset 测试只检查 `val != 400`（不是预设值），但任何其他值（200、600、甚至失败值 0）都通过。应该检查 `val == 1200`（重置后的预期默认值）或用 `val <= 400` 缩小可接受范围。

教训：硬断言不是越多越好——断言的方向必须验证"应该是什么"，而非仅仅"不是什么"。

### Metadata
- Source: code_review
- Related Files: tests/e2e/gallery_detail_test.go
- Pattern-Key: tests.assert.affirmative

---

## [LRN-20260615-018] subagent-parallel-file-conflict

**Logged**: 2026-06-15T03:30:00+08:00
**Priority**: high
**Status**: resolved
**Area**: tests

### Summary
并行子代理（subagent）不能修改同一个文件——子代理互相不知道对方的存在，同时写入同一文件会导致最后保存的覆盖先前的。

### Details
Phase 2 的 T1/T2/T3/T4 各自修改不同的测试文件（gallery_detail、compare、navigation、quick_action），可以并行。Phase 4 的 P1/P2/P3/P4 也各自修改不同的包目录，可以并行。但如果 P1 修改 `pkg/errwrap` 而其他代理也修改同一个文件，就会冲突。

### Suggested Action
并行分派时严格按文件边界拆分任务：每个代理只写一个或多个无交集的文件。如果需要同一个文件的多个修改，串行执行或在单个代理内完成。

### Metadata
- Source: insight
- Related Files: (general pattern)
- Pattern-Key: tests.parallel.file_boundary

---

## [LRN-20260615-019] mongodb-test-panic-defer-handling

**Logged**: 2026-06-15T03:30:00+08:00
**Priority**: medium
**Status**: resolved
**Area**: tests

### Summary
需要 MongoDB 的包在测试环境中会 panic，不能用 `defer recover()` 代替 mock。

### Details
`handler.Init()` 和 `cache.Init()` 在无 MongoDB 时直接 panic。用 `defer recover()` 只能确保测试不崩溃，但无法验证实际逻辑。真正的解决方案是注入 mock 存储（MemoryStorage）或条件跳过（`testing.Short()` 或环境变量守卫）。

### Suggested Action
为需要 DB 的测试函数提供条件跳过：`testing.Short()` 或在包级 `TestMain` 中检查 MongoDB 可用性，不可用时 `t.Skip()`。

### Metadata
- Source: code_review
- Related Files: cmd/server/internal/cache/cache_test.go, cmd/server/handler/init.go

---

## [LRN-20260615-020] e2e-assertion-soft-vs-hard

**Logged**: 2026-06-15T03:30:00+08:00
**Priority**: high
**Status**: resolved
**Area**: tests

### Summary
E2E 测试中观察性验证（如 "autocomplete dropdown appeared"）用 `t.Log` 合理，但交互式验证（如 "hamburger nav links visible after click"、"Escape exited link mode"）必须用 `t.Error`。

### Details
代码评审中发现多处交互验证使用 `t.Log` 记录观察结果但不使测试失败：Mobile hamburger 导航链接不可见、Escape 未退出模式、Preview 未关闭。这些场景下行为不符合预期本身就是 bug，测试应该报告失败。

### Suggested Action
交互验证用 `t.Errorf` + 分支，行为分支两种都验证：正常路径 log，异常路径 error。

### Metadata
- Source: code_review
- Related Files: tests/e2e/navigation_test.go, tests/e2e/quick_action_test.go, tests/e2e/compare_test.go
- Pattern-Key: tests.assert.interaction

---

## [LRN-20260615-021] remote-git-Bash-tool-path

**Logged**: 2026-06-15T03:30:00+08:00
**Priority**: low
**Status**: resolved
**Area**: infra

### Summary
Bash 工具默认在 `cocom/` 目录下执行，PowerShell 则相反。git add/commit 需要在正确的上下文路径中执行，否则 pathspec 不匹配。

### Details
Bash 工具用 Unix 路径 `/d/workdir/...`，PowerShell 用 Windows 路径 `D:\workdir\...`。`cd cocom` 在 Bash 中失败因为当前目录已经是 `cocom/`：`/d/workdir/leon/cocomhub/cocom`。在 PowerShell 中需要 `Set-Location D:\workdir\leon\cocomhub\cocom`。

### Suggested Action
在所有 git 操作前显式 `cd` 到目标目录或使用 PowerShell 的 `Set-Location`。使用 `git -C /d/workdir/leon/cocomhub/cocom add` 形式可避免路径问题。

### Metadata
- Source: error
- Related Files: (shell environment)

---

## [LRN-20260615-022] best_practice

**Logged**: 2026-06-15T11:30:00+08:00
**Priority**: high
**Status**: pending
**Area**: tests

### Summary
E2E 测试 t.Skip 替代方案：zoom sidebar 需先进大图模式；card count 用 `WaitForCardCount` 轮询替代条件跳过。

### Details
- Zoom sidebar 的 `display:none` 需要先调用 `toggleLargeMode()`（通过 `helpers.EnterLargeMode`）才能激活。`helpers.EnterLargeMode` 会点击大图模式按钮并等待 zoom sidebar 可见。
- 替代注入 `__E2E_FORCE_ZOOM_SIDEBAR__` 标志的方案，`EnterLargeMode` 更贴近用户真实操作路径。
- 使用 `helpers.WaitForCardCount(t, page, helpers.GalleryCard, 2)` 替代 count < 2 的 `t.Skip`。该函数用 requestAnimationFrame 轮询 DOM，直到卡片数达到阈值。
- PowerShell 的 `.Replace()` 在替换跨行 Go 代码块时，缩进和换行符容易出错。使用 sed 或 Go 脚本更可靠。

### Metadata
- Source: conversation
- Related Files: tests/e2e/gallery_detail_test.go, tests/e2e/quick_action_test.go, tests/e2e/helpers/playwright.go
- Tags: e2e, t.Skip, zoom-sidebar, gallery-cards

---

## [LRN-20260615-023] insight

**Logged**: 2026-06-15T11:30:00+08:00
**Priority**: high
**Status**: resolved
**Area**: tests

### Summary
go vet 必须作为从 worktree 拷贝代码后的验证步骤，多处子代理生成的测试代码存在编译问题不通过 vet。

### Details
- `recommend_test.go` 中 Gin handler 测试必须用 `gin.CreateTestContext`，不能用 `httptest.NewRecorder` 直接调函数
- `random_gallery_test.go` 中 `resp.Status` 是方法不是字段，必须用 `resp.Status()`
- PowerShell Replace 处理缩进层数时，输入的 3-tab 和 2-tab 混合导致多出一个右花括号

### Metadata
- Source: conversation
- Related Files: cmd/server/handler/recommend_test.go, tests/e2e/random_gallery_test.go, tests/e2e/gallery_detail_test.go
- Tags: vet, worktree, code-quality

---

## [LRN-20260615-024] correction

**Logged**: 2026-06-15T16:00:00+08:00
**Priority**: high
**Status**: resolved
**Area**: infra

### Summary
git commit message 中出现大量 shell 错误输出，因为多行 message 在 Bash 工具中被 shell 解释执行，非纯文本传递。

### Details
多行 commit message 中包含反引号 `` ` ``、`$` 等 shell 特殊字符时，Bash 工具会尝试解析执行。例如 message 中出现 `t.Skip("requires wget")` 被 shell 当作命令执行。正确做法是：
1. 使用 `git commit -m "..."` 时避免 message 中含 shell 特殊字符
2. 或使用 `git commit -F message.txt` 从文件读取（彻底避免 shell 解释）
3. 或使用 heredoc：`git commit <<'EOF'`（单引号 EOF 禁止变量展开）

### Suggested Action
含代码引用或多行文本的 commit message 使用 `git commit -F /tmp/msg` 写文件再提交。

### Metadata
- Source: error
- Related Files: (none)
- Tags: git, commit, shell

---

## [LRN-20260615-025] best_practice

**Logged**: 2026-06-15T16:00:00+08:00
**Priority**: high
**Status**: pending
**Area**: tests

### Summary
code-review skill 的 3 个 finder angle（line-by-line scan, removed-behavior auditor, cross-file tracer）能有效发现人工审查容易遗漏的隐藏缺陷。

### Details
本次 code-review 通过 3 个并行 finder agent 发现：
- `search_test.go` 的 `init()` 中 `cache.Init()` 在 Viper 配置就绪前执行（可能导致 panic）
- `verify_test.go` 的 `TestNewComicVerifier` 是空函数体（`_ = NewMemoryStorage`），永远 pass 无意义
- `helpers/playwright.go` 的 `WaitForCardCount` JS Promise 无 timeout，无限轮询
- `tags_agg_test.go` 的 `defer recover()` 只 `t.Logf` 不 `t.Error`，测试永远不会 fail

### Metadata
- Source: conversation
- Related Files: cmd/server/handler/search_test.go, pkg/comic/verify_test.go, tests/e2e/helpers/playwright.go, cmd/server/handler/tags_agg_test.go
- Tags: code-review, testing, quality

---

## [LRN-20260615-026] best_practice

**Logged**: 2026-06-15T16:00:00+08:00
**Priority**: medium
**Status**: pending
**Area**: tests

### Summary
tags_agg_test.go 中的所有测试使用 `defer recover()` + `t.Logf` 组合，不会 panic 也不会 fail——它们只是"不崩溃"检查，没有实际断言价值。

### Details
10 个测试函数（`TestAggregateTags_ReturnsOK`、`TestGetTags_*`、`TestSearchTags_*`）都使用相同的模式：
```go
defer func() {
    if r := recover(); r != nil {
        t.Logf("... panicked: %v", r)
    }
}()
// 调用 handler，decode 响应，然后 t.Logf 输出结果
// 没有 t.Error/t.Fatal
```
无论返回什么响应码，测试都 pass。只在 panic 时 pass（被 recover 吞掉）。应该至少验证响应可 decode、返回非 500 状态码等最基本的健康检查。

### Suggested Action
为这些测试添加至少一个断言（如 `resp.Head.Code != 0 || w.Code != http.StatusInternalServerError`），或者在有 MemoryStorage 支持后改为真实路径测试。

### Metadata
- Source: code-review
- Related Files: cmd/server/handler/tags_agg_test.go
- Tags: testing, assertion, quality
---


## [LRN-20260616-027] golangci-lint-v2-config-migration

**Logged**: 2026-06-16T07:10:00+08:00
**Priority**: high
**Status**: resolved
**Area**: infra

### Summary
golangci-lint v2 配置文件格式重大变更：v1 的 `linters-settings` → `linters.settings`，`gosimple` 和 `typecheck` 已移除，必须添加 `version: "2"`。

### Details
CI 自动安装 `golangci-lint v2.12.2` 后，旧版 `.golangci.yml` 报错 `unsupported version of the configuration: ""`。

关键变更：
1. **必须添加** `version: "2"` 顶层字段
2. **`gosimple` 已移除** — 合并到 staticcheck，不要再 enable
3. **`typecheck` 已移除** — 始终启用，不能显式声明
4. **`linters-settings` → `linters.settings`** — settings 移入 linters 下
5. **`issues.exclude-rules` → `linters.exclusions.rules`** — exclude 规则移到 linters 内部
6. **`run.timeout` → 顶层 `timeout`** — 不再嵌套在 run 下
7. **`gofmt` 从 `linters.enable` → `formatters.enable`** — formatter 和 linter 分离
8. **`run:` 段大幅精简** — go 版本声明保留在 `run.go`，但 timeout 等移出

### Suggested Action
新项目直接使用 v2 格式模板。迁移旧项目时：
- `golangci-lint config verify` 可检查配置合法性
- v2 的错误提示很明确（unknown linter/unsupported field），按报错逐条修改即可

### Metadata
- Source: error
- Related Files: .golangci.yml
- Tags: golangci-lint, v2, migration, config

---

## [LRN-20260616-028] shadow-rename-t-Fatal-err-漏改

**Logged**: 2026-06-16T07:30:00+08:00
**Priority**: high
**Status**: resolved
**Area**: tests

### Summary
重命名 govet shadow 变量时，`t.Fatal(err)` 和 `return nil, err` 中的 `err` 容易被遗漏。AI 代理在自动重命名 50+ 处 shadow 时漏掉了约 6 处。

### Details
当把 `if err := fn(); err != nil { t.Fatal(err) }` 改为 `if xxxErr := fn(); xxxErr != nil { ... }` 后，`t.Fatal(err)` 或 `return nil, err` 中的 `err` 仍然引用外层的旧 err 变量。这不会造成 go vet 或编译错误（因为外层 err 仍然存在），但会导致错误的错误对象被使用。

本会话中漏掉的 3 处：
- `pkg/comic/storage_test.go:159,201` — `t.Fatal(err)` 仍引用外层 err
- `pkg/imaging/image.go:75` — `SetIErr(err)` 仍引用外层 err（之前已修）
- `pkg/archive/manager/executor.go:43` — `return nil, err` 仍引用外层 err（之前已修）

### Suggested Action
批量重命名 shadow 变量后，必须用 `grep -n 'SetIErr(err)\|Fatal(err)'` 搜索所有旧引用是否已更新为新的变量名。

### Metadata
- Source: correction
- Related Files: pkg/comic/storage_test.go, pkg/imaging/image.go, pkg/archive/manager/executor.go
- Tags: lint, shadow, variable-rename, code-quality
- Pattern-Key: lint.shadow.rename.verify

---

## [LRN-20260616-029] defer-close-errcheck-wrapper-pattern

**Logged**: 2026-06-16T07:30:00+08:00
**Priority**: medium
**Status**: resolved
**Area**: backend

### Summary
修复 defer Close/resp.Body.Close 的 errcheck 时，`defer func() { _ = x.Close() }()` 比 `//nolint:errcheck` 更正式，但增加代码长度。`exclude-functions` 配置项对 defer 场景不生效。

### Details
在 golangci-lint v2 中，`errcheck.exclude-functions` 可以排除大多数 errcheck 误报，但对 `defer cursor.Close(ctx)`（带参数的 Close）和 `defer rc.Close()`（接口类型）不生效。解决方案：
- **优先**：`defer func() { _ = x.Close() }()` 包裹
- **备选**：`defer x.Close() //nolint:errcheck` 注释
- **不推荐**：`exclude-functions` 配置，因为 interface 类型的 Close 匹配规则复杂

对 `cursor.Close(ctx)` 必须用 `defer func() { _ = cursor.Close(ctx) }()` 因为 `defer` 后不能跟赋值表达式。

### Metadata
- Source: insight
- Related Files: pkg/storage/localfs/localfs.go, pkg/archive/manager/filestore.go
- Tags: lint, errcheck, defer, pattern

---

## [LRN-20260616-030] subagent-worktree-copy-verification-gap

**Logged**: 2026-06-16T07:30:00+08:00
**Priority**: high
**Status**: resolved
**Area**: infra

### Summary
从子代理 worktree 拷贝代码回主分支后，副本可能存在编译/逻辑缺陷，需运行 vet + lint + 测试验证，不能仅靠 build 检查。

### Details
四个并行子代理在独立 worktree 中修复 lint 问题后，拷贝回主 repo 的文件存在以下缺陷：
1. **shadow 变量引用漏改** — `return nil, err` 未更新为新变量名（executor.go, filestore.go, checker.go, image.go）
2. **`.golangci.yml` 被组 D 覆盖** — 回退到旧版配置（因为某些 worktree 基于 65bba2b 而非最新）
3. **`settings.go` 类型断言修改** — `key.(string)` 改为 `k, _ := key.(string)` 后 settings 赋值逻辑变化

### Suggested Action
从 worktree 拷贝代码后的标准化验证流程：
1. `go vet ./...` — 捕获编译/赋值问题
2. `golangci-lint run ./...` — 捕获 lint 回归
3. `go build ./...` — 确认可编译
4. 回归测试受影响的包
5. 检查 .golangci.yml 等非 .go 文件是否被误覆盖

### Metadata
- Source: error
- Related Files: (general pattern)
- Tags: worktree, subagent, code-quality, verification
- Pattern-Key: infra.worktree.copy.verify

---

## [LRN-20260616-031] golangci-lint-v2-gosimple-typecheck-removed

**Logged**: 2026-06-16T07:30:00+08:00
**Priority**: high
**Status**: resolved
**Area**: infra

### Summary
golangci-lint v2 移除了 `gosimple` 和 `typecheck` 两个 linter，启用它们会导致配置报错。

### Details
在 v2 中：
- `gosimple` 的功能已合并到 `staticcheck`，从 linter 列表中移除
- `typecheck` 始终启用，不能显式声明
- 如果在 `linters.enable` 中包含它们，`golangci-lint config verify` 会报：
  - `typecheck is not a linter, it cannot be enabled or disabled`
  - `unknown linters: 'gosimple'`

原先 v1 配置中 `enable: [errcheck, gosimple, govet, ineffassign, staticcheck, typecheck, unused, gofmt]` 需要改为 `enable: [errcheck, govet, ineffassign, staticcheck, unused]` 并且 `gofmt` 移到 `formatters.enable`。

### Metadata
- Source: error
- Related Files: .golangci.yml
- Tags: golangci-lint, v2, migration
- See Also: LRN-20260616-027

---

## [LRN-20260616-032] makefile-three-section-structure

**Logged**: 2026-06-16T21:45:00+08:00
**Priority**: high
**Status**: pending
**Area**: infra

### Summary
跨三个子项目统一 Makefile 结构时，使用 STANDARD / CUSTOM / OTHER 三段式拆分受控、自定义和遗留内容，成功实现了规范化与兼容性的平衡。

### Details
设计模式要点：
1. **STANDARD** — 所有项目必须实现的统一目标和变量（`prepare`、`build-ci`、`test-ci`、`check-ci`、`RAW_GO`、`GOOS`/`GOARCH`）。CI 直接引用 STANDARD 目标，不写裸 go 命令。
2. **CUSTOM** — 项目特有目标和变量覆盖（`GOTAGS`、`COVER_THRESHOLD`、`build-sub-tools`、`test-no-mongo`）。各项目按需配置，不改动 STANDARD。
3. **OTHER** — 保留的原有变量和调试辅助目标（`ALL_SRC`、`echo-all-pkgs`、`fmt-web`）。不参与标准化流水线，仅向后兼容。

关键变量设计：
- `RAW_GO ?= go` + `GO := GOOS=$(GOOS) GOARCH=$(GOARCH) $(RAW_GO)` — 分离原始 go 命令和带交叉编译参数的 go，避免自引用
- `SUB_MODULE_DIRS := $(shell find ...)` — 自动发现子 module，空时循环自动跳过
- `NOTEST_IGNORE := .notestignore` — 配置免检包列表文件

### Suggested Action
新项目可复用此三段式模板。关键是 CUSTOM 块允许每个项目在不修改 STANDARD 的情况下覆盖参数（`?=` 延迟展开实现）。

### Metadata
- Source: insight
- Related Files: cocom/Makefile, download-manager/Makefile, sproxy/Makefile
- Tags: makefile, standardization, structure, pattern
- Pattern-Key: infra.makefile.three_section

---

## [LRN-20260616-033] check-loopback-false-positive-production-config

**Logged**: 2026-06-16T21:45:00+08:00
**Priority**: medium
**Status**: pending
**Area**: infra

### Summary
`check-loopback` 目标扫描 0.0.0.0 时，生产配置代码中的合法默认值（如 `viper.SetDefault("server.host", "0.0.0.0")`）会产生误报，必须在 grep 中排除。

### Details
cocom 的 `internal/config/config.go` 中 `viper.SetDefault("server.host", "0.0.0.0")` 是合法的生产配置默认值。`check-loopback` 的 grep 模式会把它当作不安全地址误报。
- cocom：排除 `internal/config/`
- sproxy：排除 `pkg/server/config.go`
- download-manager：排除 `testdata/`、`fixtures/`

### Suggested Action
每个项目的 `check-loopback` 实现应包含已知合法引用的排除规则。如果是标准检查，使用 `--include='*_test.go'` 只检查测试文件可避免大多数误报。

### Metadata
- Source: error
- Related Files: cocom/internal/config/config.go, sproxy/pkg/server/config.go
- Tags: check-loopback, grep, false-positive

---

## [LRN-20260616-034] notest-dot-notestignore-pattern

**Logged**: 2026-06-16T21:45:00+08:00
**Priority**: medium
**Status**: pending
**Area**: tests

### Summary
用 `.notestignore`（gitignore 风格）替代硬编码的 `ALLOWLIST` 数组，实现了更灵活、可维护的"包免测"管理。

### Details
- 旧方案：cocom 的 `check-test-files.sh` 中硬编码 `ALLOWLIST` 数组，修改需要改脚本
- 新方案：`.notestignore` 文件，每一行是一个包路径或 glob 模式，支持 `#` 注释和空行
- 脚本用 `find -not -path "./$line"` 过滤，路径自动适配子 module
- 三个项目各有一个 `.notestignore`，无此文件时等同于无免检

### Suggested Action
新增包但没时间写测试时，在 `.notestignore` 中添加一行即可。后续补上测试后从文件中移除。

### Metadata
- Source: insight
- Related Files: cocom/.notestignore, download-manager/.notestignore, sproxy/.notestignore, cocom/scripts/check-test-files.sh
- Tags: notest, test-coverage, ignore-file, pattern

---

## [LRN-20260616-035] mingw32-make-grep-find-availability

**Logged**: 2026-06-16T21:45:00+08:00
**Priority**: medium
**Status**: pending
**Area**: infra

### Summary
Windows 的 mingw32-make 环境中 grep/find 等命令可能不可用或位置不同，导致 Makefile 中的 shell 命令执行失败。

### Details
Windows 上的 `mingw32-make` 运行在 Git Bash 环境中，但 `grep` 可能不在 `PATH` 中。PowerShell 调用 `make` 时，`FIND` 命令的默认行为与 `grep` 冲突（Windows 的 `find` 和 Git Bash 的 `grep` 命名差异）。

`make -n` 语法验证可以跳过此问题（不实际执行命令），但 `make check-loopback` 等实际执行时会因为 `grep` 不可用而失败。

### Suggested Action
在 Makefile 中添加 `#` 开头的 shell 兼容性检查，或使用 `go run ./tools/...` 替代 shell grep。
CI 中 Windows runner 通过 `choco install make -y --no-progress` 安装 make，且 GitHub Actions 的 windows-latest 有完整的 Git Bash。

### Metadata
- Source: error
- Related Files: (general pattern)
- Tags: windows, make, grep, compatibility

---

## [LRN-20260616-036] go-1.26-coverprofile-mkdir-required

**Logged**: 2026-06-16T21:45:00+08:00
**Priority**: high
**Status**: pending
**Area**: infra

### Summary
即使 Go 1.26，`go test -coverprofile build/cover.out` 也不会自动创建 `build/` 目录。必须确保测试前 `mkdir -p build` 已执行。

### Details
这是本次 CI 故障的直接根因：`go test -coverprofile build/cover.out ./...` 因 `build/` 目录不存在而失败（`open build/cover.out: no such file or directory`）。
- 本地 `make test` 隐含 `prepare` → `mkdir -p build`
- CI 中裸 `go test` 没有这个步骤
- 修复方案：所有测试路径统一走 `make test-cover`（依赖 `prepare` → `mkdir -p build`）

### Suggested Action
永远不要裸写 `go test -coverprofile build/...`，必须确保 `build/` 目录已存在。在 CI 中通过 `make prepare` 或 `make test-cover` 隐式创建。

### Metadata
- Source: error
- Related Files: cocom/.github/workflows/test.yaml, cocom/Makefile
- Tags: go, coverprofile, build-dir, ci
- Pattern-Key: infra.go.coverprofile.mkdir

---

## [LRN-20260617-037] embed-file-must-exist-at-compile-time

**Logged**: 2026-06-17T18:40:00+08:00
**Priority**: high
**Status**: resolved
**Area**: infra

### Summary
`//go:embed` 文件在编译时必须存在，即使变量未被使用。Makefile 中 clean repo 时删除 dirty_info.txt 导致 CI 编译失败。

### Details
`pkg/version/version.go:45` 使用 `//go:embed build/dirty_info.txt` 嵌入 Git diff。Makefile `prepare` 目标在 repo clean 时执行 `rm -f` 删除该文件，但 Go 编译器遇到 `//go:embed` 指向不存在的文件会直接报错（即使 `DirtyInfo` 变量未被引用）。

修复方案：clean repo 时写入空文件（`printf '' >`）而非删除。

### Metadata
- Source: error
- Related Files: cocom/Makefile, cocom/pkg/version/version.go
- Tags: go, embed, makefile, ci
- Pattern-Key: infra.ci.embed-missing-file

---

## [LRN-20260617-038] make-OS-env-not-builtin

**Logged**: 2026-06-17T18:40:00+08:00
**Priority**: medium
**Status**: resolved
**Area**: infra

### Summary
`$(OS)` 在 GNU Make 中**不是**内置变量。它是 Windows 环境变量（`OS=Windows_NT`），Linux/macOS 默认不设置此环境变量。

### Details
`install-webp-tools` 目标用 `ifeq ($(OS),Linux)` 检测系统，但 CI 的 Ubuntu 运行器上 `$(OS)` 为空字符串，导致被 `else` 分支"不支持的平台"截停。

修复方案：改用 `UNAME_OS := $(shell uname -s)` 在变量定义区检测，Linux → `"Linux"`，macOS → `"Darwin"`，精确匹配。

### Suggested Action
在 Makefile 中检测操作系统优先用 `$(shell uname -s)` 而非 `$(OS)`，后者只存在于 Windows 环境中。

### Metadata
- Source: error
- Related Files: cocom/Makefile
- Tags: make, os-detection, ci
- Pattern-Key: infra.make.os-detection

---

## [LRN-20260617-039] ci-all-buildengine-tests-need-mongodb-skip

**Logged**: 2026-06-17T18:40:00+08:00
**Priority**: high
**Status**: resolved
**Area**: tests

### Summary
`cmd/server/` 下所有调用 `BuildEngine(ctx, cfg, shutdownCh)` 的测试都通过 `handler.Init()` → `mongowrap.Init()` → `mongo.Connect()` 间接依赖 MongoDB，CI 中必须跳过。

### Details
最初只给 `graceful_run_test.go` 加了 `t.Skip`，但 CI 跑起来后发现还有 7 个测试同样依赖。这是因为 `BuildEngine` 在第 56 行无条件调用 `handler.Init(ctx, r)`，而 `handler.Init` 在第 21-22 行调用 `mongowrap.Init()` 失败后直接 `panic`。

受影响测试清单：
- `graceful_run_test.go` — TestHTTPStartAndGracefulShutdown
- `middleware_test.go` — TestCORSAndGzip, TestMaxBodySize, TestRateLimit
- `pprof_test.go` — TestPprofLocalAndRemote
- `server_test.go` — TestHealthzReadyz, TestAdminCronShowsArchiveStatusCheckerAndCanRun, TestAdminShutdownIsIdempotentAndReturnsValidStatus
- `settings_integration_test.go` — TestSettingsV1AndAlias

修复方案：创建 `mongo_skip_test.go` 提供共享 `skipIfNoMongo(t)` 函数，所有受影响测试在函数入口处调用 `skipIfNoMongo(t)`。MongoDB 不可用时 `t.Skip()` 优雅跳过，5s 超时后可继续运行包内其他测试。

### Suggested Action
新增 API handler 测试时，如果测试调用 BuildEngine → handler.Init，务必在 test 函数开头加 `skipIfNoMongo(t)`。如果测试仅测试 handler 函数而不依赖 BuildEngine，则不需要（handler 子包的测试使用 TestMain 中注入的 MemoryStorage，不依赖 MongoDB）。

### Metadata
- Source: error
- Related Files: cocom/cmd/server/mongo_skip_test.go, cocom/cmd/server/server.go:56, cocom/cmd/server/handler/init.go:21-22
- Tags: tests, mongodb, ci, skip
- Pattern-Key: tests.ci.mongo-skip-pattern

---

## [LRN-20260617-040] test-data-race-on-shared-slice-in-closure

**Logged**: 2026-06-17T18:40:00+08:00
**Priority**: medium
**Status**: resolved
**Area**: tests

### Summary
并发测试中通过闭包捕获的 `[]string` slice 在多个 goroutine 中同时 `append` 写入会导致 data race，`go test -race` 必定检测到。

### Details
`executor.go` 中的 `executeArchiveStatusCheckIssues` 使用 `sync.WaitGroup.Go` 并发处理 issues。测试的 hook 函数通过闭包捕获 `calls []string` 变量，在 replicate 和 check hook 中 `append` 写入，但缺少互斥锁保护。`go test -race -count=5` 可稳定复现。

修复方案：添加 `sync.Mutex` 保护 `calls` 的写入操作，同时补上 import 中缺失的 `"sync"` 包。

### Suggested Action
编写并发 goroutine 测试时，所有共享变量的写入必须通过 `sync.Mutex` 或原子操作保护。测试 hook 函数中的闭包变量尤其容易遗漏。

### Metadata
- Source: error
- Related Files: cocom/cmd/server/internal/scheduler/archive_status_check_test.go, cocom/cmd/server/internal/scheduler/archive_status_check.go
- Tags: tests, concurrency, data-race
- Pattern-Key: tests.concurrency.slice-closure-mutex

---

## [LRN-20260617-041] tempReadCloser-remove-field-default-false

**Logged**: 2026-06-17T18:40:00+08:00
**Priority**: low
**Status**: resolved
**Area**: tests

### Summary
`tempReadCloser` 的 `remove` 字段默认值 `false`，测试中未显式设置导致 Close() 不删除临时文件。

### Details
`baidupcs.go:538` 定义 `tempReadCloser{remove bool}`，`Close()` 在 `r.remove &&` 为 true 时才执行 `os.Remove(r.path)`。测试 `TestTempReadCloserClose` 创建文件后调用 Close() 并验证文件被删除，但 `rc := &tempReadCloser{ReadCloser: fd, path: tmp}` 未设置 `remove: true`，导致文件残留断言失败。

生产代码 `baidupcs.go:226` 正确设置 `remove: filePath != getOpts.TrySaveFilePath`，所以线上无此问题。

### Suggested Action
使用零值敏感的 bool 字段时，注意默认值是否符合测试意图。生产环境给所有字段明确赋值，不要在调用处依赖零值。

### Metadata
- Source: error
- Related Files: cocom/pkg/storage/baidupcs/baidupcs.go:226,538, cocom/pkg/storage/baidupcs/baidupcs_test.go
- Tags: tests, baidupcs, tempfile
- Pattern-Key: tests.missing-field-init

---

## [LRN-20260617-042] viper-dual-registration-migration

**Logged**: 2026-06-17T22:00:00+08:00
**Priority**: high
**Status**: resolved
**Area**: config

### Summary
Manager 持有实例化 `*viper.Viper` 后，旧代码仍依赖全局 `viper` 单例。需要 `setDefaultsOn(globalViper)` 双注册，直至全量迁移完成。

### Details
`config.New()` 创建 `Manager{v: viper.New()}`，Manager 的 Get() 使用自身 v 而非全局 viper。但 `GetArchivePassword()`、`GetArchiveCmd()` 等旧辅助函数仍然使用 `viper.GetString()`（全局单例）。

迁移期需要 `init()` 中调 `global.setDefaultsOn(viper.GetViper())`，把默认值同时注册到 Manager 和全局 viper。

### Suggested Action
新引入 Manager 时，已存在的全局 viper 依赖需要用 `setDefaultsOn(globalViper)` 同步默认值。迁移完所有 `viper.Get*` 后可以删除双注册。

### Metadata
- Source: insight
- Related Files: internal/config/config.go, internal/config/manager.go
- Tags: viper, migration, dual-registration

---

## [LRN-20260617-043] viper-test-global-state-cleanup

**Logged**: 2026-06-17T22:00:00+08:00
**Priority**: high
**Status**: resolved
**Area**: tests

### Summary
使用全局 viper 的测试在 `-shuffle=on` 下被前序测试通过 `viper.Set()` 修改的值污染，必须 `viper.Reset()` + `Init()` 确保干净。

### Details
`TestBackwardCompatKeys` 和 `TestDeprecatedKeyFallback` 使用全局 viper。当它们在其他使用 `mgr := New()`（manager 局部 viper）的测试之后运行时，前序测试可能通过 `viper.Set("cocom.archive.password", "legacy_val")` 污染了全局 viper。

即使 `mgr := New()` 不碰全局 viper，`config.go` 的 `init()` 只在包加载时运行一次。如果前序测试直接调了 `viper.Set()`，全局 viper 的默认值被覆盖后不会自动恢复。

### Suggested Action
使用全局 viper 的测试必须 `viper.Reset()` + `config.Init()` 开头。`config.Init()` 内部应同时同步 `setDefaultsOn(globalViper)`。新测试优先用 `mgr := New()` 或 `Parse(v)`。

### Metadata
- Source: error
- Related Files: internal/config/config_test.go, internal/config/config_keys_test.go
- Tags: viper, testing, global-state, shuffle

---

## [LRN-20260617-044] pkg-config-struct-self-defined

**Logged**: 2026-06-17T22:00:00+08:00
**Priority**: high
**Status**: resolved
**Area**: config

### Summary
pkg 层定义自身配置结构体（带 `mapstructure/json/yaml` 标签），`internal/config/types.go` 引用组合，pkg 内函数引用自身配置时直接使用类型名无需包前缀。

### Details
```go
// pkg/mongowrap/config.go — 定义自身配置
type Config struct {
    User       string `mapstructure:"user" json:"user" yaml:"user"`
    ...
}
// internal/config/types.go — 引用组合（带包前缀）
type Config struct {
    Mongo   mongowrap.Config   `mapstructure:"mongo"`
    ...
}
// pkg/mongowrap/mongo.go — 包内使用自身 Config（不用包前缀）
func Init(cfg Config) error { ... }
```

依赖关系：`pkg ← config ← cmd`。pkg 不 import config 包，只接收参数。

### Metadata
- Source: insight
- Related Files: pkg/mongowrap/config.go, pkg/download/config.go, internal/config/types.go
- Tags: config, architecture, dependency

---

## [LRN-20260617-045] mongowrap-init-param-injection

**Logged**: 2026-06-17T22:00:00+08:00
**Priority**: high
**Status**: resolved
**Area**: config

### Summary
`mongowrap.Init()` 改为 `Init(cfg Config)` 后，`Client()` 也需要存储初始配置。`sync.Once` 的闭包捕获必须在 Do 之前保存参数。

### Details
```go
// 错误：once.Do 内无法访问 Init(cfg) 的 cfg 参数
func Init(cfg Config) error {
    onceInit.Do(func() { initEngine(cfg) }) // cfg 在闭包外捕获
}
// 错误：Client() 调用 Init() 不知道传什么
func Client() (*mongo.Client, error) {
    if err := Init(cfg); err != nil { // cfg 未定义
```

需在全局变量中加入 `initCfg Config`，`Init(cfg)` 时保存，`Client()` 时传入已保存的值。

### Suggested Action
包级 `sync.Once` 配合参数注入时，用全局变量暂存配置：`initCfg Config` → `Init(cfg) { initCfg=cfg; onceInit.Do(func(){ initEngine(initCfg) }) }`。

### Metadata
- Source: error
- Related Files: pkg/mongowrap/mongo.go
- Tags: mongowrap, sync.Once, param-injection

---

## [LRN-20260617-046] config-test-yaml-build-in-manager

**Logged**: 2026-06-17T22:00:00+08:00
**Priority**: medium
**Status**: resolved
**Area**: tests

### Summary
`TestOverride_YAMLFile` 和 `TestOverride_YAMLPlusEnv` 测试 YAML 文件覆盖，改为 Manager 实例后需用 `mgr.Viper().SetConfigFile()/.ReadInConfig()` 而非全局 `viper.SetConfigFile()`，确保测试间隔离。

### Details
YAML 测试创建临时文件 → 写入 YAML 内容 → 调用 `mgr.Viper().ReadInConfig()` → `mgr.Reset()` → `mgr.Get()` 验证覆盖生效。不能用 `viper.ReadInConfig()` 污染全局状态。

### Metadata
- Source: insight
- Related Files: internal/config/config_keys_test.go
- Tags: config, testing, yaml, manager

---

## [LRN-20260617-047] scheduler-config-mixed-migration

**Logged**: 2026-06-17T22:00:00+08:00
**Priority**: medium
**Status**: pending
**Area**: config

### Summary
scheduler 的三个注册函数配置读取方式不一致：`probe_comic.go` 和 `scheduler.go` 已迁移到 `config.Get()`，`cocoma_archiver.go` 和 `archive_status_check.go` 仍部分使用 `viper.Get*`。

### Details
- `probe_comic.go` — 从 `config.Get().Server.Scheduler.ProbeComic` 读取（已迁移）
- `cocoma_archiver.go` — 混合使用 `config.Get()` 和 `viper.Get*`（部分迁移）
- `archive_status_check.go` — 完全使用 `viper.Get*`（未迁移）
- `scheduler.go` — `New()` 中的 timezone（已迁移）

### Suggested Action
`cocoma_archiver.go` 和 `archive_status_check.go` 全量迁移到 `config.Get()`。

### Metadata
- Source: insight
- Related Files: cmd/server/internal/scheduler/*.go
- Tags: scheduler, config, migration

---

## [LRN-20260620-048] best_practice

**Logged**: 2026-06-20T00:30:00+08:00
**Priority**: high
**Status**: pending
**Area**: tests

### Summary
MongoDB 测试依赖消除的标准模式：SetDefaultStore + MemoryStore + TestMain 统一注入。

### Details
分三层策略：
1. **简单 CRUD 包**（video, custom, onecomic）：定义 Store 接口 + MemoryStore，在业务函数入口加 `if s := defaultStore; s != nil { return }` 守卫，MongoDB 路径作为 fallback
2. **复杂聚合包**（tag, comic）：TagStore 接口含 10+ 方法，MemoryTagStore 用 `map[string][]*ComicTagDoc` 模拟 comicTag 集合，`GetDefaultComicStore()`（comic.Storage）做漫画聚合
3. **CLI 命令**（cmd/ar, cmd/verify）：不能用 SetDefault 方式（无持久化结构体），用包级函数变量 `var GetSourceDir func`，`init()` 赋 MongoDB 实现

关键要素：
- 每个包必须提供 `ResetDefaultXxxStore()` 函数，保证测试隔离
- TestMain 统一注入所有层 store（`handler_test.go` 中已实现）
- 归档层（`pkg/archive/manager`）已有 `MemoryIndexStore`，直接可用
- `errors.go` 中的 `IsNotFound` 不要引用 `mongo.ErrNoDocuments`，否则即使内存存储也带 MongoDB driver 依赖

### Suggested Action
新增包需要测试 Mock 时，先看是否有可用 MemoryStore。若无，按上述分层选择合适模式。

### Metadata
- Source: insight
- Related Files: cmd/server/internal/{video,custom,onecomic,tag}/default_store.go, cmd/ar/ar.go, cmd/verify/verify.go
- Tags: mongodb, testing, memory-store, pattern
- Pattern-Key: tests.mongodb.replace.pattern

---
## [LRN-20260620-049] best_practice

**Logged**: 2026-06-20T00:35:00+08:00
**Priority**: high
**Status**: pending
**Area**: tests

### Summary
替换生产关系组（tag/relation.go）的 MongoDB 内存实现时，返回的 `TagRelationDoc` 必须填充 Tags 字段，不能返回 nil，否则 handler 层访问 Tags 时 panic。

### Details
`relation.go:CreateRelation` 在 `GetDefaultRelationStore()` 非 nil 路径中只调用了 `s.CreateRelation` 但丢弃了返回值，返回 `&TagRelationDoc{ID: ..., Tags: nil}`。handler 层 `CreateRelationHandler` 在第 49 行用 `doc.Tags` 遍历 `for i, t := range doc.Tags`，Tags 为 nil 时 Go 的 range 不会 panic 但会得到 0 次迭代，前端永远看不到创建的 tags。

修复：在 store 路径中也构建 `tagBriefDocs` 填充 Tags 字段，使行为与 MongoDB 路径一致。

### Metadata
- Source: code-review
- Related Files: cmd/server/internal/tag/relation.go
- Tags: mongodb, memory-store, tag-relation, nil-pointer

---
## [LRN-20260620-050] best_practice

**Logged**: 2026-06-20T00:40:00+08:00
**Priority**: medium
**Status**: pending
**Area**: tests

### Summary
`handler/tag_like.go` 中 `GetDefaultLikeStore()` 调用了两次，两次调用之间有竞态窗口。应提取到外层变量。

### Details
```go
// 缺陷模式：
if s := tag.GetDefaultLikeStore(); s != nil && hasID {
    s.Like(...)
    return
}
if s := tag.GetDefaultLikeStore(); s != nil && !hasID {
    // name-based like — 需要 MongoDB 路径
}
// 如果第一次调用后、第二次调用前另一个 goroutine Reset 了 store，
// 两个条件都走不通，落到 MongoDB 路径。虽然不是 data race，但逻辑不一致。
```

修复：`s := tag.GetDefaultLikeStore()` 提前到函数入口处只调用一次。

### Metadata
- Source: code-review
- Related Files: cmd/server/handler/tag_like.go
- Tags: handler, tag-like, race-condition, code-quality
- Pattern-Key: backend.like_store.dual_call

---
## [LRN-20260620-051] insight

**Logged**: 2026-06-20T00:50:00+08:00
**Priority**: high
**Status**: resolved
**Area**: tests

### Summary
10+ 个对 `memory_storage_integration` tag 依赖的 txtar 测试在引入 `defaultStore` 守卫后不再需要 MongoDB，`handler_test.go` 的 TestMain 统一注入即可满足所有 handler 测试。

### Details
原来分散在各测试文件中的 `testMongoAvailable` 检查（37 处）和 `defer/recover` 模式（5 处）全部可移除。`search_test.go` 的 `init()` 中 `mongowrap.Init()` 会 panic，改为简单的 `var testMongoAvailable = true`。

TestMain 统一注入的所有 store：
- `comic.SetDefaultStorage` → MemoryStorage
- `tag.SetDefaultTagStore` → MemoryTagStore
- `tag.SetDefaultComicStore` → MemoryStorage
- `tag.SetDefaultLikeStore` → MemoryLikeStore
- `tag.SetDefaultRelationStore` → MemoryRelationStore
- `video.SetDefaultVideoStore` → MemoryVideoStore
- `onecomic.SetDefaultOneComicStore` → MemoryOneComicStore
- `custom.SetDefaultCustomStore` → MemoryCustomStore
- `setting` 包已通过 `memory_storage_integration` build tag 编译时切换

### Metadata
- Source: insight
- Related Files: cmd/server/handler/handler_test.go, cmd/server/handler/search_test.go
- Tags: testing, mongodb, testmain, injection

---
## [LRN-20260620-052] correction

**Logged**: 2026-06-20T01:00:00+08:00
**Priority**: medium
**Status**: resolved
**Area**: config

### Summary
`pkg/archive/manager/errors.go` 的 `IsNotFound` 引用 `mongo.ErrNoDocuments` 导致即使不使用 MongoDB 的包也要 import MongoDB 驱动。

### Details
```go
func IsNotFound(err error) bool {
    return os.IsNotExist(err) || errors.Is(err, mongo.ErrNoDocuments) || errors.Is(err, ErrNotFound)
}
```

`mongo` 是 `go.mongodb.org/mongo-driver/mongo`。当 `TestMongoIndexStoreIntegrationCRUDAndList` 改用 MemoryIndexStore 后，不再需要这个 import。而且 `mongo.ErrNoDocuments` 只应在真正的 MongoDB 路径中判断，`MemoryIndexStore.Get` 返回的是包装 `ErrNotFound` 的错误。移除后 `errors.Is(err, mongo.ErrNoDocuments)` 不再生效，但内存路径本就不需要它。

### Suggested Action
归档 manager 的 IsNotFound 只检查 `os.IsNotExist` 和 `ErrNotFound`，不依赖 MongoDB 驱动类型。MongoDB 存储层内部的 `ErrNoDocuments` 应在 mongostore.go 内部转换为 `ErrNotFound`。

### Metadata
- Source: correction
- Related Files: pkg/archive/manager/errors.go
- Tags: archive, mongodb, error-handling, import

---
## [LRN-20260620-053] insight

**Logged**: 2026-06-20T01:10:00+08:00
**Priority**: medium
**Status**: resolved
**Area**: backend

### Summary
迁移 view/tag_result.go 的直连 MongoDB 时，新增 `tag.GetTagByTypeURL` 函数遵循 `GetTagByTypeName` 模式，TagStore 接口也需同步扩展。

### Details
`view/tag_result.go` 第 71-75 行原本直接调用 `mongo.ComicTagBuilder().Filters("type", tagType, "url", url).Limit(1).All(c, &docs)`。改用新函数后需要：
1. `internal/tag/aggregate.go`：新增 `GetTagByTypeURL` 导出函数（含 defaultTagStore 守卫 + MongoDB fallback）
2. `internal/tag/default_store.go`：TagStore 接口加 `GetTagByTypeURL` 方法
3. `internal/tag/default_store.go`：MemoryTagStore 实现该方法
4. `view/tag_result.go`：移除 mongo import，改用 `tag.GetTagByTypeURL`

这是"视图层不直连 MongoDB"原则的应用——view 包应通过业务包（tag）间接访问数据。

### Metadata
- Source: insight
- Related Files: cmd/server/view/tag_result.go, cmd/server/internal/tag/default_store.go
- Tags: arch, view-layer, mongodb, tag

---
## [LRN-20260620-054] best_practice

**Logged**: 2026-06-20T01:20:00+08:00
**Priority**: medium
**Status**: pending
**Area**: tests

### Summary
MemoryTagStore 中的 `fmt.Sscanf` 被 errcheck 检测到未检查返回值，必须显式忽略。

### Details
MemoryTagStore 使用 `fmt.Sscanf(c.GetID(), "%d", &cid)` 将字符串 ID 转 int。errcheck 要求检查返回值（返回 `n, err`）。测试辅助代码中 ID 解析错误不影响正确性（错误时 cid=0），但 lint 规则强制处理。

修复方式选择：
- `_, _ = fmt.Sscanf(...)` — 显式忽略，errcheck 认可
- `strconv.Atoi(c.GetID())` — 更符合 Go 习惯，但需要 import strconv
- `cid, err := strconv.Atoi(c.GetID()); if err != nil { return ... }` — 最完整但增加代码量

本例选择了 `_, _ = fmt.Sscanf(...)` 保持最小改动。

### Metadata
- Source: error
- Related Files: cmd/server/internal/tag/default_store.go
- Tags: lint, errcheck, sscanf
- Pattern-Key: lint.errcheck.sscanf

---

