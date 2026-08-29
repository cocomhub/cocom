# Phase 2 测试优化实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 完全去 MongoDB 化运行所有测试 + chromedp E2E UI 自动化测试框架 + MCP AI 辅助场景

**架构：** (1) 扩展 `ComicFilter` + `MemoryStorage.Find` 完善支持分页和过滤；(2) 向 `comic.Storage` 接口新增 `SearchTags`/`ListTags` 方法，`MemoryStorage` 通过遍历漫画数据实时推导标签；(3) `internal/tag` 包函数加 `DefaultTagLikeStore`/`DefaultRelationStore` 测试注入；(4) chromedp 独立 module 隔离

**技术栈：** Go 1.26, Cobra/Viper, Gin, chromedp

---

## 文件变更清单

### 修改文件（14 个）

| 文件 | 变更 | 优先级 |
|------|------|--------|
| `pkg/comic/storage.go` | ComicFilter 加 Status/Deleted/HasRedirect/TitleORPatterns/TagIDs；Storage 接口加 SearchTags/ListTags；MemoryStorage 实现标签推导 + 完善 Find 分页 | P0 |
| `pkg/comic/storage/mongo.go` | 实现 SearchTags/ListTags MongoDB 版 | P0 |
| `cmd/server/internal/comic/storage.go` | 包装层新增 SearchTags/ListTags + toMongoFilter 新字段 | P0 |
| `cmd/server/internal/onecomic/storage.go` | 同上 | P0 |
| `cmd/server/internal/tag/aggregate.go` | 所有函数加 DefaultTagLikeStore/DefaultComicStore 检查路径 | P0 |
| `cmd/server/internal/tag/relation.go` | CreateRelation/DeleteRelation/GetRelationsForTag 加 DefaultRelationStore | P0 |
| `cmd/server/handler/tag_like.go` | LikeTag/UnlikeTag 改走 DefaultTagLikeStore | P0 |
| `cmd/server/handler/search.go` | 用 ComicFilter.TitleORPatterns 替代原始 bson 构造 | P0 |
| `cmd/server/handler/handler_test.go` | 注入 MemoryStorage + testMongoAvailable 移除 | P0 |
| `cmd/server/handler/search_test.go` | 移除 MongoDB skip guard | P0 |
| `cmd/server/handler/tags_search_test.go` | 迁移到内存存储 | P0 |
| `cmd/server/internal/testutil/factory.go` | 扩展 MockComicInfo 选项 | P1 |
| `Makefile` | +test-chromedp 目标 | P1 |
| `cmd/server/view/index.go` | filters 参数改造（移除 bson.M） | P0 |

### 新增文件（12 个）

| 文件 | 说明 | 优先级 |
|------|------|--------|
| `pkg/comic/tag.go` | TagInfo 结构体定义 | P0 |
| `cmd/server/internal/tag/default_store.go` | DefaultTagLikeStore/DefaultRelationStore 注入机制 | P0 |
| `cmd/server/internal/testutil/scenarios.go` | 场景预设系统 | P1 |
| `cmd/server/internal/testutil/generator.go` | 批量数据生成器 | P1 |
| `cmd/server/handler/testdata.go` | SeedTestData 导出函数 | P1 |
| `tests/chromedp/go.mod` | 独立 module | P1 |
| `tests/chromedp/main_test.go` | TestMain | P1 |
| `tests/chromedp/helpers/page.go` | 导航/等待/截图 | P1 |
| `tests/chromedp/helpers/selector.go` | CSS 选择器常量 | P1 |
| `tests/chromedp/tests/home_test.go` | 首页 E2E | P1 |
| `tests/mcp/scenarios/*.md` | MCP 场景描述 | P2 |
| `tests/mcp/config.yaml` | MCP 测试配置 | P2 |

---

### 任务 1：扩展 ComicFilter + 完善 MemoryStorage.Find

**文件：**
- 修改：`pkg/comic/storage.go:46-59,203-243`

