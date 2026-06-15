# Errors

Command failures and integration errors.

---

## [ERR-20260614-001] playwright-go-evaluate-missing-args

**Logged**: 2026-06-14T17:00:00+08:00
**Priority**: high
**Status**: resolved
**Area**: tests

### Summary
`page.Locator(selector).Evaluate("js code")` 编译错误，提示参数不足。

### Error
```
not enough arguments in call to page.Locator(helpers.SearchInput).Evaluate
	have (string)
	want (string, any, ...playwright.LocatorEvaluateOptions)
```

### Context
- playwright-go v0.5700
- `Evaluate` 方法签名从可选 args 变为至少 `nil`

### Suggested Fix
所有 `Evaluate` 调用补上 `nil` 参数：`Evaluate("code", nil)`

### Metadata
- Reproducible: yes
- Related Files: tests/e2e/navigation_test.go, tests/e2e/gallery_detail_test.go
- See Also: LRN-20260614-004

---
## [ERR-20260614-002] internal-package-import-from-independent-module

**Logged**: 2026-06-14T17:00:00+08:00
**Priority**: high
**Status**: resolved
**Area**: tests

### Summary
`tests/e2e` 独立 module 无法 import `cmd/server/internal` 下的包。

### Error
```
import of internal package not allowed
```

### Context
- Go 工具链禁止独立 module 访问其它 module 的 internal/ 目录
- `tests/e2e/go.mod` 使用 `replace github.com/cocomhub/cocom => ../../`

### Suggested Fix
在 `cmd/server/handler/` 中创建桥接函数 `InitE2EStorage()`，handler 包不是 internal，可以被外部 import。

### Metadata
- Reproducible: yes
- Related Files: cmd/server/handler/e2e_storage.go
- See Also: LRN-20260614-001

---
## [ERR-20260614-003] handler-init-mongowrap-panic

**Logged**: 2026-06-14T17:00:00+08:00
**Priority**: high
**Status**: resolved
**Area**: tests

### Summary
`handler.Init()` 在测试环境中 panic，因为试图连接不存在的 MongoDB。

### Context
- `handler.Init()` 内部调用 `mongowrap.Init()` 连接 MongoDB
- 测试环境没有 MongoDB 实例

### Suggested Fix
不要调用 `handler.Init()`，改为用 `gin.WrapF(handlerFunc)` 手动逐条注册路由。

### Metadata
- Reproducible: yes
- Related Files: cmd/server/handler/init.go
- See Also: LRN-20260614-006

---
## [ERR-20260614-004] png-jpg-extension-mismatch

**Logged**: 2026-06-14T17:00:00+08:00
**Priority**: medium
**Status**: resolved
**Area**: tests

### Summary
Mock 图片使用 `png.Encode` 但文件名后缀是 `.jpg`。

### Error
静默失败：浏览器收到 PNG 字节但 Content-Type 标记为 jpeg，解码异常。

### Context
- `seed.go` 的 `generateMockImage` 用 `image/png` 编码 PNG
- 文件名 `1.jpg`、`2.jpg` 与实际编码格式不匹配

### Suggested Fix
改为 `.png` 扩展名。

### Metadata
- Reproducible: yes
- Related Files: tests/e2e/fixtures/seed.go
- See Also: LRN-20260614-003

---

## [ERR-20260615-005] retry-LimitRetry-signature-mismatch

**Logged**: 2026-06-15T03:30:00+08:00
**Priority**: high
**Status**: resolved
**Area**: tests

### Summary
`pkg/mutex/internal/retry` 的 `LimitRetry` 签名是装饰器模式，不是执行器模式。

### Suggested Fix
```go
strategy := LimitRetry(LinearBackoff(1*time.Millisecond), 3)
for i := 0; i < 5; i++ {
    d := strategy.NextBackoff()
    if d == 0 { break }
}
```

### Metadata
- Reproducible: yes
- Related Files: pkg/mutex/internal/retry/retry_test.go, pkg/mutex/internal/retry/strategy.go

---

## [ERR-20260615-006] mongowrap-NewBuilder-collection-required

**Logged**: 2026-06-15T03:30:00+08:00
**Priority**: high
**Status**: resolved
**Area**: tests

### Summary
`mongowrap.NewBuilder` 需要 `*mongo.Collection` 参数，不是无参数构造函数。

### Metadata
- Reproducible: yes
- Related Files: pkg/mongowrap/builder.go, pkg/mongowrap/mongo.go, pkg/mongowrap/mongowrap_test.go

---

## [ERR-20260615-007] cmd-Execute-function-comparison

**Logged**: 2026-06-15T03:30:00+08:00
**Priority**: low
**Status**: resolved
**Area**: tests

### Summary
`cmd.Execute` 是函数变量，不能检查 `nil`（Go 编译器警告为 never nil）。

### Metadata
- Reproducible: yes
- Related Files: cmd/cmd_test.go

---

## [ERR-20260615-008] NewComicVerifier-wget-panic-windows

**Logged**: 2026-06-15T11:30:00+08:00
**Priority**: high
**Status**: resolved
**Area**: tests

