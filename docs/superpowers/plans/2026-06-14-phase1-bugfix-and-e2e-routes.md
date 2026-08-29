# Phase 1 实现计划：缺陷修复与统一路由注册

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 修复已知代码缺陷，统一 E2E 路由注册（复用生产代码而非手动维护两套），为 Phase 2/3 的 E2E 测试增强铺平道路。

**架构：** 利用 `cmd/server/internal/comic.NewTestStorage(store)` 已有的 `inner` 委托模式，在 `handler/e2e_storage.go` 新增 `RegisterE2ERoutes` 函数，复用 `pkg/comic.Handler.RegisterRoutes` 注册 v2 路由；提取 `init.go` 中的 API 路由为 `registerAPIRoutes(r)` 供生产与 E2E 共用；JS 层注入 `window.__E2E_TEST__` 标志使测试环境跳过 `location.reload()` 改为 AJAX 更新。

**技术栈：** Go 1.26, Gin, playwright-go, JavaScript（原生 DOM API）

---

## 需修改的文件清单

| 文件 | 职责 |
|------|------|
| `cmd/server/handler/init.go` | 提取 API 路由为 `registerAPIRoutes(r)`，`Init()` 调用它 |
| `cmd/server/handler/e2e_storage.go` | 新增 `RegisterE2ERoutes(ctx, r, store)` 统一注册所有路由 |
| `cmd/server/handler/handler_test.go` | 用 `handler.RegisterE2ERoutes` 替换手动 store 注入（可选简化） |
| `tests/e2e/main_test.go` | 用 `handler.RegisterE2ERoutes` 替换行 95-111 的手动路由注册 |
| `pkg/comic/verify.go` | 实现 `SetMessage()` 空函数体 |
| `pkg/comic/monitor.go` | IO 统计从单字段拆分为 read/write 双字段 |
| `internal/config/config.go` | 移除 3 个弃用配置 key 的 `init()` 注册 |
| `custom/js/modules/quick-link.js` | `confirmLinkAction` 中 `location.reload()` 加 `__E2E_TEST__` 条件 |
| `custom/js/modules/gallery-actions.js` | `archiveComic`/`restoreComic` 中的 toast-only 不含 reload，无需改造 |
| `custom/js/modules/admin-compare.js` | `confirmLink`/`unlinkComic` 中的 `confirm()` 加 `__E2E_TEST__` 跳过 |
| `tests/e2e/helpers/playwright.go` | 新增 `InjectTestMode(page)` 辅助函数 |
| `tests/e2e/gallery_detail_test.go` | `t.Logf` → `t.Errorf`/`t.Fatal` 硬断言化 |
| `tests/e2e/quick_action_test.go` | 同上 |
| `tests/e2e/navigation_test.go` | 同上 |
| `tests/e2e/compare_test.go` | 同上 |

---

### 任务 1：统一路由注册

**文件：**
- 修改：`cmd/server/handler/init.go:6-24`
- 修改：`cmd/server/handler/e2e_storage.go`
- 修改：`cmd/server/handler/handler_test.go:24-31`
- 修改：`tests/e2e/main_test.go:89-111`

- [ ] **步骤 1：从 `init.go` 提取 `registerAPIRoutes(r)`**

  将 `Init()` 中除 `comic.Init(ctx)`/`download.Init()`/`mongowrap.Init()` 之外的所有路由注册提取为独立函数 `registerAPIRoutes(r *gin.RouterGroup)`：

  ```go
  // cmd/server/handler/init.go

  // Init 初始化所有路由（生产环境）。
  func Init(ctx context.Context, r *gin.RouterGroup) {
      comic.Init(ctx)          // 初始化 mongowrap (MongoDB)
      download.Init()          // 初始化下载器
      mongowrap.Init()         // 初始化 MongoDB wrapper
      registerAPIRoutes(r)     // 注册 API 路由
  }

  // registerAPIRoutes 注册 /api/* 路由（生产和 E2E 共用）。
  func registerAPIRoutes(r *gin.RouterGroup) {
      r.POST(webp.InstallScriptEndpoint, ...)
      r.GET(webp.InstallScriptEndpoint, ...)
      r.POST("/api/comic/addLikeGroup", ...)
      // ... 其余所有 r.POST/GET/DELETE 路由 ...
  }
  ```

  注意：`init.go` 被压缩无法完整读取，确保 `registerAPIRoutes` 包含 `Init()` 中除了 `comic.Init()`/`download.Init()`/`mongowrap.Init()` 调用之外的所有路由注册代码。