**上下文：** 当前 `ComicFilter` 缺少首页/搜索页面需要的 `Status`、`Deleted`、`HasRedirect` 字段，且 `MemoryStorage.Find` 不读取 `Limit`/`Skip`/`IDRangeLeft`/`IDRangeRight`。这些是 view 层渲染首页和搜索页的过滤基础。

- [ ] **步骤 1：ComicFilter 新增过滤字段**

```go
// ComicFilter 漫画过滤器
type ComicFilter struct {
    ID              *string `json:"id,omitempty"`
    IDRangeLeft     *int64  `json:"idRangeLeft,omitempty"`
    IDRangeRight    *int64  `json:"idRangeRight,omitempty"`
    TitlePattern    *string `json:"titlePattern,omitempty"`
    TitleORPatterns []string `json:"titleORPatterns,omitempty"`  // 新增：多字段 OR 搜索
    PageMin         *int64  `json:"pageMin,omitempty"`
    PageMax         *int64  `json:"pageMax,omitempty"`
    Valid           *bool   `json:"valid,omitempty"`
    HasValid        *bool   `json:"hasValid,omitempty"`
    NotArchived     *bool   `json:"notArchived,omitempty"`
    Status          *bool   `json:"status,omitempty"`           // 新增
    Deleted         *bool   `json:"deleted,omitempty"`           // 新增
    HasRedirect     *bool   `json:"hasRedirect,omitempty"`       // 新增
    TagIDs          []int   `json:"tagIDs,omitempty"`            // 新增
    Limit           int64   `json:"limit,omitempty"`
    Skip            int64   `json:"skip,omitempty"`
}
```

- [ ] **步骤 2：新增 Setter 方法**

```go
func (filter *ComicFilter) SetStatus(status bool) *ComicFilter {
    filter.Status = &status; return filter
}
func (filter *ComicFilter) SetDeleted(deleted bool) *ComicFilter {
    filter.Deleted = &deleted; return filter
}
func (filter *ComicFilter) SetHasRedirect(hasRedirect bool) *ComicFilter {
    filter.HasRedirect = &hasRedirect; return filter
}
func (filter *ComicFilter) SetTitleORPatterns(patterns ...string) *ComicFilter {
    filter.TitleORPatterns = patterns; return filter
}
func (filter *ComicFilter) SetTagIDs(ids ...int) *ComicFilter {
    filter.TagIDs = ids; return filter
}
```

- [ ] **步骤 3：完善 MemoryStorage.Find 实现分页和新过滤字段**

