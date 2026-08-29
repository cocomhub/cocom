# Test Experience Optimization Design (Phase 2)

> 在 Phase 1 测试基础设施重构（MemoryStorage + CLI 测试 + 数据工厂 + 全量空壳测试文件）基础上，实现第二阶段目标：**完全去 MongoDB 化运行测试** + **E2E UI 自动化测试框架** + **AI 辅助浏览器测试**。

## 背景

Phase 1 已将大部分 handler 测试从 MongoDB 迁移到 `MemoryStorage`（通过 `internal/comic.SetDefaultStorage` 注入），但仍有 2 个测试文件（`search_test.go`、`tags_search_test.go`）共 8 个测试用例依赖 MongoDB 的 `testMongoAvailable` 跳过守卫。更深层的问题是标签模块（`cmd/server/internal/tag/`）全部直接调用 MongoDB 聚合管道，无存储抽象。

用户核心关注三点：
1. **完全不依赖 MongoDB 运行所有测试**
2. **chromedp + MCP 双轨 UI 自动化测试**，增强 AI 设计 UI 的能力
3. **WebMCP 新标准关注**，为未来手机端优化预留方向

## 架构设计

### 存储层扩展

**原则**：标签是漫画数据的物化视图，不是独立数据源。`comicTag` 集合本质是对 `comicInfo.tags` 数组的聚合结果。因此不单独定义 `TagStorage` 接口，而是将标签搜索/列表能力扩展到 `comic.Storage` 接口中。

**`pkg/comic.Storage` 接口新增方法**：

```go
type Storage interface {
    // ...现有 9 个方法 (Get/Update/Find/FindTotal/FindChannel/
    //     SaveVerifyResult/ArchiveByID/RestoreByID/FindByTags)...

    // SearchTags 按名称搜索标签（支持模糊匹配），从漫画数据推导
    SearchTags(ctx context.Context, tagType string, query string, limit int64) ([]TagInfo, int64, error)

    // ListTags 获取标签列表（分页、排序、仅点赞），从漫画数据推导
    ListTags(ctx context.Context, tagType string, sortType int, skip, limit int64, likedOnly bool) ([]TagInfo, int64, error)
}
```

**`TagInfo` 新定义**（在 `pkg/comic/` 包内）：

```go
type TagInfo struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Type  string `json:"type"`
    URL   string `json:"url"`
    Count int    `json:"count"`
    Like  bool   `json:"like"`
}
```

**`MemoryStorage` 实现策略**：

标签搜索/列表实现通过遍历 `m.comics` 实时推导：
- 遍历所有漫画，收集其 `GetTags()` 中的标签 → 按 `type:id` 去重 → 计数
- `SearchTags`: 匹配后过滤 `name`（`strings.Contains` 或 `regexp`），按 `count` 降序，截取 `limit`
- `ListTags`: 支持分页（skip/limit）、排序（name 升序 / count 降序）、`likedOnly` 过滤
- 点赞状态：在 `MemoryStorage` 里维护一个 `likedTags map[string]bool`（key: `"type:id"`）

**MongoDB 实现**（`pkg/comic/storage/mongo.go`）：
- 对接现有 `comicTag` 集合，复用现有 MongoDB 查询代码
- `SearchTags` → `$regex` 匹配 + `$sort` count 降序
- `ListTags` → `$match(type)` + `$sort` + `$skip` + `$limit`

### 标签点赞和关系处理

这两项不合并到 `Storage` 接口，保持在 `internal/tag` 包内：

- **点赞**：`internal/tag` 包新增 `DefaultLikeStorage` 机制，测试时注入内存 `map[string]bool`
- **关系**：标签关系是独立数据（标签之间的关联，不属于漫画），保持现有 `internal/tag/relation.go`，同样加 `DefaultRelationStorage` 注入

不抽象成通用接口（仅在测试用），通过包级函数 + 构建标签或钩子注入实现。

### ComicFilter 扩展

为支持首页/搜索页面的过滤需求，扩展 `ComicFilter`：

```go
type ComicFilter struct {
    // ...现有字段...
    Status          *bool     // 新增：启用状态过滤
    Deleted         *bool     // 新增：删除标记过滤
    HasRedirect     *bool     // 新增：排除重定向漫画
    TitleORPatterns []string  // 新增：多字段标题 OR 搜索（匹配 english/japanese/pretty）
    TagIDs          []int     // 新增：按标签 ID 过滤
}
```

### chromedp UI 自动化测试框架

**隔离设计**：独立 Go module 在 `tests/chromedp/`，避免污染主库 `go.mod`。

```
tests/chromedp/
├── go.mod                    # module github.com/cocomhub/cocom/tests/chromedp
│   replace github.com/cocomhub/cocom => ../../
├── main_test.go              # TestMain: 启动 Gin TestServer + 注入 MemoryStorage
├── helpers/
│   ├── page.go               # 导航/等待/截图辅助
│   └── selector.go           # CSS 选择器常量
├── tests/
│   ├── home_test.go          # 首页 E2E
│   ├── search_test.go        # 搜索 E2E
│   ├── gallery_test.go       # 画廊详情 E2E
│   ├── tag_test.go           # 标签浏览 E2E
│   └── admin_test.go         # 管理后台 E2E
└── fixtures/
    └── seed.go               # 种子数据（复用 testutil 工厂）
```

