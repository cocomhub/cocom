# M2：搜索与标签体验升级 — 设计规格

> 基于 2026-06-04 路线图设计的 M2 阶段，范围：cocom 子项目。

---

## 1. 搜索建议/自动补全

### 行为

- 导航栏搜索框输入 ≥2 字符后，200ms 去抖后请求自动补全
- 下拉分两区：上部分匹配的漫画标题，下部分匹配的标签名
- 键盘 ↑↓ 导航选中，Enter 跳转：标题项跳转到 `/g/{cid}/`，标签项跳转到 `/tag/{type}/{name}/`
- Esc 关闭下拉，点击外部关闭
- 最多各展示 5 条（共 10 条）

### 新增 API 端点

```
GET /api/search/autocomplete?q=xxx&limit=5
```

- 搜索 `comicInfo` 集合 `title.english`、`title.japanese`、`title.pretty` 三字段，使用 `$regex` 不区分大小写
- 返回轻量结果，无需完整 ComicInfo

```go
// api/search.go
type AutocompleteRequest struct {
    Query string `form:"q"`
    Limit int    `form:"limit,default=5"`
}

type AutocompleteResponse struct {
    Comics []AutocompleteComic `json:"comics"`
    Tags   []*TagInfo          `json:"tags"`
}
```

- 响应走 `httpwrap` 统一格式：`{head: {...}, body: AutocompleteResponse}`

### 前端

- 新文件：`custom/js/modules/search-autocomplete.js`
- 使用 `autocomplete.js` 的 `bindAutocompleteKeys` 处理键盘导航
- 下拉使用绝对定位浮层，CSS 类名 `.search-autocomplete-dropdown`
- 集成到 `initGalleryPage()` 中

### 涉及文件

- 创建：`cmd/server/api/search.go`（新 API 类型定义）
- 创建：`cmd/server/handler/search.go`（新 Handler）
- 创建：`custom/js/modules/search-autocomplete.js`
- 修改：`cmd/server/handler/init.go`（注册新路由）
- 修改：`custom/css/styles.css`（下拉浮层样式）
- 修改：`cmd/server/view/static/tpl/head.tpl`（搜索框绑定初始化）

---

## 2. 搜索结果高亮

### 行为

- **搜索列表页**（`/search?q=xxx`）：卡片标题中匹配的关键词用 `<mark>` 标黄，服务端渲染（SSR）
- **标签结果页**（`/tag/:type/:name`）：当前标签名在页面标题区域高亮
- 关键词中的特殊正则字符（`.`, `*`, `+` 等）提前转义

### 后端

- 新增模板函数 `HighlightKeyword(text, keyword string) template.HTML`
- 在 `cmd/server/view/` 包中实现
- 将 `text` 中匹配 `keyword` 的子串替换为 `<mark>keyword</mark>`
- 在 `GalleryIndexPage` 中应用到漫画标题

### CSS

```css
mark.search-highlight {
    background: #ffc107;
    color: #000;
    padding: 0 2px;
    border-radius: 2px;
}
```

### 涉及文件

- 修改：`cmd/server/view/static/tpl/index.tpl`（标题区域调用高亮函数）
- 修改：`cmd/server/view/static/tpl/tag_list_result.tpl`（标签名高亮）
- 修改：`custom/css/styles.css`（高亮样式）
- 修改：`cmd/server/view/index.go`（配置 SearchQuery）
- 修改：`cmd/server/view/tag_result.go`（传递 CurTag 高亮）

---

## 3. 标签批量操作增强

### API 增量聚合

- `BatchAddTagToComics` 中的 `tag.AggregateTags(ctx)` 替换为 `tag.UpdateComicTagIncremental(ctx, tagType, tagName, delta int)`
- 增量失败时回退到全量 `AggregateTags`

### 涉及文件

- 修改：`cmd/server/handler/tag_edit.go`（批量添加后增量更新）
- 修改：`cmd/server/internal/tag/aggregate.go`（确认增量接口可用）

---

## 4. 标签聚合 API 分页增强

### 现状

- `GET /api/comic/tags` 已支持 `?skip=N&limit=M` 参数，limit 默认 20
- `GET /api/comic/tags/search` 已有 `limit` 参数（默认 20，最大 100）
- 缺失：`?page=N` 参数（目前只有 `skip`）

### 改动

- 为 `GET /api/comic/tags` 增加 `?page=N` 和 `?page_size=M` 参数支持
- `page_size` 默认 20，最大 100
- 内部转换为 `skip = (page - 1) * page_size`
- 兼容保留 `skip`/`limit` 参数（优先使用 `page`/`page_size` 当它们存在时）

### 涉及文件

- 修改：`cmd/server/handler/tags_agg.go`（解析 page/page_size 参数）

---

## 5. 搜索 API 测试

### 测试范围

- `GET /api/comic/tags/search` 端点测试
- `GET /api/comic/tags/search-unique` 端点测试
- `GET /api/search/autocomplete` 端点测试（新增）
- 使用内存存储模式（`memory_storage_integration` tag）

### 涉及文件

- 创建：`cmd/server/handler/search_test.go`（自动补全端点测试）
- 创建：`cmd/server/handler/tags_search_test.go`（标签搜索端点测试）

---

## 验收标准

- [ ] 搜索框输入 2 个字符后 300ms 内展示下拉建议
- [ ] 搜索结果页中匹配文字有 `<mark>` 高亮
- [ ] 单个标签聚合 API 返回不超过 100 条/页，支持 `?page=` 参数
- [ ] 搜索和标签 API 的测试覆盖率 ≥ 60%

---

## 不包含的范围

- 标签树/关系可视化（推迟到后续里程碑）
- MongoDB 全文搜索索引（`$text`）（推迟到 M3 性能优化）
- 标签聚合 API 分页（已有 `skip`/`limit` 支持，仅补充文档）
- 搜索 API 响应格式改造（沿用 `httpwrap` 已有格式）