- [ ] **步骤 2：实现 `RegisterE2ERoutes()`**

  ```go
  // cmd/server/handler/e2e_storage.go

  package handler

  import (
      "context"
      "github.com/cocomhub/cocom/cmd/server/internal/comic"
      "github.com/cocomhub/cocom/cmd/server/internal/onecomic"
      comicpkg "github.com/cocomhub/cocom/pkg/comic"
      "github.com/cocomhub/cocom/cmd/server/internal/tag"
      "github.com/gin-gonic/gin"
  )

  // InitE2EStorage 初始化 E2E 测试需要的内存存储并注入到各包默认存储中。
  func InitE2EStorage() *comicpkg.MemoryStorage {
      store := comicpkg.NewMemoryStorage()
      comic.SetDefaultStorage(store)
      tag.SetDefaultLikeStore(tag.NewMemoryLikeStore())
      tag.SetDefaultComicStore(store)
      tag.SetDefaultRelationStore(tag.NewMemoryRelationStore())
      return store
  }

  // RegisterE2ERoutes 注册 E2E 测试所需的所有路由，复用生产代码路由注册逻辑。
  // 创建 Service 实例时注入 MemoryStorage 而非 MongoDB。
  func RegisterE2ERoutes(ctx context.Context, r *gin.Engine) {
      store := comicpkg.NewMemoryStorage()
      comic.SetDefaultStorage(store)
      tag.SetDefaultLikeStore(tag.NewMemoryLikeStore())
      tag.SetDefaultComicStore(store)
      tag.SetDefaultRelationStore(tag.NewMemoryRelationStore())
      registerE2EGinRoutes(ctx, r, store)
  }

  // RegisterE2ERoutesWithStore 使用已有 store 注册 E2E 路由（供 TestMain 中 seed 后使用）。
  func RegisterE2ERoutesWithStore(ctx context.Context, r *gin.Engine, store *comicpkg.MemoryStorage) {
      comic.SetDefaultStorage(store)
      tag.SetDefaultLikeStore(tag.NewMemoryLikeStore())
      tag.SetDefaultComicStore(store)
      tag.SetDefaultRelationStore(tag.NewMemoryRelationStore())
      registerE2EGinRoutes(ctx, r, store)
  }

  // registerE2EGinRoutes 注册 Gin 路由的内部实现。
  func registerE2EGinRoutes(ctx context.Context, r *gin.Engine, store *comicpkg.MemoryStorage) {
      // API 路由（与生产代码共用）
      apiGroup := r.Group("/api")
      registerAPIRoutes(apiGroup)

      // 管理员路由
      adminGroup := r.Group("/admin")
      adminGroup.POST("/compare", gin.WrapF(CompareComics))
      adminGroup.POST("/swap", gin.WrapF(LinkComics))

      // v2 API 路由 — 复用 pkg/comic.Handler.RegisterRoutes（与生产代码 server.go:159 完全一致）
      nhSrv, err := comicpkg.NewService(ctx, comic.NewTestStorage(store))
      if err != nil {
          panic(fmt.Errorf("new nhcomic service for e2e failed: %w", err))
      }
      comicpkg.NewHandler(ctx, nhSrv).RegisterRoutes(r.Group("/v2/api/nhcomic"))

      ocSrv, err := comicpkg.NewService(ctx, onecomic.NewTestStorage(store))
      if err != nil {
          panic(fmt.Errorf("new onecomic service for e2e failed: %w", err))
      }
      comicpkg.NewHandler(ctx, ocSrv).RegisterRoutes(r.Group("/v2/api/onecomic"))

      // g/api 路由 — gallery_detail 前端调用的 like/archive/restore
      galleryGroup := r.Group("/g")
      galleryGroup.POST("/api/like", gin.WrapF(LikeTag))
      galleryGroup.POST("/api/archive", gin.WrapF(AddLikeGroup))
      galleryGroup.POST("/api/restore", gin.WrapF(RestoreComic))
  }
  ```