**chromedp vs MCP 双轨分工**：

| 维度 | chromedp | MCP (chrome-devtools-mcp) |
|------|----------|--------------------------|
| 目的 | CI 可执行的回归测试 | AI 探索式测试 + 设计验证 |
| 运行环境 | `make test-chromedp` | Claude 会话中手动触发 |
| 断言方式 | Go 代码精确断言 | AI 视觉/语义判断 |
| 截图用途 | 保存到文件供 diff 对比 | 实时查看 UI 效果 |

### MCP AI 辅助测试

```
tests/mcp/
├── scenarios/
│   ├── home-page.md          # 首页交互场景
│   ├── search-flow.md        # 搜索流程
│   ├── gallery-detail.md     # 画廊详情页操作
│   └── admin-operations.md   # 管理后台操作
└── config.yaml               # MCP 测试配置（server URL、seed API）
```

场景描述包含：前置条件、操作步骤、断言点、验证标准。配合 chrome-devtools-mcp，AI 可以直接操作浏览器进行探索式测试和 UI 设计验证。

### Makefile 集成

```makefile
.PHONY: test-chromedp
test-chromedp:
	cd tests/chromedp && go test -tags=memory_storage_integration -count=1 ./...

.PHONY: test-all
test-all: test test-chromedp
```

## 实施计划

### P0：完全去 MongoDB 化（核心）

| # | 任务 | 文件 |
|---|------|------|
| 1 | ComicFilter 扩展 + MemoryStorage.Find 完善 | `pkg/comic/storage.go` |
| 2 | Storage 接口新增 SearchTags/ListTags | `pkg/comic/storage.go`, `pkg/comic/storage/mongo.go` |
| 3 | MemoryStorage 实现标签推导 | `pkg/comic/storage.go` |
| 4 | 各 Storage 包装层同步新增方法 | `cmd/server/internal/comic/storage.go`, `cmd/server/internal/onecomic/storage.go` |
| 5 | tag 包函数接入 DefaultStorage | `cmd/server/internal/tag/aggregate.go` (15+ 函数) |
| 6 | tag 包函数接入 DefaultStorage | `cmd/server/internal/tag/relation.go` (6+ 函数) |
| 7 | 改造 SearchAutocomplete handler | `cmd/server/handler/search.go` |
| 8 | TestMain 注入 + 移除 MongoDB | `cmd/server/handler/handler_test.go` |
| 9 | search_test.go 移除 skip + 种子数据 | `cmd/server/handler/search_test.go` |
| 10 | tags_search_test.go 迁移到内存存储 | `cmd/server/handler/tags_search_test.go` |
| 11 | 验证 `make test` 无 MongoDB 通过 | CI |

### P1：测试场景数据 + chromedp 框架

| # | 任务 | 文件 |
|---|------|------|
| 1 | 扩展 testutil.MockComicInfo 支持更多选项 | `cmd/server/internal/testutil/factory.go` |
| 2 | 场景预设系统 | `cmd/server/internal/testutil/scenarios.go` |
| 3 | 批量生成器 | `cmd/server/internal/testutil/generator.go` |
| 4 | 导出 SeedTestData 函数 | `cmd/server/handler/testdata.go` (新增) |
| 5 | chromedp 独立 module + TestMain | `tests/chromedp/go.mod`, `tests/chromedp/main_test.go` |
| 6 | chromedp 辅助库 | `tests/chromedp/helpers/page.go`, `selector.go` |
| 7 | chromedp 核心测试 | `tests/chromedp/tests/home_test.go`, `search_test.go`, `gallery_test.go` 等 |
| 8 | Makefile 集成 | `Makefile` |

### P2：MCP AI 辅助

| # | 任务 | 文件 |
|---|------|------|
| 1 | MCP 场景描述 | `tests/mcp/scenarios/*.md` |
| 2 | MCP 测试配置 | `tests/mcp/config.yaml` |
| 3 | AI 辅助测试说明文档 | `tests/mcp/README.md` |

## 验证方案

- **P0 验证**：`go test -tags=memory_storage_integration -count=1 ./cmd/server/...` 在无 MongoDB 环境下全部通过
- **P1 验证**：`make test-chromedp` 启动浏览器执行 E2E 流程并截图
- **P2 验证**：按 `tests/mcp/scenarios/*.md` 手动执行 MCP 操作，确认 UI 设计一致性

## 未含内容（明确不做的）

- Redis 缓存抽象 — 当前缓存仅用于优化，不影响功能正确性
- BaiduPCS 后端测试 — 第三方网盘服务，不适合单元测试
- 非 Chrome 浏览器测试 — 用户明确指定 Chrome
- 性能基准测试 — 当前目标是功能正确性和回归测试
