# TODO List

> 更新于 2026-06-15，基于 E2E 测试和代码审查发现的已知问题。

---

## 已实现/已修复

- [x] CI 测试工作流 (`test.yaml`) — PR/推送自动运行 test + lint + build
- [x] 覆盖率门禁 (`cover-check`) — 初始阈值 20%，Makefile + CI 集成
- [x] 基准测试框架 — MemoryStorage Save/Get/Find/SearchTags 基准测试
- [x] `pkg/comic/comic_test.go` — ComicImpl JSON round-trip、构造函数
- [x] `pkg/comic/service_test.go` — ServiceImpl 单元测试（mock storage）
- [x] `pkg/comic/verify_test.go` — VerifyProgress/VerifyTask/MetricsCollector 单元测试
- [x] `Makefile` — `bench`/`bench-cpu` 目标、默认 `-count=1` 防缓存干扰

---

## 功能增强

### 评论系统（未实现）

- [ ] 评论区仅占位 HTML（`gallery_detail.tpl`），无后端逻辑
  - 需要登录态、数据库存储、内容审核
  - E2E 测试不可行（当前跳过）
  - 参考：`cmd/server/view/static/tpl/gallery_detail.tpl` 中的 `#comment-post-container` / `#comment-container`

### 用户认证（未实现）

- [ ] 登录/注册功能仅前端链接占位，无后端实现
  - 导航栏 "Sign in" / "Register" 链接无对应 handler
  - 标签管理/点赞等需要用户态的功能当前使用全局模式

### `/random/` 路由（未实现）

- [ ] 模板 `head.tpl` 中有 `<a href="/random/">Random</a>` 链接
  - **但无对应的 Go handler 注册**，点击返回 404
  - 需要实现 `view.RandomPage` handler 并注册到 `view/init.go`
  - E2E 测试 `random_gallery_test.go:RandomRedirect` 当前为观察性测试

### 搜索种子数据（E2E）

- [ ] `SearchScenario`（CID 1011-1013: Naruto/One Piece/Bleach）未加载到 E2E seed
  - 当前 `handler/testdata.go:SeedTestData` 只加载 `HomePage`/`E2ECompare`/`E2ESidebar` 三个场景
  - 导致 `search_page_test.go` 的 SearchGalleryCards 无法验证搜索结果
  - 修复方式：在 `SeedTestData` 中加入 `SearchScenario()`

### WebP alpha 解码性能

- [ ] `pkg/imaging/webp/decode.go:176` 上游 TODO
  - 解码 *image.NRGBA 后仅提取 green 值到独立 []byte 的低效分配
  - 需要 vp8l 包 API 变更才能修复

### Monitor IO 统计

- [x] `pkg/comic/monitor.go` IO 统计已完成拆分（Phase 1 修复）
  - DiskIO/NetworkIO → DiskRead/DiskWrite/NetworkRead/NetworkWrite
  - 实际的数据采集还需要 metrics collector 接入

### Verify SetMessage

- [x] `SetMessage()` 空函数已实现（Phase 1 修复）

---

## E2E 测试待办

- [ ] **移动端更多测试**：MobileHamburger 已有（Phase 2），但移动端侧边栏折叠、touch 事件等未覆盖
- [ ] **页管理器异步操作**：插入/删除/替换/重排的实际 API 调用未 E2E 测试
- [ ] **标签编辑**：`tag-editor.js` 模态框交互未 E2E 测试
- [ ] **推荐刷新**：`recommend.js` 的推荐刷新按钮点击后验证新内容加载
- [ ] **BaiduPCS 测试**：`pkg/storage/baidupcs/config_test.go` 中的 `TestNewFnAndRegistration` 因缺乏实际凭据被 `t.Skip` 跳过

---

## 弃用/清理

- [ ] `TODO.md` 中的 "MongoDB 支持" 条目已过时（MongoDB 已是主存储），需更新描述
- [ ] `.learnings/` 目录中的已修复问题可以考虑归档（已计入 git 历史）
- [ ] `internal/config/config.go` 中 `cocom.archive.*` 弃用 key 已清理（Phase 1）
  - `GetArchivePassword()`/`GetArchiveCmd()`/`GetArchiveReplicate()` 保留向后兼容

---

## 已知架构决策记录

### E2E 测试限制
- v2 API 路由注册：Phase 1 已通过 `RegisterE2ERoutesWithStore` 解决
- `location.reload()` 跳过：Phase 1 已通过 `window.__E2E_TEST__` 解决
- `window.prompt()` 删除确认：Playwright 通过 `dialog.accept("delete")`/`dialog.dismiss()` 处理
- Zoom sidebar 默认 `display:none`：需要通过 JS `toggleLargeMode()` 或特定条件触发显示

### 存储抽象两层架构
1. `pkg/storage.Storage` — 文件/对象存储（localfs, baidupcs）
2. `pkg/comic.Storage` — 业务数据 CRUD（MongoDB, MemoryStorage）
