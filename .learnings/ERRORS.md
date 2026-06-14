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