```go
func (m *MemoryStorage) Find(ctx context.Context, filter *ComicFilter) ([]Comic, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    var result []Comic
    for _, comic := range m.comics {
        if filter == nil {
            result = append(result, comic)
            continue
        }

        match := true

        // 标题正则（单字段，通配）
        if filter.TitlePattern != nil {
            re, err := regexp.Compile(*filter.TitlePattern)
            if err != nil {
                return nil, fmt.Errorf("无效的匹配模式: %w", err)
            }
            match = match && re.MatchString(comic.GetTitle())
        }

        // 多字段 OR 标题搜索（english/japanese/pretty）
        if match && len(filter.TitleORPatterns) > 0 {
            titleMatch := false
            for _, pattern := range filter.TitleORPatterns {
                re, err := regexp.Compile("(?i)" + pattern)
                if err != nil {
                    return nil, fmt.Errorf("无效的匹配模式: %w", err)
                }
                if re.MatchString(comic.GetTitleEnglish()) ||
                    re.MatchString(comic.GetTitleJapanese()) ||
                    re.MatchString(comic.GetTitlePretty()) {
                    titleMatch = true
                    break
                }
            }
            match = match && titleMatch
        }

        // 范围过滤
        if match && filter.IDRangeLeft != nil {
            id, _ := strconv.ParseInt(comic.GetID(), 10, 64)
            match = match && id >= *filter.IDRangeLeft
        }
        if match && filter.IDRangeRight != nil {
            id, _ := strconv.ParseInt(comic.GetID(), 10, 64)
            match = match && id <= *filter.IDRangeRight
        }

        // NotArchived
        if match && filter.NotArchived != nil && *filter.NotArchived {
            match = match && comic.GetArchivePath() == ""
        }

        // Valid
        if match && filter.Valid != nil {
            match = match && comic.IsValid() == *filter.Valid
        }

        // Status — 通过 Comic 接口的 IsStatus() 判断
        if match && filter.Status != nil {
            match = match && comic.IsStatus() == *filter.Status
        }

        // Deleted
        if match && filter.Deleted != nil {
            match = match && comic.IsDeleted() == *filter.Deleted
        }

        // HasRedirect
        if match && filter.HasRedirect != nil {
            hasRedirect := comic.GetRedirectCID() > 0
            if *filter.HasRedirect {
                match = match && hasRedirect
            } else {
                match = match && !hasRedirect
            }
        }

        // TagIDs
        if match && len(filter.TagIDs) > 0 {
            tagIDSet := make(map[int]bool, len(filter.TagIDs))
            for _, id := range filter.TagIDs {
                tagIDSet[id] = true
            }
            comicMatch := false
            for _, t := range comic.GetTags() {
                if tagIDSet[t.ID] {
                    comicMatch = true
                    break
                }
            }
            match = match && comicMatch
        }

        if match {
            result = append(result, comic)
        }
    }

    // 排序
    sort.Slice(result, func(i, j int) bool {
        return result[i].GetID() < result[j].GetID()
    })

    // 分页
    if filter != nil && filter.Skip > 0 {
        if int(filter.Skip) < len(result) {
            result = result[filter.Skip:]
        } else {
            result = nil
        }
    }
    if filter != nil && filter.Limit > 0 && int64(len(result)) > filter.Limit {
        result = result[:filter.Limit]
    }

    return result, nil
}
```

- [ ] **步骤 4：运行测试验证**

运行：`go test -tags=memory_storage_integration ./pkg/comic/ -run TestMemoryStorage -v`
预期：PASS，应包含所有现有过滤 + 分页测试

- [ ] **步骤 5：Commit**

```bash
git add pkg/comic/storage.go
git commit -m "feat: extend ComicFilter with Status/Deleted/HasRedirect/TitleORPatterns/TagIDs, implement pagination in MemoryStorage.Find"
```

---

### 任务 2：Comic 接口补充访问方法

**文件：**
- 修改：`pkg/comic/comic.go`

**上下文：** MemoryStorage.Find 需要 `IsStatus()`、`IsDeleted()`、`GetRedirectCID()`、`GetTitleEnglish()` 等方法访问漫画字段。

- [ ] **步骤 1：在 ComicImpl 上补充缺失方法**

```go
func (c *ComicImpl) IsStatus() bool { return c.Status }
func (c *ComicImpl) IsDeleted() bool { return c.Deleted }
func (c *ComicImpl) GetRedirectCID() int {
    if c.RedirectTo != nil { return *c.RedirectTo }
    return 0
}
func (c *ComicImpl) GetTitleEnglish() string {
    if c.Title != nil { return c.Title.English }
    return ""
}
func (c *ComicImpl) GetTitleJapanese() string {
    if c.Title != nil { return c.Title.Japanese }
    return ""
}
func (c *ComicImpl) GetTitlePretty() string {
    if c.Title != nil { return c.Title.Pretty }
    return ""
}
```

- [ ] **步骤 2：运行测试验证**

运行：`go build ./...` 确认编译通过

- [ ] **步骤 3：Commit**

```bash
git add pkg/comic/comic.go
git commit -m "feat: add IsStatus/IsDeleted/GetRedirectCID/GetTitle* to ComicImpl"
```

---

### 任务 3：Storage 接口新增 SearchTags/ListTags

**文件：**
- 新增：`pkg/comic/tag.go`
- 修改：`pkg/comic/storage.go`
- 修改：`pkg/comic/storage/mongo.go`

