# 多维度推荐 · 自动标签 ID · Admin 漫画对比工具

> 日期：2026-06-05
> 状态：设计稿

---

## 需求一：Gallery Detail 多维度推荐

### 背景

当前 Gallery Detail 页面只有一行 "More Like This" 推荐，基于共同标签 ID 的 `$elemMatch` 查询，随机打乱返回 5 条。如果漫画无标签则降级为重复当前条目。

### 改动内容

#### 推荐维度

按以下 5 个维度各生成一行推荐（去掉 `language`、`category`）：

| 维度 | Tag 类型 | 展示标题 |
|------|---------|---------|
| 同作者 | `artist` | 同作者 · More by Artist |
| 同团体 | `group` | 同团体 · More from Group |
| 同 Parody | `parody` | 同系列 · More from Parody |
| 同角色 | `character` | 同角色 · More by Character |
| 同标签 | `tag` | 同标签 · More Like This |

#### 推荐逻辑

**`cmd/server/internal/comic/recommend.go`** — 新增：

1. **`GetByTagType(ctx, cid int, tagType string, limit int) ([]*api.ComicInfo, error)`**
   - 从当前漫画中提取指定 `tagType` 的标签 ID 列表
   - 使用 MongoDB `$elemMatch` 查询 `comicInfo` 集合中匹配这些标签 ID 的漫画
   - 排除当前 CID（`{"cid": {"$ne": cid}}`）
   - 随机打乱候选项，取前 `limit` 条

2. **`GetRecommendations(ctx, cid int, tags []api.Tag, limit int) map[string][]*api.ComicInfo`**
   - 对 `["artist", "group", "parody", "character", "tag"]` 各维度遍历
   - 每个维度去重：维护一个全局 `seenCIDs map[int]bool`，已出现在其他推荐栏的漫画跳过
   - 排除当前 CID
   - 某维度无结果时不显示该栏（不填充）

#### API 端点

**`GET /api/comic/recommendations?cid={cid}&type={tagType}`**

新增 handler 函数 `GetRecommendationsHandler`（`cmd/server/handler/recommend.go`）。

- `type` 参数：`artist` | `group` | `parody` | `character` | `tag`
- 返回 JSON：

```json
{
  "cid": 12345,
  "type": "artist",
  "limit": 5,
  "results": [
    {"cid": 1001, "title_english": "...", "media_id": "123", "cover_name": "1.jpg", "tags": [...]},
    ...
  ]
}
```

内部调用 `GetByTagType(ctx, cid, tagType, limit)` 获取数据，limit 来自配置 `recommend.limit`（默认 5）。

#### 模板修改

**`cmd/server/view/static/tpl/gallery_detail.tpl`** — 在现有 `#related-container` 附近增加推荐容器：

```html
<div id="recommend-container" data-cid="{{.CID}}">
  <section class="recommend-section" data-recommend-type="artist">
    <div class="recommend-header">
      <h2>同作者 · More by Artist</h2>
      <button class="recommend-refresh" onclick="refreshRecommend(this, 'artist')">
        <i class="fa fa-sync-alt"></i>
      </button>
    </div>
    <div class="recommend-grid">骨架屏...</div>
  </section>
  <!-- group, parody, character, tag 同理 -->
</div>
```

- `data-cid` 记录当前漫画 ID
- 每个 section 的 `data-recommend-type` 记录维度类型
- 初始渲染时，`recommend-grid` 内显示骨架屏（skeleton cards）

#### 前端 JS

**`cmd/server/view/static/custom/js/modules/recommend.js`** — 新建模块：

