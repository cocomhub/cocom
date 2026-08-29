# Test Experience Optimization Design

Date: 2026-06-13

## 1. 包级默认存储（Handler Mock 注入）

### 目标
解除 handler 测试对 MongoDB 的依赖，将 `comic.Storage` 接口注入到 `cmd/server/internal/comic` 包的包级函数中，handler 代码零改动。

### 方案
在 `cmd/server/internal/comic/` 中增加默认存储变量，现有所有包级函数内部优先检查默认存储。测试时通过 `SetDefaultStorage(MemoryStorage)` 切换。

### 新增文件

**`cmd/server/internal/comic/default_storage.go`**

```go
package comic

import "github.com/cocomhub/cocom/pkg/comic"

var defaultStorage comic.Storage

func SetDefaultStorage(s comic.Storage)  { defaultStorage = s }
func GetDefaultStorage() comic.Storage   { return defaultStorage }
func ResetDefaultStorage()               { defaultStorage = nil }
```

### 改造的包级函数

| 函数 (文件) | Storage 方法 | 改造方式 |
|---|---|---|
| `GetComicInfo` (comic_info.go) | `Storage.Get` | 调用 defaultStorage.Get 后做 ComicImpl→api.ComicInfo 转换 |
| `UpdateComicInfo` (comic_info.go) | `Storage.Update` | 通过 ComicImpl 桥接，map→ComicImpl→Update |
| `DeleteComicByID` (comic_info.go) | `Storage.ArchiveByID` | 委托 ArchiveByID + 保留文件清理逻辑 |
| `RestoreComicByID` (archive.go) | `Storage.RestoreByID` | 已有 inner 委托模式，直接类似 |
| `CountTotalComicInfos` (comic_info.go) | `Storage.FindTotal` | Filter 转换后委托 |
| `GetRangeComicInfos` (comic_info.go) | `Storage.Find` | Filter 转换后委托 |
| `GetByTagType` (recommend.go) | 新增 `FindByTags(tags, tagType, limit)` | 需新增方法到 `comic.Storage` 接口，MemoryStorage 中实现 |
| `AggregateTagList` (comic_info.go) | 无对应（MongoDB 聚合） | 测试中跳过（纯 aggregation 查询，不走 Storage） |

### 接口扩展

在 `pkg/comic/storage.go` 的 `Storage` 接口中新增方法：

```go
type Storage interface {
    // ... 现有方法 ...
    FindByTags(ctx context.Context, cid int, tags []Tag, tagType string, limit int) ([]Comic, error)
}
```

在 `pkg/comic/memory_storage.go` 的 `MemoryStorage` 中实现。

### handler_test.go 改造

- 移除 `mongowrap.Init()` 和 `testMongoAvailable`
- 在 `TestMain` 中创建 `MemoryStorage`，注入测试数据
- 调用 `internalcomic.SetDefaultStorage(memStorage)`
- 移除所有 `if !testMongoAvailable { t.Skip(...) }`

## 2. CLI 命令测试

### cmd/gallery/ 测试

在 `compare_test.go` 和 `merge_test.go` 中测试：
- 参数验证（非法 cid 返回错误）
- 请求 URL 构建逻辑（提取 `buildCompareRequest` 等可测小函数）

### cmd/ar/ 测试

在 `ar_test.go` 中测试：
- 通过 mock `archivecli.Options` 回调验证归档流程决策逻辑
- `OutputMode` 解析
- `GetSourceDir`/`GetArchiveFilePath` 构造逻辑

### cmd/verify/ 测试

在 `verify_test.go` 中测试：
- 参数解析验证（`--type`, `--interval`, `--priority` 等）
- Cobra 子命令注册检查

## 3. 其他优化

### 3a. 补充空壳测试文件
运行 `scripts/check-test-files.sh` 找出无真实测试用例的包，补充基础测试。

### 3b. 清理 testMongoAvailable
随默认存储改造自动覆盖，删除 handler 测试中所有 `if !testMongoAvailable` 条件。

### 3c. 测试数据工厂
在 `cmd/server/internal/testutil` 中增加数据工厂函数：

```go
func MockComicInfo(cid int, opts ...func(*api.ComicInfo)) *api.ComicInfo
func MockTags(types ...string) []api.Tag
```

## 实施顺序

1. `default_storage.go` 新增 + 接口扩展 (FindByTags)
2. 逐个改造包级函数（GetComicInfo → UpdateComicInfo → DeleteComicByID → ...）
3. 改造 handler_test.go
4. CLI 测试（cmd/ar/, cmd/gallery/, cmd/verify/）
5. 3a 补充空壳 + 3c 数据工厂
6. 验证 `make test` 通过