### Summary
`NewComicVerifier` 在 Windows 上 panic，因为 `findWgetPath()` 直接 `panic("wget未找到")`。`service_test.go` 和 `verify_test.go` 无法通过 `NewService` 进行测试。

### Error
```
panic: wget未找到，请确保已安装wget
```

### Context
- Windows 环境默认没有 wget
- `NewService` → `NewComicVerifier` → `findWgetPath` → panic
- 所有需要用 `NewService` 或 `NewComicVerifier` 的测试都受影响

### Suggested Fix
- `service_test.go`: 直接 `&ServiceImpl{storage: ms}` 避免 verifier 初始化
- `verify_test.go`: 跳过需要 wget 的测试（`t.Skip("requires wget binary")`）

### Metadata
- Reproducible: yes
- Related Files: pkg/comic/comic.go:558, pkg/comic/verify.go:366, pkg/comic/service_test.go, pkg/comic/verify_test.go

---

## [ERR-20260615-009] E2E-vet-status-method-vs-field

**Logged**: 2026-06-15T11:30:00+08:00
**Priority**: medium
**Status**: resolved
**Area**: tests

### Summary
`httptest.ResponseRecorder.Result()` 返回 `*http.Response`，其 `Status` 是方法（`Status()`）不是字段。`resp.Status == 200` 编译错误。

### Error
```
random_gallery_test.go:73:22: invalid operation: resp.Status == 200 (mismatched types func() int and untyped int)
```

### Context
- `httptest.NewRecorder().Result()` 返回 `*http.Response`
- `http.Response.Status` 是字符串字段，`http.Response.StatusCode` 才是 int
- `resp.Status == 200` 应该是 `resp.StatusCode == 200` 或 `resp.Status() == "200 OK"`

### Metadata
- Reproducible: yes
- Related Files: tests/e2e/random_gallery_test.go

---

## [ERR-20260615-010] E2E-vet-extra-brace-after-PowerShell-replace

**Logged**: 2026-06-15T11:30:00+08:00
**Priority**: medium
**Status**: resolved
**Area**: tests

### Summary
PowerShell 的文本替换在处理跨行代码块时，由于缩进层级不一致（3-tab 和 2-tab 混合），导致一个多余的右花括号未被替换掉，引起 `go vet` 失败。

### Error
```
gallery_detail_test.go:75:5: missing ',' before newline in argument list
gallery_detail_test.go:87:3: expected statement, found ')'
```

### Context
- `ZoomReset` 测试块中原有 `if IsVisible ResetBtn { Skip }`（2-tab indent）
- PowerShell `.Replace()` 替换后变成了 3-tab `if IsVisible ZoomSidebar { EnterLargeMode }`
- 外加一个额外的 2-tab `}` 未被移除

### Suggested Fix
替换跨行 Go 代码块时优先使用 sed 或 Go 脚本，而非 PowerShell 的字符串替换。

### Metadata
- Reproducible: yes
- Related Files: tests/e2e/gallery_detail_test.go

## [ERR-20260615-011] git-commit-message-shell-interpolation

**Logged**: 2026-06-15T16:00:00+08:00
**Priority**: high
**Status**: resolved
**Area**: infra

### Summary
Bash 工具中执行多行 `git commit -m "..."` 时，message 内的反引号、$变量、括号会被 shell 解析执行，导致错误输出污染 commit message。

### Error
```
/usr/bin/bash: command substitution: line 20: syntax error: unexpected end of file
/usr/bin/bash: t.Skip: command not found
/usr/bin/bash: t.Logf: command not found
```

### Context
- Bash 工具将 `git commit -m "..."` 的引号内文本传给 shell 解释
- `t.Skip("requires wget")` 被当作命令调用
- `$(...)` 内嵌表达式被展开
- heredoc 也在 shell 中执行

### Suggested Fix
将 commit message 写入临时文件再 `git commit -F /tmp/msg`，避免 shell 解释。

### Metadata
- Reproducible: yes
- Related Files: (none)
- See Also: LRN-20260615-024

---

## [ERR-20260615-012] rebase-i-editor-not-available-windows

**Logged**: 2026-06-15T16:00:00+08:00
**Priority**: medium
**Status**: resolved
**Area**: infra

### Summary
Windows 上 `GIT_SEQUENCE_EDITOR="sed -i '...'" git rebase -i` 失败，因为系统没有默认 EDITOR 且 `git -c core.editor=true` 不是有效命令。

### Error
```
git: 'D:/workdir/leon/cocomhub/cocom/.git/rebase-merge/git-rebase-todo' is not a git command.
```

### Context
- Windows Git Bash 中 `rebase -i` 需要 editor
- `sed -i` 内联编辑 rebase todo 文件前需要先初始化 editor
- `GIT_SEQUENCE_EDITOR` 只有在 editor 完成后才会输入

### Suggested Fix
使用 `GIT_SEQUENCE_EDITOR="sed -i '2s/^pick/squash/'" git rebase -i HEAD~N` 方案，或在 PowerShell 中直接手动编辑 `.git/rebase-merge/git-rebase-todo`。

### Metadata
- Reproducible: no (Windows-specific)
- Related Files: (none)
- See Also: LRN-20260615-024
---