```js
// 页面加载时自动加载所有推荐
document.addEventListener('DOMContentLoaded', () => {
  const container = document.getElementById('recommend-container');
  if (!container) return;
  const cid = container.dataset.cid;
  document.querySelectorAll('[data-recommend-type]').forEach(section => {
    loadRecommendations(cid, section.dataset.recommendType, section);
  });
});

// 点击刷新图标时重新加载该维度
function refreshRecommend(btn, tagType) {
  const section = btn.closest('[data-recommend-type]');
  const cid = document.getElementById('recommend-container').dataset.cid;
  loadRecommendations(cid, tagType, section);
}

function loadRecommendations(cid, tagType, section) {
  const grid = section.querySelector('.recommend-grid');
  grid.innerHTML = '...'; // 显示骨架屏
  fetch(`/api/comic/recommendations?cid=${cid}&type=${encodeURIComponent(tagType)}`)
    .then(r => r.json())
    .then(data => renderRecommendGrid(grid, data.results))
    .catch(() => { grid.innerHTML = '<div class="empty-state">加载失败</div>'; });
}

function renderRecommendGrid(grid, comics) {
  if (!comics || comics.length === 0) {
    grid.innerHTML = '<div class="empty-state">暂无推荐</div>';
    return;
  }
  grid.innerHTML = comics.map(c => `
    <div class="gallery" data-tags="${c.tags_id_string}">
      <a href="/g/${c.cid}/" class="cover" style="padding:0 0 141.6% 0">
        <img class="lazyload" width="250" height="354"
             data-src="/galleries/${c.media_id}/${c.cover_name}" />
        <div class="caption">${escapeHtml(c.title_english)}</div>
      </a>
    </div>
  `).join('');
}
```

#### GalleryDetail 结构调整

`GalleryDetail` 结构体内不再包含推荐字段，保留原有结构不变。SSR 模板只嵌入一个容器标签和一个加载中的骨架屏。

#### 路由注册

**`cmd/server/handler/init.go`** 新增：
```
GET /api/comic/recommendations → GetRecommendations
```

#### 配置

```yaml
recommend:
  limit: 5  # 每栏推荐数量，默认 5，可配置
```

#### 降级策略

- 某维度没有匹配漫画 → 该栏隐藏（不渲染 `<section>`）
- 漫画本身没有 tags → 所有栏都不显示
- 全局去重后某栏不足 limit 个 → 有多个显示多个（不填充占位）

---

## 需求二：自动标签 ID 分配

### 背景

当前 `api.Tag.ID` 由 nhentai 源提供，用户新增 tag 时无 ID 生成机制。已有 "like" 自定义 tag 使用硬编码 ID `99999`。`UpdateComicTags` 端点中客户端必须自行提供 ID。

### 改动内容

#### ID 生成策略

在 `UpdateComicTags` handler（`cmd/server/handler/tag_edit.go`）中，对客户端传 `ID: 0` 的新增 tag 自动分配 ID：

1. 查询 `comicInfo` 集合中当前最大 tag ID（`$max` aggregation）
2. 新 ID = `max(maxId + 1, 1000000000)`
3. 所有 tag type 统一走此逻辑（不仅是 `custom` 类型）

#### 新增函数

**`cmd/server/internal/tag/aggregate.go`** — 新增 `GetMaxTagID(ctx) (int, error)`：

```go
func GetMaxTagID(ctx context.Context) (int, error) {
    pipeline := []bson.M{
        {"$unwind": "$tags"},
        {"$group": bson.M{"_id": nil, "maxId": bson.M{"$max": "$tags.id"}}},
    }
    // 执行聚合，返回 maxId，如果无结果返回 0
}
```

该函数使用 `mongowrap.Builder.Aggregate()` 在 `comicInfo` 集合上执行聚合管道。

#### 并发安全

考虑到自定义 tag 操作频率低（管理操作），当前不引入分布式锁。如果后续出现并发冲突，可加 `$set` 版本号或 MongoDB 事务。

#### 不受影响的部分

- nhentai 导入的标签（通过探针 `probe.go`）直接写入已有 ID，不经过 `UpdateComicTags`
- 已有标签的 ID 不变

---

## 需求三：Admin 漫画对比工具

### 背景

Admin 页面位于 `GET /admin`，受 `LocalGuard` 中间件保护。当前只有系统设置、缓存控制、服务器关闭、NHComic 校验功能。没有漫画对比和链接管理能力。

### 改动内容

#### 1. 数据模型

在 **`cmd/server/api/comic.go`** — `ComicInfo` 新增字段：