**上下文：** 标签搜索/列表本质是对漫画数据的查询，不单独定义 TagStorage。将其直接加入 `comic.Storage`。

- [ ] **步骤 1：在 pkg/comic/tag.go 中定义 TagInfo**

```go
package comic

// TagInfo 标签信息（从漫画数据推导）
type TagInfo struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Type  string `json:"type"`
    URL   string `json:"url"`
    Count int    `json:"count"`
    Like  bool   `json:"like"`
}
```

- [ ] **步骤 2：在 Storage 接口新增 SearchTags/ListTags**

```go
type Storage interface {
    // ...现有方法...
    SearchTags(ctx context.Context, tagType string, query string, limit int64) ([]TagInfo, int64, error)
    ListTags(ctx context.Context, tagType string, sortType int, skip, limit int64, likedOnly bool) ([]TagInfo, int64, error)
}
```

- [ ] **步骤 3：MemoryStorage 实现标签推导**

在 `MemoryStorage` 结构体新增 `likedTags map[string]bool`。实现 `collectTags()`、`SearchTags()`、`ListTags()` 方法（遍历 m.comics 收集标签 → 去重 → 计数 → 过滤 → 排序 → 分页）。

- [ ] **步骤 4：MongoDB 实现（pkg/comic/storage/mongo.go）**

实现 `SearchTags`/`ListTags`，对接现有 `comicTag` 集合查询。

- [ ] **步骤 5：运行测试验证**

运行：`go test -tags=memory_storage_integration ./pkg/comic/ -v -count=1`
预期：新增标签测试通过

- [ ] **步骤 6：Commit**

```bash
git add pkg/comic/tag.go pkg/comic/storage.go pkg/comic/storage/mongo.go
git commit -m "feat: add SearchTags/ListTags to Storage interface with MemoryStorage tag derivation and MongoDB impl"
```

---

### 任务 4：包装层同步新方法

**文件：**
- 修改：`cmd/server/internal/comic/storage.go`
- 修改：`cmd/server/internal/onecomic/storage.go`

**上下文：** 两个包装层同步新增 `SearchTags`/`ListTags` 代理方法，以及 `toMongoFilter` 的新过滤字段。

- [ ] **步骤 1：在 internal/comic/storage.go 新增 SearchTags/ListTags**

```go
func (s *Storage) SearchTags(ctx context.Context, tagType string, query string, limit int64) ([]comic.TagInfo, int64, error) {
    if s.inner != nil { return s.inner.SearchTags(ctx, tagType, query, limit) }
    tags, err := tag.SearchTags(ctx, tagType, query, limit)
    if err != nil { return nil, 0, err }
    result := make([]comic.TagInfo, len(tags))
    for i, t := range tags {
        result[i] = comic.TagInfo{ID: t.ID, Name: t.Name, Type: t.Type, URL: t.URL, Count: t.Count, Like: t.Like}
    }
    return result, int64(len(result)), nil
}
func (s *Storage) ListTags(ctx context.Context, tagType string, sortType int, skip, limit int64, likedOnly bool) ([]comic.TagInfo, int64, error) {
    if s.inner != nil { return s.inner.ListTags(ctx, tagType, sortType, skip, limit, likedOnly) }
    tags, total, err := tag.AggregateTagList(ctx, tagType, sortType, skip, limit, likedOnly)
    if err != nil { return nil, 0, err }
    result := make([]comic.TagInfo, len(tags))
    for i, t := range tags { result[i] = comic.TagInfo{ID: t.ID, Name: t.Name, Type: t.Type, URL: t.URL, Count: t.Count, Like: t.Like} }
    return result, total, nil
}
```

- [ ] **步骤 2：同步 onecomic/storage.go**

- [ ] **步骤 3：同步 toMongoFilter 新字段**

```go
if filter.Status != nil { mongoFilter["status"] = *filter.Status }
if filter.Deleted != nil { mongoFilter["deleted"] = *filter.Deleted }
// HasRedirect, TitleORPatterns
```

