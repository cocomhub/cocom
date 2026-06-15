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