```go
type ComicInfo struct {
    // ... 现有字段
    RedirectTo *int   `json:"redirect_to,omitempty" bson:"redirect_to,omitempty"`
}
```

- `RedirectTo = nil` → 独立漫画（默认）
- `RedirectTo = 12345` → 从属漫画，所有页面访问重定向到 CID 12345

#### 2. 后端 API（`cmd/server/handler/admin.go` 新增）

##### 2.1 `POST /api/admin/comic/compare`

请求：
```json
{"cid1": 1001, "cid2": 1002}
```

处理流程：
1. 通过 `comic.GetComicInfo` 获取两个 comic 的元数据
2. 从磁盘读取每个 comic 的图片文件（`SaveDir()` 下的实际文件）
3. 按文件名对齐（`1.jpg vs 1.jpg`）
4. 逐文件计算 MD5
5. 返回对比结果

响应：
```json
{
  "cid1": {"info": {...}, "pages": [{"page":1, "name":"1.jpg", "md5":"abc...", "exists":true}, ...]},
  "cid2": {"info": {...}, "pages": [...]},
  "comparison": [
    {"page":1, "name":"1.jpg", "md5_match":true, "cid1_md5":"abc", "cid2_md5":"abc"},
    {"page":5, "name":"5.jpg", "md5_match":false, "cid1_md5":"efg", "cid2_md5":"xyz"}
  ],
  "stats": {"total":18, "matched":16, "mismatched":2, "match_ratio":0.889}
}
```

##### 2.2 `POST /api/admin/comic/link`

请求：
```json
{"main_cid": 1001, "sub_cid": 1002}
```

处理：
1. 获取两个 comic 的当前数据
2. 将从属 comic 的 tags 去重后合并到主 comic（按 `(ID, Type)` 去重）
3. 更新主 comic 文档（tags 合并）
4. 更新从属 comic 文档（`RedirectTo = 1001`）

##### 2.3 `POST /api/admin/comic/unlink`

请求：
```json
{"main_cid": 1001, "sub_cid": 1002}
```

处理：
1. 将从属 comic 的 `RedirectTo` 设为 `nil`
2. 不删除已合并的 tags

##### 2.4 `GET /api/admin/comic/links?main_cid=1001&all=false`

- 查询所有 `RedirectTo != nil` 的漫画
- 支持按主 CID 过滤（`main_cid` 参数）
- `all=true` 返回全部链接，`all=false`（默认）返回指定 CID 相关的链接

#### 3. 搜索结果和页面过滤

在以下位置加 `{"redirect_to": nil}` 过滤条件，排除从属漫画：

| 位置 | 文件 | 修改点 |
|------|------|--------|
| 首页列表 | `cmd/server/view/index.go` — `initComicInfos` | 过滤器中加 `redirect_to` 条件 |
| 搜索 | `cmd/server/view/search.go` — `SearchResultPage` | 加 `redirect_to` 过滤 |
| 标签页 | `cmd/server/view/tag_result.go` — `TagResultPage` | 加 `redirect_to` 过滤 |
| Gallery Detail | `cmd/server/view/gallery_detail.go` — `GalleryDetailPage` | 检查 `RedirectTo`，302 到主 comic |
| Gallery Picture | `cmd/server/view/gallery_picture.go` — `GalleryPicturePage` | 同上 |
| 图片服务 | `cmd/server/view/picture.go` — `Picture` | 同上 |

#### 4. Admin 页面 UI

**`cmd/server/view/static/tpl/admin.tpl`** — 新增完整功能区域（在现有容器下方）：

- **CID 输入区** — 两个带自动补全（搜索标题）的输入框 + 对比/交换按钮
- **漫画信息卡** — 并排展示主/从 comic 的 CID、标题、标签、存储路径
- **统计栏** — 总页数、对齐页数、匹配度、匹配/不匹配计数
- **对比表格** — 逐行文件对比（MD5 匹配/不匹配/缺文件），不匹配行支持并排预览
- **并排预览面板** — 左右显示两张图片（调用现有 `/galleries/:cid/:name` 端点）
- **建立从属关系** — 输入主/从 CID，确认弹窗后发起 link API
- **已链接漫画管理** — 表格列出所有链接，支持"本次比较"和"全部链接"切换，每行有取消合并和重新对比按钮
- **取消合并确认弹窗** — 二次确认后发起 unlink API

