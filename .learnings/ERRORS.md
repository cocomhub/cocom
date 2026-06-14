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
error while importing github.com/cocomhub/cocom/cmd/server/internal/comic: 
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
`pkg/mutex/internal/retry` 的 `LimitRetry` 签名是 `LimitRetry(s Strategy, max int) Strategy`，不是 `LimitRetry(max int, interval time.Duration, fn func() error)`。

### Error
```
too many arguments in call to LimitRetry
	have (number, time.Duration, func() error)
	want (Strategy, int)
```

### Context
- 把标准库 sync 的重试模式与项目的 strategy 模式混为一谈
- `LimitRetry` 是 strategy 装饰器，返回新的 Strategy，不是执行器
- 需要先创建 `Strategy`（如 `LinearBackoff`），再用 `LimitRetry` 包装，最后手动读取 backoff

### Suggested Fix
```go
strategy := LimitRetry(LinearBackoff(1*time.Millisecond), 3)
for i := 0; i < 5; i++ {
    d := strategy.NextBackoff()
    if d == 0 {
        break
    }
}
```

### Metadata
- Reproducible: yes
- Related Files: pkg/mutex/internal/retry/retry_test.go, pkg/mutex/internal/retry/strategy.go
- See Also: LRN-20260615-015

---

## [ERR-20260615-006] mongowrap-NewBuilder-collection-required

**Logged**: 2026-06-15T03:30:00+08:00
**Priority**: high
**Status**: resolved
**Area**: tests

### Summary
`mongowrap.NewBuilder` 需要 `*mongo.Collection` 参数，不是无参数。且 Builder 没有 `.URI()` 或 `.Database()` 方法。

### Error
```
not enough arguments in call to NewBuilder
	have ()
	want (*mongo.Collection)
b.URI undefined (type *Builder has no field or method URI)
```

### Context
- 错误地假设 `NewBuilder()` 是无参数构造函数
- Builder 模式用于链式构建 mongo 查询，不是配置连接
- `buildMongoDBURI()` 是与连接相关的可用函数

### Suggested Fix
测试 mongowrap 时调用其内部函数 `buildMongoDBURI()` 而非 `NewBuilder().URI().Database()`。

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
`cmd` 包中 `Execute` 是函数变量（`var Execute = rootCmd.Execute`），不是 `func() error` 类型。`if Execute == nil` 触发 Go 编译器的永不 nil 警告。

### Error
```
comparison of function Execute == nil is always false
```

### Context
- `Execute` 是 Cobra 命令的 `Execute` 方法的别名，用 `var` 声明
- Go 不允许比较函数值和 nil（总是 false）
- 编译器将其标记为编译错误（Go 1.26 更严格）

### Suggested Fix
不要检查 `Execute == nil`，改为引用 `rootCmd` 或直接调用 `Execute()` 后忽略错误。

### Metadata
- Reproducible: yes
- Related Files: cmd/cmd_test.go
- See Also: LRN-20260615-015

---