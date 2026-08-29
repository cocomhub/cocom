# 四阶段缺陷修复与 E2E 测试增强设计

> 基于 brainstorming /go 修复/完善项目缺陷、E2E 测试、UI 自动化测试需求
> 日期：2026-06-14

## 背景与目标

cocom 项目的 E2E 测试框架（playwright-go）已完成基础搭建，但存在以下不足：

1. **v2 API 路由未注册** — `pkg/comic.Handler.RegisterRoutes` 在 E2E TestMain 中被跳过，导致点赞/归档/恢复按钮仅验证 DOM 可见性，无法真实点击触发 XHR
2. **测试质量偏低** — 大量 `t.Logf` 软断言而非 `t.Errorf` 硬断言
3. **子测试被跳过** — 移动端汉堡菜单、缩放滑块等交互未验证
4. **搜索无防抖** — 快速输入时 XHR 请求无 `setTimeout/clearTimeout` 防抖
5. **32 个测试文件仅作编译检查** — `Test*_Compiles` 占位符不验证任何逻辑
6. **代码缺陷** — 空函数体、占位 IO 统计、弃用配置 key

本设计覆盖四个阶段，总目标：**修复已知缺陷 + 补齐当前 E2E 场景 + 扩展新页面测试 + 填充占位符测试**。

## 关键设计决策

### 1. 统一路由注册：消除两套维护

**现状：** E2E TestMain（`tests/e2e/main_test.go`）手工注册路由子集，生产路由在 `cmd/server/handler/init.go` 注册两套（`/api/*` + `pkg/comic.Handler` 的 `/v2/api/nhcomic/*`）。这导致 v2 API 路由在 E2E 中缺失。

**方案：** 利用 `cmd/server/internal/comic.NewTestStorage(store)` 已有的委托模式——当 `inner` 字段不为 nil 时所有存储操作委托给 MemoryStorage。在 `handler/e2e_storage.go` 中新增 `RegisterE2ERoutes`：

```
// cmd/server/handler/e2e_storage.go (新增)
func RegisterE2ERoutes(ctx context.Context, r *gin.Engine, store *comicpkg.MemoryStorage) {
    // 注入 MemoryStorage 到 internal/comic 包
    comic.SetDefaultStorage(store)
    tag.SetDefaultLikeStore(tag.NewMemoryLikeStore())
    tag.SetDefaultComicStore(store)
    tag.SetDefaultRelationStore(tag.NewMemoryRelationStore())

    // 创建 Service 实例（使用 MemoryStorage）
    nhSrv, _ := comicpkg.NewService(ctx, comic.NewStorage())
    ocSrv, _ := comicpkg.NewService(ctx, onecomic.NewStorage())

    // 注册所有路由：/api/* + /v2/api/nhcomic/* + /v2/api/onecomic/*
    // 复用生产代码的 handler.Init() 的 /api 路由
    registerAPIRoutes(r)

    // 复用 pkg/comic.Handler.RegisterRoutes 注册 v2 路由 — 与生产代码完全一致
    comicpkg.NewHandler(ctx, nhSrv).RegisterRoutes(r.Group("/v2/api/nhcomic"))
}
```

**好处：** 生产 `server.go:159-160` 也是 `comicpkg.NewHandler(...).RegisterRoutes(r.Group(...))`，E2E 测试与生产使用**完全一致的路由注册代码**。

### 2. location.reload() 改造（A 方案）

**问题：** `quick-link.js:329` 在链接确认后调用 `location.reload()`，测试无法验证后续 UI 状态。

**方案：** 在所有 `location.reload()` 调用前检查 `window.__E2E_TEST__`：

```javascript
// quick-link.js
function confirmLinkAction() {
    // ... fetch /api/admin/comic/link ...
    .then(function (data) {
        if (data.head && data.head.code === 0) {
            showToast('链接成功！', 'success');
            exitMode();
            if (window.__E2E_TEST__) {
                // E2E 测试环境下不重载 — 用 UI 状态更新代替
                updateLinkUI(state.mainCID, state.selectedCIDs);
            } else {
                location.reload();
            }
        }
    })
}
```

`window.__E2E_TEST__` 在 E2E TestMain 中通过 `page.Evaluate("window.__E2E_TEST__ = true")` 注入。

### 3. E2E 移动端测试

使用 Playwright 内置设备描述符创建独立 browser context：

```go
// tests/e2e/main_test.go (新增 mobilePage 创建)
mobileCtx, _ = browser.NewContext(playwright.BrowserNewContextOptions{
    UserAgent: playwright.String("Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X)..."),
    Viewport:  &playwright.BrowserNewContextOptionsViewport{Width: 390, Height: 844},
    IsMobile:  playwright.Bool(true),
})
mobilePage, _ = mobileCtx.NewPage()
```

## 分阶段实现计划

### Phase 1：代码缺陷修复