#### 5. 路由注册

**`cmd/server/handler/init.go`** — 新增 4 个路由（挂载在 `/api/admin/comic/` 下，受 `LocalGuard` 保护）：

```
POST /api/admin/comic/compare  → CompareComics
POST /api/admin/comic/link     → LinkComics
POST /api/admin/comic/unlink   → UnlinkComics
GET  /api/admin/comic/links    → GetLinks
```

#### 6. JS 和 CSS

- **CSS**：追加到 `cmd/server/view/static/custom/css/styles.css`（对比工具专用样式：并排卡片、匹配/不匹配行配色、预览面板等）
- **JS**：追加到 `cmd/server/view/static/custom/js/scripts.js`（对比工具前端交互逻辑，或新建独立模块 `admin-compare.js`）

---

## 文件变更汇总

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `cmd/server/api/comic.go` | 修改 | `ComicInfo` 加 `RedirectTo *int` |
| `cmd/server/internal/comic/recommend.go` | 修改 | 新增 `GetByTagType`、`GetRecommendations`，保留原 `GetMoreLikeThis` |
| `cmd/server/handler/recommend.go` | **新建** | `GetRecommendationsHandler` API |
| `cmd/server/handler/init.go` | 修改 | 注册 5 个新路由（推荐 + 4 个 admin） |
| `cmd/server/view/gallery_detail.go` | 修改 | 移除推荐逻辑，保留 RedirectTo 重定向检查 |
| `cmd/server/view/static/tpl/gallery_detail.tpl` | 修改 | 新增推荐容器（骨架屏 + 刷新按钮） |
| `cmd/server/view/static/custom/js/modules/recommend.js` | **新建** | 前端推荐加载/渲染/刷新逻辑 |
| `internal/config/config.go` | 修改 | 新增 `recommend.limit` 配置 |
| `cmd/server/handler/tag_edit.go` | 修改 | `UpdateComicTags` 中自动分配 ID |
| `cmd/server/internal/tag/aggregate.go` | 修改 | 新增 `GetMaxTagID` |
| `cmd/server/handler/admin.go` | **新建** | 4 个 API handler |
| `cmd/server/handler/init.go` | 修改 | 注册 4 个新路由 |
| `cmd/server/view/index.go` | 修改 | 加 `redirect_to` 过滤 |
| `cmd/server/view/search.go` | 修改 | 加 `redirect_to` 过滤 |
| `cmd/server/view/tag_result.go` | 修改 | 加 `redirect_to` 过滤 |
| `cmd/server/view/gallery_picture.go` | 修改 | RedirectTo 重定向检查 |
| `cmd/server/view/picture.go` | 修改 | RedirectTo 重定向检查 |
| `cmd/server/view/static/tpl/admin.tpl` | 修改 | 新增对比工具 UI |
| `cmd/server/view/static/custom/css/styles.css` | 修改 | 对比工具样式 |
| `cmd/server/view/static/custom/js/scripts.js` | 修改 | 对比工具前端逻辑 |

## 验证方案

1. **多维度推荐**：启动 server，访问漫画 detail 页面，检查 5 个推荐栏是否显示、内容不重复、不含当前漫画
2. **自动标签 ID**：通过 API 创建一个新 tag（ID=0），检查返回的 tag ID ≥ 1000000000；连续创建确认 ID 递增
3. **漫画对比工具**：通过 admin 页面输入两个 CID，确认文件列表、MD5 对比、匹配度统计正确
4. **建立/取消链接**：建立链接后访问从属漫画确认 302 重定向；取消链接后确认恢复正常
5. **搜索过滤**：将从属后的漫画搜索确认不在结果中；首页确认不显示
6. **单元测试**：`go test -tags=memory_storage_integration` 确认不影响现有测试