- [ ] **步骤 4：运行测试验证**

运行：`go build ./cmd/server/...` 确认编译通过

- [ ] **步骤 5：Commit**

```bash
git add cmd/server/internal/comic/storage.go cmd/server/internal/onecomic/storage.go
git commit -m "feat: sync SearchTags/ListTags and ComicFilter new fields to Storage wrappers"
```

---

### 任务 5：tag 包接入 DefaultTagLikeStore + DefaultRelationStore

**文件：**
- 新增：`cmd/server/internal/tag/default_store.go`
- 修改：`cmd/server/internal/tag/aggregate.go`
- 修改：`cmd/server/internal/tag/relation.go`
- 修改：`cmd/server/handler/tag_like.go`

**上下文：** 点赞是 tag 的属性，关系是 tag 之间的独立数据。通过轻量包装在测试时注入。

- [ ] **步骤 1：创建 default_store.go**

```go
package tag

import "github.com/cocomhub/cocom/cmd/server/api"

type LikeStore interface {
    Like(ctx context.Context, tagType string, tagID int) error
    Unlike(ctx context.Context, tagType string, tagID int) error
    IsLiked(ctx context.Context, tagType string, tagID int) (bool, error)
}

type RelationStore interface {
    CreateRelation(ctx context.Context, tags []api.TagBrief) (string, error)
    DeleteRelation(ctx context.Context, groupID string) error
    GetRelationsForTag(ctx context.Context, tagType string, tagID int) ([]api.RelationGroup, error)
}

var defaultLikeStore LikeStore
var defaultRelationStore RelationStore
func SetDefaultLikeStore(s LikeStore) { defaultLikeStore = s }
func GetDefaultLikeStore() LikeStore { return defaultLikeStore }
func ResetDefaultLikeStore() { defaultLikeStore = nil }
func SetDefaultRelationStore(s RelationStore) { defaultRelationStore = s }
func GetDefaultRelationStore() RelationStore { return defaultRelationStore }
func ResetDefaultRelationStore() { defaultRelationStore = nil }

// MemoryLikeStore 内存点赞存储
type MemoryLikeStore struct {
    mu    sync.RWMutex
    likes map[string]bool
}
func NewMemoryLikeStore() *MemoryLikeStore { return &MemoryLikeStore{likes: make(map[string]bool)} }
func (s *MemoryLikeStore) Like(_ context.Context, tagType string, tagID int) error { ... }
func (s *MemoryLikeStore) Unlike(_ context.Context, tagType string, tagID int) error { ... }
func (s *MemoryLikeStore) IsLiked(_ context.Context, tagType string, tagID int) (bool, error) { ... }
```

- [ ] **步骤 2：改造 aggregate.go 核心函数**

在每个函数（`SearchTags`、`AggregateTagList`、`AggregateTags`、`GetTags`、`AggregateTagSectionIndices`、`GetSearchUniqueTags`、`GetRelatedTags`、`getComputedRelatedTags`、`GetTagByTypeName`、`GetMaxTagID`）入口处增加 `GetDefaultComicStore()` 检查。

- [ ] **步骤 3：改造 relation.go**

`CreateRelation`、`DeleteRelation`、`GetRelationsForTag` 加 `GetDefaultRelationStore()` 检查路径。

- [ ] **步骤 4：改造 tag_like.go**

`LikeTag`/`UnlikeTag` handler 加 `GetDefaultLikeStore()` 检查路径。

- [ ] **步骤 5：运行测试验证**

运行：`go build ./cmd/server/...` 确认编译通过

- [ ] **步骤 6：Commit**

```bash
git add cmd/server/internal/tag/ cmd/server/handler/tag_like.go
git commit -m "feat: add LikeStore/RelationStore injection to tag package, adapt handlers"
```

---

### 任务 6：改造 SearchAutocomplete handler

**文件：**
- 修改：`cmd/server/handler/search.go`