| 文件 | 改动 |
|------|------|
| `pkg/comic/verify.go` | 实现 `SetMessage(msg string)` — 追加消息到内部列表，暴露 `GetMessages()` |
| `pkg/comic/monitor.go` | 将 `DiskIO`/`NetworkIO` 拆分为 `DiskRead`/`DiskWrite`/`NetworkRead`/`NetworkWrite`，使用 `atomic.Int64` |
| `cmd/server/handler/e2e_storage.go` | 新增 `RegisterE2ERoutes()` — 统一注册 `/api/*` + v2 路由 |
| `tests/e2e/main_test.go` | 调用 `handler.RegisterE2ERoutes()` 替代手动路由注册 |
| `cmd/server/view/static/custom/js/modules/search-autocomplete.js` | 300ms 防抖（`clearTimeout` + `setTimeout`） |
| `tests/e2e/*_test.go` | 所有 `t.Logf` 断言改为 `t.Errorf`/`t.Fatal` 硬断言 |
| `internal/config/config.go` | 移除 `cocom.archive.*` 3 个弃用 key 的 `init()` 注册 |
| `custom/js/modules/quick-link.js:329` | 注入 `window.__E2E_TEST__` + 条件跳过 `location.reload()` |
| `custom/js/modules/gallery-actions.js` | 类似改造：`archiveComicCID` / `restoreComicCID` 中的 `location.reload()` |

### Phase 2：E2E 当前页面补齐

| 测试文件 | 新增/增强子测试 |
|----------|----------------|
| `gallery_detail_test.go` | `LikeToggle_ActualClick` — 点击后验证按钮文本 ♡ → ♥ |
| | `ArchiveRestore_ActualClick` — 归档后按钮变"恢复"，恢复后变"归档" |
| | `ZoomSlider_Change` — 滑块设置 800px，验证 zoomValue |
| | `ZoomPlusMinus` — 点击 +/- 验证步进 ±20px |
| | `ZoomMobile` — 移动端浮动按钮切换缩放侧边栏 |
| | `DeleteConfirm_Cancel` — 删除按钮 + dialog.dismiss() |
| | `DeleteConfirm_Accept` — 删除按钮 + dialog.accept() |
| `compare_test.go` | `Error_InvalidCID` — 硬断言错误文本 |
| | `LinkConfirm` — 链接确认后验证 UI 状态 |
| | `Preview_KeyboardNav` — 预览覆盖层 ArrowLeft/ArrowRight/Esc |
| `navigation_test.go` | `MobileHamburger` — 移动端上下文 + 汉堡菜单交互 |
| `quick_action_test.go` | `LinkMode_VerifyCSS` — 验证 selected-main/sub 样式 |
| | `CompareMode_Redirect` — 验证跳转到 /admin?cids=... |
| | `EscapeKey_ExitMode` — Esc 退出链接/对比模式 |

### Phase 3：扩展新页面 E2E

| 新增测试文件 | 测试内容 |
|-------------|---------|
| `search_page_test.go` | 搜索 `/search?q=naruto` → 验证 gallery cards、标题匹配、分页 |
| `tag_list_test.go` | `/list/tags/` → 标签网格、A-Z/热门排序、字母分页 |
| `random_test.go` | `/random/` → 跳转到有效详情页 |
| `page_manager_test.go` | 详情页激活页管理器 → 插入/删除/替换/重排模式 UI |
| `recommend_test.go` | 详情页推荐容器 → 骨架屏 → 内容加载 |

### Phase 4：占位符测试补全

32 个占位符测试按优先级分组填充：

**高优先级（核心 logic 包）：**
- `pkg/logging/` — 测试日志初始化、level 设置、context 挂载
- `pkg/middlewares/` — 测试 AccessLog、CORS、限流中间件响应头
- `pkg/mongowrap/` — 测试配置初始化路径

**中优先级（cmd 包）：**
- `cmd/cmv/` — 测试 CLI 参数解析
- `cmd/image/` — 测试图片处理命令 flag 解析
- `cmd/install/` — 测试安装路径解析

**低优先级（简单导出包）：**
- `pkg/conv/`、`pkg/version/`、`pkg/man/`、`pkg/errwrap/` 等

## 不做项（记录经验待办）

以下功能因未实现/依赖外部服务/超出范围，记录到 `docs/TODO.md`：

- **评论区 E2E 测试** — 评论功能仅占位，未实现后端逻辑
- **标签关系管理器 E2E** — 需要登录态，当前无认证体系
- **设置页 E2E** — 依赖配置持久化（MongoDB 生产环境）
- **服务器关闭测试** — 危险操作，不适合自动化
- **BaiduPCS 被跳过测试** — 需要实际百度网盘凭据

## 验证方案

- **Phase 1**：`make test` 全绿 + `make test-e2e` 全绿
- **Phase 2**：`make test-e2e` 覆盖所有子测试，Assert 硬断言全通过
- **Phase 3**：`make test-e2e` 新增测试文件无 skip
- **Phase 4**：`go test ./...` 实际测试逻辑不 panic，覆盖率提升
- **总体**：`make test-all` 全部通过