- [ ] **步骤 3：更新 `tests/e2e/main_test.go`**

  将行 89-111 的手动路由注册替换为 `handler.RegisterE2ERoutesWithStore(ctx, r, testMemStore)`：

  ```go
  // tests/e2e/main_test.go

  // 注册所有路由 — 复用生产代码
  handler.RegisterE2ERoutesWithStore(ctx, r, testMemStore)
  ```

  删除原来的手动路由注册块（`apiGroup`/`adminGroup`/`galleryGroup` 全部）。

- [ ] **步骤 4：更新 `handler/handler_test.go`**（可选简化）

  handler 测试的 `TestMain` 已使用 `comic.SetDefaultStorage(testMemStorage)` + `tag.SetDefaultLikeStore(testTagLikeStore)` 手动注入。可以不改（它不依赖路由注册），但需确认两个 TestMain 不冲突。

- [ ] **步骤 5：验证**

  运行：`cd tests/e2e && CGO_ENABLED=1 go test -count=1 -v -timeout 120s -run TestNavigation ./...`
  预期：现有的导航测试通过，v2 路由已注册（无 panic）

  运行：`go test -tags=memory_storage_integration -timeout 5m -run TestHandler ./cmd/server/handler/...`
  预期：handler 测试不受影响

---

### 任务 2：代码缺陷修复

- [ ] **步骤 1：实现 `SetMessage()`**

  ```go
  // pkg/comic/verify.go — VerifyProgress 结构体添加字段
  type VerifyProgress struct {
      TaskID   string    `json:"taskId"`
      // ... 已有字段 ...
      messages []string  // 内部消息列表
      mu       sync.Mutex
  }

  // SetMessage 设置进度消息
  func (p *VerifyProgress) SetMessage(msg string) {
      p.mu.Lock()
      defer p.mu.Unlock()
      p.messages = append(p.messages, msg)
  }

  // GetMessages 获取所有进度消息
  func (p *VerifyProgress) GetMessages() []string {
      p.mu.Lock()
      defer p.mu.Unlock()
      result := make([]string, len(p.messages))
      copy(result, p.messages)
      return result
  }
  ```

  添加 `sync` 到 import。

- [ ] **步骤 2：IO 统计拆分**

  ```go
  // pkg/comic/monitor.go — 替换字段
  type MonitorStats struct {
      // ... 已有字段 ...
      DiskIO      uint64  // 替换为：
      NetworkIO   uint64

      // 新字段：
      DiskRead    atomic.Uint64  `json:"disk_read"`
      DiskWrite   atomic.Uint64  `json:"disk_write"`
      NetworkRead atomic.Uint64  `json:"network_read"`
      NetworkWrite atomic.Uint64 `json:"network_write"`
  }
  ```

  `GetResourceStats()` 方法返回时填充新字段：

  ```go
  func (m *Monitor) GetResourceStats() *ResourceStats {
      return &ResourceStats{
          NumGoroutine: runtime.NumGoroutine(),
          MemStats:     m.getMemStats(),
          DiskRead:     m.stats.DiskRead.Load(),
          DiskWrite:    m.stats.DiskWrite.Load(),
          NetworkRead:  m.stats.NetworkRead.Load(),
          NetworkWrite: m.stats.NetworkWrite.Load(),
          GCStats:      m.stats.GCStats,
          RetryCount:   len(m.retryQueue),
          QueueLength:  len(m.checkpoints),
      }
  }
  ```

  搜索 `ResourceStats` 结构定义并添加新字段。

- [ ] **步骤 3：弃用配置 key 清理**

  ```go
  // internal/config/config.go — 移除 init() 中的三行
  func init() {
      viper.SetDefault(StorageGalleryKey, "/data/cocom/data/gallery")
      viper.SetDefault(StorageArchiveKey, "/data/cocom/data/archive")
      viper.SetDefault(StorageArchiveTempKey, "/data/cocom/data/archive-temp")
      // 移除以下三行：
      // viper.SetDefault("cocom.archive.password", "archive@123456")
      // viper.SetDefault("cocom.archive.cmd", "7z")
      // viper.SetDefault("cocom.archive.replicate", false)
      // ... 保留其余 SetDefault ...
  }
  ```

  getter 函数保留不变（它们已有 fallback 逻辑）。

- [ ] **步骤 4：验证**

  运行：`go test -tags=memory_storage_integration -timeout 5m ./pkg/comic/... ./internal/config/...` 预期：全绿

---

### 任务 3：JS location.reload() + confirm() 改造