**上下文：** `SearchAutocomplete` 中 `GetRangeComicInfos` 已有 DefaultStorage 路径，但传入的 `"$or"` + `bson.M` 过滤器无法被 MemoryStorage 识别。需要让 handler 层能用 NewComicFilter 查询。

- [ ] **步骤 1：改造 search.go**

```go
func SearchAutocomplete(w http.ResponseWriter, req *http.Request) {
    ctx := req.Context()
    query := req.URL.Query().Get("q")
    if query == "" { /* 400返回 */ }
    limit := int64(5)
    // ...limit解析...

    // 搜索漫画标题 — 优先使用 DefaultStorage
    if s := comic.GetDefaultStorage(); s != nil {
        filter := comic.NewComicFilter().
            SetLimit(limit).
            SetTitleORPatterns(regexp.QuoteMeta(query))
        comics, err := s.Find(ctx, filter)
        if err != nil {
            slog.ErrorContext(ctx, "search autocomplete failed", ...)
            comics = nil
        }
        // 转换为 AutocompleteComic...
    } else {
        // 原 MongoDB 路径作为 fallback
        escapedQuery := primitive.Regex{...}
        infos, err := comic.GetRangeComicInfos(ctx, limit, 0, "$or", ...)
        // ...
    }

    // 搜索标签 — 通过 tag.SearchTags 已有 DefaultComicStore
    tags, err := tag.SearchTags(ctx, "", query, limit)
    // ...
}
```

- [ ] **步骤 2：运行测试验证**

运行：`go build ./cmd/server/...` 确认编译通过

- [ ] **步骤 3：Commit**

```bash
git add cmd/server/handler/search.go
git commit -m "refactor: SearchAutocomplete uses ComicFilter.TitleORPatterns for MemoryStorage compatibility"
```

---

### 任务 7：改造 handler_test.go 完全去 MongoDB

**文件：**
- 修改：`cmd/server/handler/handler_test.go`
- 修改：`cmd/server/handler/search_test.go`
- 修改：`cmd/server/handler/tags_search_test.go`

**上下文：** 最终目标。移除 MongoDB 依赖，注入 LikeStore 和标签种子数据。

- [ ] **步骤 1：改造 handler_test.go TestMain**

```go
var testMemStorage *comic.MemoryStorage
var testTagLikeStore *tag.MemoryLikeStore

func TestMain(m *testing.M) {
    testMemStorage = comic.NewMemoryStorage()
    internalComic.SetDefaultStorage(testMemStorage)
    testTagLikeStore = tag.NewMemoryLikeStore()
    tag.SetDefaultLikeStore(testTagLikeStore)
    // 写入种子漫画数据，包含标签信息
    // 移除 mongowrap.Init() 和 testMongoAvailable
    os.Exit(m.Run())
}
```

- [ ] **步骤 2：改造 search_test.go**

移除所有 `if !testMongoAvailable { t.Skip(...) }` 行。

- [ ] **步骤 3：改造 tags_search_test.go**

移除 MongoDB skip guards，测试使用内存种子数据。

- [ ] **步骤 4：运行测试验证**

运行：`go test -tags=memory_storage_integration -count=1 ./cmd/server/handler/`
预期：全部 PASS

- [ ] **步骤 5：Commit**

```bash
git add cmd/server/handler/
git commit -m "feat: completely remove MongoDB dependency from handler tests"
```

---

### 任务 8：验证全量测试无 MongoDB 通过

**文件：** 无

- [ ] **步骤 1：运行全量测试**

运行：`go test -tags=memory_storage_integration -count=1 -timeout 5m ./...`
预期：全部 PASS

- [ ] **步骤 2：如果失败，修复**

- [ ] **步骤 3：Commit**

```bash
git add -A
git commit -m "fix: repair tests affected by MemoryStorage migration"
```

---

### 任务 9：扩展 testutil 工厂 + 场景预设

**文件：**
- 修改：`cmd/server/internal/testutil/factory.go`
- 新增：`cmd/server/internal/testutil/scenarios.go`
- 新增：`cmd/server/internal/testutil/generator.go`
- 新增：`cmd/server/handler/testdata.go`

- [ ] **步骤 1：扩展 factory.go 新增选项**

`WithArchived`、`WithRedirect`、`WithDeleted`、`WithStatus`、`WithTags`

- [ ] **步骤 2：创建 scenarios.go**

`HomePageScenario()`、`SearchScenario()`、`TagManagementScenario()`、`ArchiveScenario()`

- [ ] **步骤 3：创建 generator.go**

`GenerateComics(n int, opts ...func(*api.ComicInfo))`、`GenerateTags(baseID int, typ string, n int)`

- [ ] **步骤 4：创建 testdata.go**

`SeedTestData(ctx, store, likeStore)` 导出函数

- [ ] **步骤 5：Commit**

```bash
git add cmd/server/internal/testutil/ cmd/server/handler/testdata.go
git commit -m "feat: expand testutil factories, add scenarios, generators, and SeedTestData"
```

---

### 任务 10：chromedp 独立 module + 辅助库

**文件：**
- 新增：`tests/chromedp/go.mod`
- 新增：`tests/chromedp/main_test.go`
- 新增：`tests/chromedp/helpers/page.go`
- 新增：`tests/chromedp/helpers/selector.go`

- [ ] **步骤 1：创建 go.mod**

```
module github.com/cocomhub/cocom/tests/chromedp
go 1.26
require (
    github.com/cocomhub/cocom v0.0.0
    github.com/chromedp/chromedp v0.11.0
)
replace github.com/cocomhub/cocom => ../../
```

- [ ] **步骤 2：创建 main_test.go（TestMain 启动 Gin 测试服务器 + 注入 MemoryStorage）**

- [ ] **步骤 3：创建 helpers/page.go（Navigate/WaitForVisible/Screenshot/GetText）**

- [ ] **步骤 4：创建 helpers/selector.go（SearchInput / GalleryItem / TagLink 等选择器常量）**

- [ ] **步骤 5：Commit**

```bash
git add tests/chromedp/
git commit -m "feat: add chromedp UI test framework with independent go.mod"
```

---

### 任务 11：chromedp 核心 E2E 测试

**文件：**
- 新增：`tests/chromedp/tests/home_test.go`
- 新增：`tests/chromedp/tests/search_test.go`
- 新增：`tests/chromedp/tests/gallery_test.go`
- 修改：`Makefile`

- [ ] **步骤 1：创建 home_test.go**（首页加载、漫画列表显示）

- [ ] **步骤 2：创建 search_test.go**（搜索输入、自动补全、结果页）

- [ ] **步骤 3：创建 gallery_test.go**（详情页、标签显示、推荐区）

- [ ] **步骤 4：集成 Makefile**

```makefile
test-chromedp:
	cd tests/chromedp && CGO_ENABLED=1 go test -tags=memory_storage_integration -count=1 -v ./...
test-all: test test-chromedp
```

- [ ] **步骤 5：Commit**

```bash
git add tests/chromedp/tests/ Makefile
git commit -m "feat: add chromedp E2E tests for home, search, and gallery"
```

---

### 任务 12：MCP AI 辅助测试场景

**文件：**
- 新增：`tests/mcp/scenarios/home-page.md`
- 新增：`tests/mcp/scenarios/search-flow.md`
- 新增：`tests/mcp/scenarios/gallery-detail.md`
- 新增：`tests/mcp/scenarios/admin-operations.md`
- 新增：`tests/mcp/config.yaml`
- 新增：`tests/mcp/README.md`

- [ ] **步骤 1：创建搜索流程场景文档**

- [ ] **步骤 2：创建其他场景文档**

- [ ] **步骤 3：创建 config.yaml 和 README.md**

- [ ] **步骤 4：Commit**

```bash
git add tests/mcp/
git commit -m "docs: add MCP AI-assisted test scenarios for core user flows"
```