- [ ] **步骤 1：`helpers/playwright.go` 新增 `InjectTestMode`**

  ```go
  // tests/e2e/helpers/playwright.go

  // InjectTestMode 在页面上注入 __E2E_TEST__ 标志，使 JS 跳过 location.reload() 等操作。
  func InjectTestMode(tb testing.TB, page playwright.Page) {
      tb.Helper()
      _, err := page.Evaluate("window.__E2E_TEST__ = true")
      if err != nil {
          tb.Fatalf("failed to inject test mode: %v", err)
      }
  }
  ```

  添加 `testing` import。

- [ ] **步骤 2：`quick-link.js` 改造 `confirmLinkAction`**

  ```javascript
  // cmd/server/view/static/custom/js/modules/quick-link.js

  function confirmLinkAction() {
      if (!state.mainCID || state.selectedCIDs.length === 0) {
          showToast('请选择主 comic 和至少 1 个备选 comic', 'error');
          return;
      }

      if (
          !window.__E2E_TEST__ &&
          !confirm(
              '确认将 ' +
                  state.selectedCIDs.length +
                  ' 个备 comic 链接到 ' +
                  state.mainCID +
                  '？',
          )
      )
          return;

      fetch('/api/admin/comic/link', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
              main_cid: state.mainCID,
              sub_cids: state.selectedCIDs,
          }),
      })
          .then(function (r) {
              return r.json();
          })
          .then(function (data) {
              if (data.head && data.head.code === 0) {
                  showToast('链接成功！', 'success');
                  exitMode();
                  if (window.__E2E_TEST__) {
                      // E2E 测试环境：不重载，由测试验证 toast 和状态
                  } else {
                      location.reload();
                  }
              } else {
                  showToast(
                      '链接失败: ' + (data.head ? data.head.msg : '未知错误'),
                      'error',
                  );
              }
          })
          .catch(function (err) {
              showToast('请求失败: ' + err.message, 'error');
          });
  }
  ```

- [ ] **步骤 3：`admin-compare.js` 改造 `confirmLink` + `unlinkComic`**

  ```javascript
  // cmd/server/view/static/custom/js/modules/admin-compare.js

  window.confirmLink = function () {
      var mainCID = parseInt(document.getElementById('link-main').value, 10);
      var subCID = parseInt(document.getElementById('link-sub').value, 10);
      if (!mainCID || !subCID || mainCID === subCID) {
          showAdminToast('请输入有效的主/从 CID');
          return;
      }
      if (
          !window.__E2E_TEST__ &&
          !confirm(
              '确认将从属 CID ' +
                  subCID +
                  ' 链接到主 CID ' +
                  mainCID +
                  ' ？\n操作可撤销。',
          )
      )
          return;
      // ... 其余不变 ...
  };

  window.unlinkComic = function (subCID) {
      if (
          !window.__E2E_TEST__ &&
          !confirm('确认取消 CID ' + subCID + ' 的从属关系？已合并的 tags 将保留。')
      )
          return;
      // ... 其余不变 ...
  };
  ```

- [ ] **步骤 4：验证 JS 修改**

  手动确认：搜索所有 `location.reload()` 和 `confirm(` 在 `custom/js/modules/` 下，确认所有需要改造的都已覆盖。

---

### 任务 4：测试硬断言化

- [ ] **步骤 1：`gallery_detail_test.go` — 审查并替换**

  搜索 `t.Logf` 并逐行判断：
  - `t.Logf("Like button text: %s", text)` → `t.Errorf("expected Like button text to contain %q, got %q", want, text)`
  - 如果只是调试日志（如 `t.Logf("clicked zoom preset")`）则保留
  - 前置条件失败（如按钮未找到）→ `t.Fatalf`

- [ ] **步骤 2：`quick_action_test.go` — 审查并替换**

  Same approach as above.

- [ ] **步骤 3：`navigation_test.go` — 审查并替换**

  Same approach as above.

- [ ] **步骤 4：`compare_test.go` — 审查并替换**

  Same approach as above.

- [ ] **步骤 5：验证**

  运行：`cd tests/e2e && CGO_ENABLED=1 go test -count=1 -v -timeout 120s ./...`
  预期：所有测试通过，无 `t.Logf` 断言残留

---

### 验证

最终运行 `cd tests/e2e && CGO_ENABLED=1 go test -count=1 -v -timeout 120s ./...` 全部通过。`make test` 通过。
