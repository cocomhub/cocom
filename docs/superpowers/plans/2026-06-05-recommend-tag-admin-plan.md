# 多维度推荐 · 自动标签 ID · Admin 漫画对比工具 实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 实现三个功能：(1) Gallery Detail 页多维度推荐（Web UI 异步加载 + 刷新按钮），(2) 新增 tag 自动分配 ≥1000000000 的 ID，(3) Admin 漫画对比工具 + 从属链接管理。

**架构：**
- 推荐：SSR 只输出容器骨架屏，前端 JS 异步调用 `GET /api/comic/recommendations` 加载每个维度的推荐数据
- 标签 ID：在 `UpdateComicTags` handler 中为 ID=0 的新 tag 自动调用 `GetMaxTagID` 分配
- 对比工具：新建 `cmd/server/handler/admin.go` 提供 4 个 API，修改 admin.tpl 增加完整 UI，新增 `admin-compare.js` 前端模块

**技术栈：** Go 1.26, Gin, MongoDB, jQuery-like DOM 操作, 前端原生 JS

---

## 文件结构

### 新建文件

| 文件 | 职责 |
|------|------|
| `cmd/server/handler/recommend.go` | `GetRecommendations` handler — 接收 cid+type 参数，调用 recommend.go 查询并返回 JSON |
| `cmd/server/handler/admin.go` | 4 个 API handler：CompareComics, LinkComics, UnlinkComics, GetLinks |
| `cmd/server/view/static/custom/js/modules/recommend.js` | 前端推荐加载/渲染/刷新逻辑 |
| `cmd/server/view/static/custom/js/modules/admin-compare.js` | 前端对比工具交互逻辑 |

### 修改文件

| 文件 | 职责 |
|------|------|
| `cmd/server/api/comic.go` | 加字段 `RedirectTo *int` |
| `cmd/server/internal/comic/recommend.go` | 新增 `GetByTagType` 函数 |
| `cmd/server/handler/init.go` | 注册 5 个新路由（推荐 + 4 个 admin） |
| `cmd/server/handler/tag_edit.go` | `UpdateComicTags` 中增加 ID 自动分配逻辑 |
| `cmd/server/internal/tag/aggregate.go` | 新增 `GetMaxTagID` 函数 |
| `cmd/server/view/gallery_detail.go` | 移除 `MoreLikeThis` 方法，增加 RedirectTo 检查 |
| `cmd/server/view/gallery_detail.tpl` | 新增推荐容器骨架屏区域 |
| `cmd/server/view/index.go` | 加 `redirect_to` 过滤 |
| `cmd/server/view/search.go` | 加 `redirect_to` 过滤 |
| `cmd/server/view/tag_result.go` | 加 `redirect_to` 过滤 |
| `cmd/server/view/gallery_picture.go` | RedirectTo 重定向检查 |
| `cmd/server/view/picture.go` | RedirectTo 重定向检查 |
| `cmd/server/view/static/tpl/admin.tpl` | 新增对比工具 UI |
| `cmd/server/view/static/custom/css/styles.css` | 新增对比工具样式 |

---

## 任务分解

### 任务 1：数据模型 — ComicInfo 添加 RedirectTo 字段

**文件：** `cmd/server/api/comic.go`

- [ ] **步骤 1：在 ComicInfo 结构体中添加 RedirectTo 字段**

在 `cmd/server/api/comic.go` 的 `ComicInfo` 结构体末尾（`Archive` 字段之后）添加：

```go
RedirectTo *int `json:"redirect_to,omitempty" bson:"redirect_to,omitempty"`
```

- [ ] **步骤 2：确认构建通过**

运行：`cd D:/workdir/leon/cocomhub/cocom && go build ./...`
预期：无编译错误

- [ ] **步骤 3：Commit**

```bash
cd D:/workdir/leon/cocomhub/cocom
git add cmd/server/api/comic.go
git commit -m "feat: add RedirectTo field to ComicInfo for subordinate comic redirection"
```

---

### 任务 2：配置 — 新增 recommend.limit

**文件：** `internal/config/config.go`

- [ ] **步骤 1：添加 RecommendLimit 配置**

搜索现有配置文件模式，确认 Viper 配置项在哪里定义。通常在 `internal/config/config.go` 中有类似 `func GetSomething() int { return viper.GetInt("xxx") }` 的模式。新增：

```go
func GetRecommendLimit() int {
    return viper.GetInt("recommend.limit")
}
```

- [ ] **步骤 2：Commit**

```bash
cd D:/workdir/leon/cocomhub/cocom
git add internal/config/config.go
git commit -m "feat: add recommend.limit config for recommendation count per dimension"
```

---

### 任务 3：推荐后端 — GetByTagType

**文件：** `cmd/server/internal/comic/recommend.go`

- [ ] **步骤 1：在推荐逻辑文件中新增 GetByTagType**

在现有 `GetMoreLikeThis` 函数下方添加：

```go
// GetByTagType 根据 tag 类型获取推荐漫画
// 从当前漫画提取指定 tagType 的标签 ID，查询有相同标签的漫画，排除当前漫画并随机打乱
func GetByTagType(ctx context.Context, cid int, tags api.Tags, tagType string, limit int) (infos []*api.ComicInfo, err error) {
	infos = []*api.ComicInfo{}
	if limit <= 0 {
		return infos, nil
	}

	// 提取该类型的标签 ID
	var idList []int
	for _, t := range tags {
		if t.Type == tagType && t.ID != 0 {
			idList = append(idList, t.ID)
		}
	}

	// 如果没有该类型的标签，返回空
	if len(idList) == 0 {
		return infos, nil
	}

	// 查询有相同标签的其他漫画
	candidateLimit := int64(max(limit*4, limit))
	builder := mongo.ComicInfoBuilder().
		FilterKV("cid", bson.M{"$ne": cid}).
		FilterKV("tags", bson.M{"$elemMatch": bson.M{"id": bson.M{"$in": idList}}}).
		SortKV("cid", -1).
		Limit(candidateLimit)

	if err = builder.All(ctx, &infos); err != nil {
		return nil, err
	}
	if len(infos) == 0 {
		infos = []*api.ComicInfo{}
		return infos, nil
	}

	// 随机打乱并截取
	util.Shuffle(len(infos), func(i, j int) { infos[i], infos[j] = infos[j], infos[i] })
	if len(infos) > limit {
		infos = infos[:limit]
	}
	return infos, nil
}
```

注意使用与现有代码相同的包 `util`（`github.com/cocomhub/cocom/pkg/util`）。

- [ ] **步骤 2：构建验证**

运行：`cd D:/workdir/leon/cocomhub/cocom && go build ./cmd/server/...`
预期：无编译错误

- [ ] **步骤 3：Commit**

```bash
cd D:/workdir/leon/cocomhub/cocom
git add cmd/server/internal/comic/recommend.go
git commit -m "feat: add GetByTagType for multi-dimension recommendations"
```

---

### 任务 4：推荐 API Handler

**文件：** 新建 `cmd/server/handler/recommend.go`

- [ ] **步骤 1：创建 handler 文件**

```go
package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/internal/config"
	"github.com/cocomhub/cocom/pkg/httpwrap"
	"github.com/gin-gonic/gin"
)

// GetRecommendations 返回指定维度的推荐漫画
// GET /api/comic/recommendations?cid=12345&type=artist
func GetRecommendations(c *gin.Context) {
	ctx := c.Request.Context()

	cidStr := c.Query("cid")
	cid, err := strconv.Atoi(cidStr)
	if err != nil || cid <= 0 {
		httpwrap.GinRespondError(c, http.StatusBadRequest, httpwrap.ErrCodeInvalid, "invalid cid")
		return
	}

	tagType := c.Query("type")
	validTypes := map[string]bool{"artist": true, "group": true, "parody": true, "character": true, "tag": true}
	if !validTypes[tagType] {
		httpwrap.GinRespondError(c, http.StatusBadRequest, httpwrap.ErrCodeInvalid, "invalid type, must be one of: artist, group, parody, character, tag")
		return
	}

	// 获取漫画信息以获取标签列表
	info := api.ComicInfo{}
	if err := comic.GetComicInfo(ctx, cid, &info); err != nil {
		slog.ErrorContext(ctx, "GetRecommendations: GetComicInfo failed",
			slog.Int("cid", cid),
			slog.String("errmsg", err.Error()))
		httpwrap.GinRespondError(c, http.StatusInternalServerError, httpwrap.ErrCodeServer, "get comic info failed")
		return
	}

	limit := config.GetRecommendLimit()
	if limit <= 0 {
		limit = 5
	}

	infos, err := comic.GetByTagType(ctx, cid, info.Tags, tagType, limit)
	if err != nil {
		slog.ErrorContext(ctx, "GetRecommendations: GetByTagType failed",
			slog.Int("cid", cid),
			slog.String("type", tagType),
			slog.String("errmsg", err.Error()))
		httpwrap.GinRespondError(c, http.StatusInternalServerError, httpwrap.ErrCodeServer, "get recommendations failed")
		return
	}

	// 检查是否存在 `tags_id_string` 方法，如果有的 `ComicInfo` 没有暴露 Tags.IdString，直接在 handler 中构建
	type recommResult struct {
		CID           int        `json:"cid"`
		TitleEnglish  string     `json:"title_english"`
		MediaID       string     `json:"media_id"`
		CoverName     string     `json:"cover_name"`
		TagsIDString  string     `json:"tags_id_string,omitempty"`
		NumPages      int        `json:"num_pages,omitempty"`
	}

	results := make([]recommResult, 0, len(infos))
	for _, item := range infos {
		results = append(results, recommResult{
			CID:          item.CID,
			TitleEnglish: item.Title.English,
			MediaID:      fmt.Sprint(item.CID), // ShowMediaId 逻辑
			CoverName:    item.Images.CoverName(),
			TagsIDString: item.Tags.IdString(),
			NumPages:     item.NumPages,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"cid":     cid,
		"type":    tagType,
		"limit":   limit,
		"results": results,
	})
}
```

注意使用 `fmt` 包，需要添加到 import 列表：
```go
import (
	"fmt"
	...
)
```

- [ ] **步骤 2：构建验证**

运行：`cd D:/workdir/leon/cocomhub/cocom && go build ./cmd/server/...`
预期：无编译错误

- [ ] **步骤 3：Commit**

```bash
cd D:/workdir/leon/cocomhub/cocom
git add cmd/server/handler/recommend.go
git commit -m "feat: add GET /api/comic/recommendations endpoint for async recommendation loading"
```

---

### 任务 5：注册推荐路由

**文件：** `cmd/server/handler/init.go`

- [ ] **步骤 1：在 Init 函数中添加路由**

在 `cmd/server/handler/init.go` 的 `Init` 函数中，在现有路由注册之后添加：

```go
r.GET("/api/comic/recommendations", GetRecommendations)
```

注意该 handler 是 Gin handler（`gin.HandlerFunc`），不是 `http.HandlerFunc`，所以不要用 `gin.WrapF`。

- [ ] **步骤 2：构建验证**

运行：`cd D:/workdir/leon/cocomhub/cocom && go build ./cmd/server/...`
预期：无编译错误

- [ ] **步骤 3：Commit**

```bash
cd D:/workdir/leon/cocomhub/cocom
git add cmd/server/handler/init.go
git commit -m "feat: register GET /api/comic/recommendations route"
```

---

### 任务 6：模板 — 推荐容器骨架屏

**文件：** `cmd/server/view/static/tpl/gallery_detail.tpl`

- [ ] **步骤 1：在 gallery_detail.tpl 的现有 More Like This 区域后新增推荐容器**

搜索模板中现有 `#related-container` 的位置（约第 158-170 行），在其后追加：

```html
<!-- 多维推荐容器（异步加载） -->
<div id="recommend-container" data-cid="{{.CID}}" style="display:none;">
  <section class="recommend-section" data-recommend-type="artist">
    <div class="recommend-header">
      <h2>同作者 · More by Artist</h2>
      <button class="btn btn-secondary btn-sm recommend-refresh" onclick="refreshRecommend(this, 'artist')" title="重新获取">
        <i class="fa fa-sync-alt"></i>
      </button>
    </div>
    <div class="recommend-grid">
      <div class="skeleton-grid">
        {{range $i := seq 5}}<div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>{{end}}
      </div>
    </div>
  </section>
  <section class="recommend-section" data-recommend-type="group">
    <div class="recommend-header">
      <h2>同团体 · More from Group</h2>
      <button class="btn btn-secondary btn-sm recommend-refresh" onclick="refreshRecommend(this, 'group')" title="重新获取">
        <i class="fa fa-sync-alt"></i>
      </button>
    </div>
    <div class="recommend-grid">
      <div class="skeleton-grid">
        {{range $i := seq 5}}<div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>{{end}}
      </div>
    </div>
  </section>
  <section class="recommend-section" data-recommend-type="parody">
    <div class="recommend-header">
      <h2>同系列 · More from Parody</h2>
      <button class="btn btn-secondary btn-sm recommend-refresh" onclick="refreshRecommend(this, 'parody')" title="重新获取">
        <i class="fa fa-sync-alt"></i>
      </button>
    </div>
    <div class="recommend-grid">
      <div class="skeleton-grid">
        {{range $i := seq 5}}<div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>{{end}}
      </div>
    </div>
  </section>
  <section class="recommend-section" data-recommend-type="character">
    <div class="recommend-header">
      <h2>同角色 · More by Character</h2>
      <button class="btn btn-secondary btn-sm recommend-refresh" onclick="refreshRecommend(this, 'character')" title="重新获取">
        <i class="fa fa-sync-alt"></i>
      </button>
    </div>
    <div class="recommend-grid">
      <div class="skeleton-grid">
        {{range $i := seq 5}}<div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>{{end}}
      </div>
    </div>
  </section>
  <section class="recommend-section" data-recommend-type="tag">
    <div class="recommend-header">
      <h2>同标签 · More Like This</h2>
      <button class="btn btn-secondary btn-sm recommend-refresh" onclick="refreshRecommend(this, 'tag')" title="重新获取">
        <i class="fa fa-sync-alt"></i>
      </button>
    </div>
    <div class="recommend-grid">
      <div class="skeleton-grid">
        {{range $i := seq 5}}<div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>{{end}}
      </div>
    </div>
  </section>
</div>
```

注意：需要在模板的 `funcMap` 中确认 `seq` 是否存在。如果不存在，新增一个简单的 `seq` 模板函数（`func seq(n int) []int { r := make([]int, n); for i := 0; i < n; i++ { r[i] = i } return r }`），或者在模板中用硬编码 5 个重复骨架屏代替。

- [ ] **步骤 2：如果必要，在 view/init.go 的 funcMap 中添加 seq 函数**

- [ ] **步骤 3：Commit**

```bash
cd D:/workdir/leon/cocomhub/cocom
git add cmd/server/view/static/tpl/gallery_detail.tpl
git commit -m "feat: add recommendation container with skeleton screens to gallery detail template"
```

---

### 任务 7：前端推荐模块

**文件：** 新建 `cmd/server/view/static/custom/js/modules/recommend.js`

- [ ] **步骤 1：创建 recommend.js**

```javascript
// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

(function () {
  'use strict';

  // 页面加载后自动加载所有推荐
  document.addEventListener('DOMContentLoaded', function () {
    var container = document.getElementById('recommend-container');
    if (!container) return;
    var cid = container.getAttribute('data-cid');
    if (!cid) return;
    container.style.display = ''; // 显示容器
    var sections = container.querySelectorAll('[data-recommend-type]');
    sections.forEach(function (section) {
      var type = section.getAttribute('data-recommend-type');
      loadRecommendations(cid, type, section);
    });
  });

  /**
   * 刷新指定维度的推荐
   * @param {HTMLElement} btn 点击的刷新按钮
   * @param {string} tagType 推荐维度类型
   */
  window.refreshRecommend = function (btn, tagType) {
    var section = btn.closest('[data-recommend-type]');
    if (!section) return;
    var container = document.getElementById('recommend-container');
    if (!container) return;
    var cid = container.getAttribute('data-cid');
    loadRecommendations(cid, tagType, section);
  };

  /**
   * 从 API 加载推荐数据
   * @param {number|string} cid 当前漫画 ID
   * @param {string} tagType 推荐维度类型
   * @param {HTMLElement} section 推荐区域 DOM 元素
   */
  function loadRecommendations(cid, tagType, section) {
    var grid = section.querySelector('.recommend-grid');
    if (!grid) return;

    // 显示骨架屏
    grid.innerHTML =
      '<div class="skeleton-grid">' +
        '<div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>'.repeat(5) +
      '</div>';

    fetch('/api/comic/recommendations?cid=' + encodeURIComponent(cid) + '&type=' + encodeURIComponent(tagType))
      .then(function (resp) {
        if (!resp.ok) throw new Error('HTTP ' + resp.status);
        return resp.json();
      })
      .then(function (data) {
        renderRecommendGrid(grid, data.results || []);
      })
      .catch(function () {
        grid.innerHTML = '<div class="empty-state"><i class="fa fa-exclamation-circle"></i><p>加载失败，点击刷新重试</p></div>';
      });
  }

  /**
   * 渲染推荐网格
   * @param {HTMLElement} grid 网格容器
   * @param {Array} comics 漫画列表
   */
  function renderRecommendGrid(grid, comics) {
    if (!comics || comics.length === 0) {
      grid.innerHTML = '<div class="empty-state"><i class="fa fa-inbox"></i><p>暂无推荐</p></div>';
      return;
    }
    var html = '';
    comics.forEach(function (c) {
      html +=
        '<div class="gallery" data-tags="' + (c.tags_id_string || '') + '">' +
          '<a href="/g/' + c.cid + '/" class="cover" style="padding:0 0 141.6% 0">' +
            '<img class="lazyload" width="250" height="354" ' +
                 'data-src="/galleries/' + c.media_id + '/' + c.cover_name + '" />' +
            '<div class="caption">' + escapeHtml(c.title_english || '') + '</div>' +
          '</a>' +
        '</div>';
    });
    grid.innerHTML = html;

    // 触发 lazyload（如果页面有 lazyload 机制）
    if (window.lazySizes && lazySizes.init) {
      lazySizes.init();
    }
  }

  /**
   * HTML 转义
   */
  function escapeHtml(str) {
    if (!str) return '';
    var div = document.createElement('div');
    div.appendChild(document.createTextNode(str));
    return div.innerHTML;
  }
})();
```

- [ ] **步骤 2：在 head.common.tpl 中引用 recommend.js**

搜索 `head.common.tpl` 中 JS 引用的位置，在 `search-autocomplete.js` 之后添加：

```html
<script src="/static/custom/js/modules/recommend.js"></script>
```

- [ ] **步骤 3：验证构建**

运行：`cd D:/workdir/leon/cocomhub/cocom && go build ./...`
预期：无编译错误

- [ ] **步骤 4：Commit**

```bash
cd D:/workdir/leon/cocomhub/cocom
git add cmd/server/view/static/custom/js/modules/recommend.js cmd/server/view/static/tpl/head.common.tpl
git commit -m "feat: add recommend.js for async recommendation loading with refresh support"
```

---

### 任务 8：清理 GalleryDetail — 删除 SSR MoreLikeThis，增加 RedirectTo 检查

**文件：** `cmd/server/view/gallery_detail.go`、`cmd/server/view/gallery_picture.go`、`cmd/server/view/picture.go`

- [ ] **步骤 1：删除 MoreLikeThis 方法**

从 `cmd/server/view/gallery_detail.go` 中删除 `MoreLikeThis()` 方法（第 108-131 行），以及相关的 `import` 中不再需要的包（`context`、`fmt`、「internal/comic」等，确认保留需要被其他方法使用的包）。

- [ ] **步骤 2：在 GalleryDetailPage 中添加 RedirectTo 重定向检查**

在 `GalleryDetailPage` 函数中，获取漫画信息后，在页面渲染之前加入：

```go
// 检查是否有重定向（从属漫画）
if info.RedirectTo != nil && *info.RedirectTo > 0 {
    c.Redirect(http.StatusFound, fmt.Sprintf("/g/%d/", *info.RedirectTo))
    return
}
```

- [ ] **步骤 3：在 GalleryPicturePage 中添加 RedirectTo 重定向检查**

在 `cmd/server/view/gallery_picture.go` 的 `GalleryPicturePage` 函数中，类似的模式，在获取 `ComicInfo` 后检查 `RedirectTo` 并 302 到主 comic。

- [ ] **步骤 4：在 Picture handler 中添加 RedirectTo 重定向检查**

在 `cmd/server/view/picture.go` 的 `Picture` 函数中，在获取 `ComicInfo` 后检查 `RedirectTo`，302 到主 comic 的图片路径。

注意：图片重定向需要将图片路径中的 CID 替换为主 comic 的 CID：
```go
if info.RedirectTo != nil && *info.RedirectTo > 0 {
    c.Redirect(http.StatusFound, fmt.Sprintf("/galleries/%d/%s", *info.RedirectTo, c.Param("name")))
    return
}
```

- [ ] **步骤 5：构建验证**

运行：`cd D:/workdir/leon/cocomhub/cocom && go build ./cmd/server/...`
预期：无编译错误

- [ ] **步骤 6：Commit**

```bash
cd D:/workdir/leon/cocomhub/cocom
git add cmd/server/view/gallery_detail.go cmd/server/view/gallery_picture.go cmd/server/view/picture.go
git commit -m "feat: remove SSR MoreLikeThis, add RedirectTo redirect check in gallery views"
```

---

### 任务 9：自动标签 ID — GetMaxTagID

**文件：** `cmd/server/internal/tag/aggregate.go`

- [ ] **步骤 1：添加 GetMaxTagID 函数**

在 `aggregate.go` 末尾添加：

```go
// GetMaxTagID 查询 comicInfo 集合中所有 tag 的最大 ID
// 返回当前最大 ID，如果没有任何 tag 则返回 0
func GetMaxTagID(ctx context.Context) (int, error) {
	type maxResult struct {
		MaxID int `bson:"maxId"`
	}
	var results []maxResult
	pipe := []bson.M{
		{"$unwind": "$tags"},
		{"$group": bson.M{"_id": nil, "maxId": bson.M{"$max": "$tags.id"}}},
	}
	if err := mongo.ComicInfoBuilder().Aggregate(ctx, pipe, &results); err != nil {
		return 0, err
	}
	if len(results) == 0 {
		return 0, nil
	}
	return results[0].MaxID, nil
}
```

- [ ] **步骤 2：构建验证**

运行：`cd D:/workdir/leon/cocomhub/cocom && go build ./cmd/server/...`
预期：无编译错误

- [ ] **步骤 3：Commit**

```bash
cd D:/workdir/leon/cocomhub/cocom
git add cmd/server/internal/tag/aggregate.go
git commit -m "feat: add GetMaxTagID for auto-assigning tag IDs"
```

---

### 任务 10：自动标签 ID — UpdateComicTags 自动分配

**文件：** `cmd/server/handler/tag_edit.go`

- [ ] **步骤 1：在 UpdateComicTags 中添加 ID 自动分配逻辑**

在 `UpdateComicTags` 函数中，在处理 `updateReq.Added` 之前添加 ID 分配步骤（在 `diff` 初始化之后，在 `// 添加 tag` 循环之前）：

```go
// 自动分配 ID：对 ID == 0 的新增 tag 从 1000000000 起始分配
needAssign := false
for _, at := range updateReq.Added {
    if at.ID == 0 {
        needAssign = true
        break
    }
}
if needAssign {
    maxID, err := tag.GetMaxTagID(ctx)
    if err != nil {
        slog.WarnContext(ctx, "GetMaxTagID failed, using 1000000000 as base", slog.String("errmsg", err.Error()))
        maxID = 0
    }
    nextID := max(maxID+1, 1000000000)
    for i := range updateReq.Added {
        if updateReq.Added[i].ID == 0 {
            updateReq.Added[i].ID = nextID
            nextID++
        }
    }
}
```

注意放在 `// 添加 tag：去重后追加` 之前，确保传入的 Tag 已经有了分配的 ID。

- [ ] **步骤 2：构建验证**

运行：`cd D:/workdir/leon/cocomhub/cocom && go build ./cmd/server/...`
预期：无编译错误

- [ ] **步骤 3：Commit**

```bash
cd D:/workdir/leon/cocomhub/cocom
git add cmd/server/handler/tag_edit.go
git commit -m "feat: auto-assign tag IDs >= 1000000000 for new tags in UpdateComicTags"
```

---

### 任务 11：Admin API Handler

**文件：** 新建 `cmd/server/handler/admin.go`

- [ ] **步骤 1：创建 admin.go 文件**

```go
package handler

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/cache"
	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/cmd/server/internal/tag"
	"github.com/cocomhub/cocom/pkg/httpwrap"
)

// ---------- Compare ----------

type compareRequest struct {
	CID1 int `json:"cid1"`
	CID2 int `json:"cid2"`
}

type pageInfo struct {
	Page   int    `json:"page"`
	Name   string `json:"name"`
	MD5    string `json:"md5"`
	Exists bool   `json:"exists"`
}

type comparisonRow struct {
	Page      int    `json:"page"`
	Name      string `json:"name"`
	MD5Match  bool   `json:"md5_match"`
	CID1MD5   string `json:"cid1_md5"`
	CID2MD5   string `json:"cid2_md5"`
}

type compareStats struct {
	Total      int     `json:"total"`
	Matched    int     `json:"matched"`
	Mismatched int     `json:"mismatched"`
	MatchRatio float64 `json:"match_ratio"`
}

// CompareComics 对比两个漫画的图片文件
// POST /api/admin/comic/compare
func CompareComics(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	var cr compareRequest
	if err := json.NewDecoder(req.Body).Decode(&cr); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "invalid request body")
		return
	}
	if cr.CID1 <= 0 || cr.CID2 <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "cid1 and cid2 are required")
		return
	}

	// 获取两个漫画的元数据
	info1, info2, err := getTwoComicInfos(ctx, cr.CID1, cr.CID2)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, err.Error())
		return
	}

	// 读取文件列表并计算 MD5
	pages1, err := readComicPages(cr.CID1, info1.SaveDir())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("read pages for cid %d failed: %s", cr.CID1, err))
		return
	}
	pages2, err := readComicPages(cr.CID2, info2.SaveDir())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("read pages for cid %d failed: %s", cr.CID2, err))
		return
	}

	// 按文件名对齐比较
	comparison, stats := alignAndCompare(pages1, pages2)

	resp := gin.H{
		"cid1": gin.H{
			"info":  info1,
			"pages": pages1,
		},
		"cid2": gin.H{
			"info":  info2,
			"pages": pages2,
		},
		"comparison": comparison,
		"stats":      stats,
	}

	httpwrap.ResponseSucc(ctx, w, resp)
}

// ---------- Link ----------

type linkRequest struct {
	MainCID int `json:"main_cid"`
	SubCID  int `json:"sub_cid"`
}

// LinkComics 建立从属关系
// POST /api/admin/comic/link
func LinkComics(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	var lr linkRequest
	if err := json.NewDecoder(req.Body).Decode(&lr); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "invalid request body")
		return
	}
	if lr.MainCID <= 0 || lr.SubCID <= 0 || lr.MainCID == lr.SubCID {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "main_cid and sub_cid must be positive and different")
		return
	}

	info1, info2, err := getTwoComicInfos(ctx, lr.MainCID, lr.SubCID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, err.Error())
		return
	}

	// 将从属 comic 的 tags 合并到主 comic（按 id+type 去重）
	existingTags := make(map[string]bool)
	for _, t := range info1.Tags {
		key := fmt.Sprintf("%s:%d", t.Type, t.ID)
		existingTags[key] = true
	}
	for _, t := range info2.Tags {
		key := fmt.Sprintf("%s:%d", t.Type, t.ID)
		if !existingTags[key] {
			info1.Tags = append(info1.Tags, t)
			existingTags[key] = true
		}
	}

	// 更新主 comic 的 tags
	m1, err := util.ToMap(info1)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, "encode main comic info failed")
		return
	}
	if err := comic.UpdateComicInfo(ctx, lr.MainCID, m1); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, "update main comic info failed")
		return
	}

	// 设置从属 comic 的 RedirectTo
	redirectTo := lr.MainCID
	info2.RedirectTo = &redirectTo
	m2, err := util.ToMap(info2)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, "encode sub comic info failed")
		return
	}
	if err := comic.UpdateComicInfo(ctx, lr.SubCID, m2); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, "update sub comic info failed")
		return
	}

	cache.Reset()

	httpwrap.ResponseSucc(ctx, w, gin.H{
		"main_cid": lr.MainCID,
		"sub_cid":  lr.SubCID,
		"status":   "linked",
	})
}

// UnlinkComics 取消从属关系
// POST /api/admin/comic/unlink
func UnlinkComics(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	var lr linkRequest
	if err := json.NewDecoder(req.Body).Decode(&lr); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "invalid request body")
		return
	}

	info := api.ComicInfo{}
	if err := comic.GetComicInfo(ctx, lr.SubCID, &info); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, "get sub comic info failed")
		return
	}

	// 清除 RedirectTo
	info.RedirectTo = nil
	m, err := util.ToMap(info)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, "encode comic info failed")
		return
	}
	if err := comic.UpdateComicInfo(ctx, lr.SubCID, m); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, "update comic info failed")
		return
	}

	cache.Reset()

	httpwrap.ResponseSucc(ctx, w, gin.H{
		"sub_cid": lr.SubCID,
		"status":  "unlinked",
	})
}

// GetLinks 获取已链接的漫画列表
// GET /api/admin/comic/links?main_cid=1001&all=false
func GetLinks(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	mainCIDStr := req.URL.Query().Get("main_cid")
	all := req.URL.Query().Get("all") == "true"

	// 构建查询过滤器
	filter := bson.M{"redirect_to": bson.M{"$ne": nil}}
	if !all && mainCIDStr != "" {
		mainCID, err := strconv.Atoi(mainCIDStr)
		if err == nil && mainCID > 0 {
			filter["redirect_to"] = mainCID
		}
	}

	type linkedComic struct {
		CID        int    `bson:"cid"`
		RedirectTo int    `bson:"redirect_to"`
		TitleEnglish string `bson:"title.english"`
	}

	var comics []linkedComic
	builder := mongo.ComicInfoBuilder().
		FilterKV("redirect_to", filter["redirect_to"]).
		SortKV("cid", 1).
		NoLimit()

	if err := builder.All(ctx, &comics); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, "query links failed")
		return
	}

	// 组装响应
	type linkItem struct {
		SubCID  int    `json:"sub_cid"`
		SubTitle string `json:"sub_title"`
		MainCID int    `json:"main_cid"`
	}

	links := make([]linkItem, 0, len(comics))
	for _, c := range comics {
		links = append(links, linkItem{
			SubCID:   c.CID,
			SubTitle: c.TitleEnglish,
			MainCID:  c.RedirectTo,
		})
	}

	httpwrap.ResponseSucc(ctx, w, gin.H{
		"links": links,
		"total": len(links),
	})
}

// ---------- Helpers ----------

func getTwoComicInfos(ctx context.Context, cid1, cid2 int) (*api.ComicInfo, *api.ComicInfo, error) {
	info1 := api.ComicInfo{}
	if err := comic.GetComicInfo(ctx, cid1, &info1); err != nil {
		return nil, nil, fmt.Errorf("get cid %d info failed: %w", cid1, err)
	}
	info2 := api.ComicInfo{}
	if err := comic.GetComicInfo(ctx, cid2, &info2); err != nil {
		return nil, nil, fmt.Errorf("get cid %d info failed: %w", cid2, err)
	}
	return &info1, &info2, nil
}

func readComicPages(cid int, saveDir string) ([]pageInfo, error) {
	entries, err := os.ReadDir(saveDir)
	if err != nil {
		// 如果目录不存在，返回空列表
		if os.IsNotExist(err) {
			return []pageInfo{}, nil
		}
		return nil, err
	}

	var pages []pageInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" && ext != ".webp" {
			continue
		}
		fullPath := filepath.Join(saveDir, entry.Name())
		md5sum, err := fileMD5(fullPath)
		if err != nil {
			slog.Warn("readComicPages: md5 failed", slog.String("path", fullPath), slog.String("errmsg", err.Error()))
			md5sum = ""
		}
		pages = append(pages, pageInfo{
			Page:   0, // 后面排序赋值
			Name:   entry.Name(),
			MD5:    md5sum,
			Exists: true,
		})
	}

	// 按文件名排序（自然顺序）
	sort.Slice(pages, func(i, j int) bool {
		return pages[i].Name < pages[j].Name
	})
	for i := range pages {
		pages[i].Page = i + 1
	}
	return pages, nil
}

func fileMD5(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func alignAndCompare(pages1, pages2 []pageInfo) ([]comparisonRow, compareStats) {
	// 构建文件名 → pageInfo 映射
	m1 := make(map[string]pageInfo)
	for _, p := range pages1 {
		m1[p.Name] = p
	}
	m2 := make(map[string]pageInfo)
	for _, p := range pages2 {
		m2[p.Name] = p
	}

	// 取所有文件名并排序
	allNames := make(map[string]bool)
	for _, p := range pages1 {
		allNames[p.Name] = true
	}
	for _, p := range pages2 {
		allNames[p.Name] = true
	}
	var names []string
	for n := range allNames {
		names = append(names, n)
	}
	sort.Strings(names)

	var comparison []comparisonRow
	stats := compareStats{}

	for _, name := range names {
		p1, ok1 := m1[name]
		p2, ok2 := m2[name]
		row := comparisonRow{
			Name: name,
		}

		if ok1 && ok2 {
			row.Page = 0 // page 由前端展示
			row.CID1MD5 = p1.MD5
			row.CID2MD5 = p2.MD5
			row.MD5Match = p1.MD5 == p2.MD5
			stats.Total++
			if row.MD5Match {
				stats.Matched++
			} else {
				stats.Mismatched++
			}
		} else if ok1 {
			row.CID1MD5 = p1.MD5
			row.CID2MD5 = ""
			row.MD5Match = false
			stats.Total++
			stats.Mismatched++
		} else {
			row.CID1MD5 = ""
			row.CID2MD5 = p2.MD5
			row.MD5Match = false
			stats.Total++
			stats.Mismatched++
		}
		comparison = append(comparison, row)
	}

	if stats.Total > 0 {
		stats.MatchRatio = float64(stats.Matched) / float64(stats.Total)
	}
	return comparison, stats
}
```

注意导入 `github.com/cocomhub/cocom/pkg/util` 和 `go.mongodb.org/mongo-driver/bson`，以及 `github.com/gin-gonic/gin`（`gin.H` 需要）。

`gin.H` 被用于构建 JSON 响应，需要 import `github.com/gin-gonic/gin`。

但 `httpwrap.ResponseSucc` 接收的是 `any`，也可以用 `map[string]any`。建议统一用 `map[string]any` 避免引入 gin 包：

```go
httpwrap.ResponseSucc(ctx, w, map[string]any{...})
```

- [ ] **步骤 2：构建验证**

运行：`cd D:/workdir/leon/cocomhub/cocom && go build ./cmd/server/...`
预期：无编译错误

- [ ] **步骤 3：Commit**

```bash
cd D:/workdir/leon/cocomhub/cocom
git add cmd/server/handler/admin.go
git commit -m "feat: add admin comic compare/link/unlink/links API handlers"
```

---

### 任务 12：注册 Admin 路由

**文件：** `cmd/server/handler/init.go`

- [ ] **步骤 1：在 Init 函数中添加 admin 路由**

```go
// Admin 漫画对比工具（需 LocalGuard 保护，已在 /admin 路由层处理）
r.POST("/api/admin/comic/compare", gin.WrapF(CompareComics))
r.POST("/api/admin/comic/link", gin.WrapF(LinkComics))
r.POST("/api/admin/comic/unlink", gin.WrapF(UnlinkComics))
r.GET("/api/admin/comic/links", gin.WrapF(GetLinks))
```

- [ ] **步骤 2：构建验证**

运行：`cd D:/workdir/leon/cocomhub/cocom && go build ./cmd/server/...`
预期：无编译错误

- [ ] **步骤 3：Commit**

```bash
cd D:/workdir/leon/cocomhub/cocom
git add cmd/server/handler/init.go
git commit -m "feat: register admin comic API routes (compare/link/unlink/links)"
```

---

### 任务 13：搜索/首页/标签页过滤 redirect_to

**文件：** `cmd/server/view/index.go`、`cmd/server/view/search.go`、`cmd/server/view/tag_result.go`

- [ ] **步骤 1：在 index.go 的 initComicInfos 中添加过滤**

在 `cmd/server/view/index.go` 的 `initComicInfos()` 函数中，在现有 filter 中添加 `"redirect_to": nil`：

找到 filter 构建的位置（约第 146-169 行），在 filter map 中添加：
```go
filter["redirect_to"] = nil
```

或者如果 filter 是通过 builder 链式构建的，加：
```go
builder.FilterKV("redirect_to", nil)
```

- [ ] **步骤 2：在 search.go 的 SearchResultPage 中添加过滤**

在 `cmd/server/view/search.go` 中，在构建搜索查询时添加 `redirect_to = nil` 条件。

- [ ] **步骤 3：在 tag_result.go 的 TagResultPage 中添加过滤**

在 `cmd/server/view/tag_result.go` 中，在构建标签过滤时添加 `redirect_to = nil` 条件。

- [ ] **步骤 4：构建验证**

运行：`cd D:/workdir/leon/cocomhub/cocom && go build ./cmd/server/...`
预期：无编译错误

- [ ] **步骤 5：Commit**

```bash
cd D:/workdir/leon/cocomhub/cocom
git add cmd/server/view/index.go cmd/server/view/search.go cmd/server/view/tag_result.go
git commit -m "feat: filter out subordinate comics (redirect_to != nil) from search/index/tag pages"
```

---

### 任务 14：Admin 页面模板

**文件：** `cmd/server/view/static/tpl/admin.tpl`

- [ ] **步骤 1：在现有 admin.tpl 末尾、</body> 之前插入对比工具 UI 区域**

在现有 admin 功能区域之后（在 `<script>` 之前）插入完整的对比工具 UI。由于模板代码较长，关键区域结构：

```html
<!-- ===== 漫画对比工具 ===== -->
<div class="container index-container">
    <h2><i class="fa fa-images color-icon"></i> 漫画对比工具</h2>
    
    <!-- CID 输入区 -->
    <div style="display:flex;gap:12px;align-items:flex-end;flex-wrap:wrap;margin-bottom:16px;">
        <div>
            <label>主漫画 CID：</label>
            <input id="cid-main" type="text" value="" placeholder="输入 CID 或搜索标题..." style="width:220px;" />
        </div>
        <div>
            <label>对比漫画 CID：</label>
            <input id="cid-target" type="text" value="" placeholder="输入 CID 或搜索标题..." style="width:220px;" />
        </div>
        <button class="btn btn-primary" onclick="compareComics()"><i class="fa fa-search"></i> 对比</button>
        <button class="btn btn-secondary" onclick="swapCids()"><i class="fa fa-exchange-alt"></i> 交换</button>
    </div>

    <!-- 对比结果区域（初始隐藏） -->
    <div id="compare-result" style="display:none;">
        <!-- 漫画信息卡 -->
        <div id="comic-info-pair" style="display:flex;gap:12px;margin-bottom:12px;"></div>
        <!-- 统计栏 -->
        <div id="stats-bar" style="margin-bottom:12px;"></div>
        <!-- 对比表格 -->
        <div id="compare-table-container" style="overflow-x:auto;margin-bottom:12px;"></div>
        <!-- 并排预览 -->
        <div id="preview-panel" style="display:none;border:2px solid #ed2553;padding:12px;margin-bottom:12px;background:#1a1a1a;"></div>
        <!-- 建立链接区 -->
        <div id="link-action" style="margin-bottom:12px;"></div>
    </div>
</div>

<!-- 已链接漫画管理 -->
<div class="container index-container">
    <div style="display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:8px;">
        <h2><i class="fa fa-link color-icon"></i> 已链接的漫画</h2>
        <div>
            <button id="btn-show-current" class="btn btn-primary btn-sm" onclick="switchLinksView('current')">本次比较</button>
            <button id="btn-show-all" class="btn btn-secondary btn-sm" onclick="switchLinksView('all')">全部链接</button>
        </div>
    </div>
    <div id="linked-table-container" style="overflow-x:auto;margin-top:8px;"></div>
</div>
```

完整的交互逻辑放在 JS 模块中（任务 15）。

- [ ] **步骤 2：Commit**

```bash
cd D:/workdir/leon/cocomhub/cocom
git add cmd/server/view/static/tpl/admin.tpl
git commit -m "feat: add comic compare tool UI to admin page"
```

---

### 任务 15：前端 admin-compare.js 模块

**文件：** 新建 `cmd/server/view/static/custom/js/modules/admin-compare.js`

- [ ] **步骤 1：创建 admin-compare.js**

```javascript
// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

(function () {
  'use strict';

  var currentCID1 = 0;
  var currentCID2 = 0;
  var compareData = null;

  /* ===== CID 搜索补全 ===== */
  // 利用已有的 search-autocomplete 功能，嵌入到 CID 输入框

  /* ===== 对比操作 ===== */
  window.compareComics = function () {
    var cid1 = parseInt(document.getElementById('cid-main').value, 10);
    var cid2 = parseInt(document.getElementById('cid-target').value, 10);
    if (!cid1 || !cid2 || cid1 === cid2) {
      showAdminToast('请输入两个不同的有效 CID');
      return;
    }
    currentCID1 = cid1;
    currentCID2 = cid2;

    fetch('/api/admin/comic/compare', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ cid1: cid1, cid2: cid2 })
    })
      .then(function (resp) {
        if (!resp.ok) throw new Error('HTTP ' + resp.status);
        return resp.json();
      })
      .then(function (data) {
        compareData = data;
        renderCompareResult(data);
        loadLinks(currentCID1, currentCID2);
      })
      .catch(function (err) {
        showAdminToast('对比失败: ' + err.message);
      });
  };

  window.swapCids = function () {
    var a = document.getElementById('cid-main');
    var b = document.getElementById('cid-target');
    var t = a.value; a.value = b.value; b.value = t;
  };

  /* ===== 渲染对比结果 ===== */
  function renderCompareResult(data) {
    var container = document.getElementById('compare-result');
    container.style.display = 'block';

    var info1 = data.cid1.info;
    var info2 = data.cid2.info;

    // 漫画信息卡
    document.getElementById('comic-info-pair').innerHTML =
      '<div style="flex:1;background:#1a1a1a;border:1px solid #333;border-radius:4px;padding:12px;border-left:3px solid #ed2553;">' +
        '<strong style="color:#ed2553;">CID ' + info1.cid + '</strong>' +
        '<div style="color:#aaa;font-size:13px;margin-top:4px;">' + escapeHtml(info1.title && info1.title.english || '') + '</div>' +
        '<div style="font-size:11px;color:#666;margin-top:4px;">' + (data.cid1.pages.length || 0) + ' 页</div>' +
      '</div>' +
      '<div style="flex:1;background:#1a1a1a;border:1px solid #333;border-radius:4px;padding:12px;border-left:3px solid #f39c12;">' +
        '<strong style="color:#f39c12;">CID ' + info2.cid + '</strong>' +
        '<div style="color:#aaa;font-size:13px;margin-top:4px;">' + escapeHtml(info2.title && info2.title.english || '') + '</div>' +
        '<div style="font-size:11px;color:#666;margin-top:4px;">' + (data.cid2.pages.length || 0) + ' 页</div>' +
      '</div>';

    // 统计栏
    var stats = data.stats;
    document.getElementById('stats-bar').innerHTML =
      '<div style="display:flex;gap:16px;flex-wrap:wrap;font-size:13px;padding:8px 0;">' +
        '<span><strong>对齐页数：</strong>' + stats.total + '</span>' +
        '<span><strong>匹配度：</strong><span style="color:#4caf50;font-weight:bold;">' + (stats.match_ratio * 100).toFixed(1) + '%</span></span>' +
        '<span><strong style="color:#4caf50;">✅ ' + stats.matched + ' 匹配</strong></span>' +
        '<span><strong style="color:#f44336;">❌ ' + stats.mismatched + ' 不匹配</strong></span>' +
      '</div>';

    // 对比表格
    var html = '<table style="width:100%;border-collapse:collapse;font-size:13px;">' +
      '<thead><tr style="background:#2a2a2a;">' +
        '<th style="padding:6px 10px;text-align:left;">文件名</th>' +
        '<th style="padding:6px 10px;text-align:left;">CID1 MD5</th>' +
        '<th style="padding:6px 10px;text-align:left;">CID2 MD5</th>' +
        '<th style="padding:6px 10px;text-align:center;">状态</th>' +
        '<th style="padding:6px 10px;">操作</th>' +
      '</tr></thead><tbody>';
    (data.comparison || []).forEach(function (row) {
      var ok = row.md5_match;
      var cls = ok ? '' : ' style="background:rgba(244,67,54,0.08);"';
      html += '<tr' + cls + '>' +
        '<td style="padding:4px 10px;">' + escapeHtml(row.name) + '</td>' +
        '<td style="padding:4px 10px;font-family:monospace;font-size:11px;color:#666;">' + (row.cid1_md5 ? row.cid1_md5.substr(0, 12) + '...' : '<span style="color:#ffc107;">无</span>') + '</td>' +
        '<td style="padding:4px 10px;font-family:monospace;font-size:11px;' + (ok ? 'color:#666;' : 'color:#f44336;') + '">' + (row.cid2_md5 ? row.cid2_md5.substr(0, 12) + '...' : '<span style="color:#ffc107;">无</span>') + '</td>' +
        '<td style="padding:4px 10px;text-align:center;">' + (ok ? '<span style="color:#4caf50;">✅</span>' : '<span style="color:#f44336;">❌</span>') + '</td>' +
        '<td style="padding:4px 10px;text-align:center;">' +
          (ok ? '' : '<button class="btn btn-primary btn-sm" onclick="showPreview(\'' + escapeHtml(row.name) + '\')">并排预览</button>') +
        '</td>' +
      '</tr>';
    });
    html += '</tbody></table>';
    document.getElementById('compare-table-container').innerHTML = html;

    // 建立链接区
    renderLinkAction(data);
  }

  function renderLinkAction(data) {
    var cid1 = data.cid1.info.cid;
    var cid2 = data.cid2.info.cid;
    document.getElementById('link-action').innerHTML =
      '<div style="border-top:1px solid #333;padding-top:12px;">' +
        '<h3 style="font-size:14px;margin-bottom:8px;"><i class="fa fa-link color-icon"></i> 建立从属关系</h3>' +
        '<div style="display:flex;gap:12px;align-items:center;flex-wrap:wrap;">' +
          '<label>主：<input id="link-main" type="text" value="' + cid1 + '" style="width:80px;text-align:center;" /></label>' +
          '<span style="color:#555;">← 从属于 ←</span>' +
          '<label>从：<input id="link-sub" type="text" value="' + cid2 + '" style="width:80px;text-align:center;" /></label>' +
          '<button class="btn btn-primary" onclick="confirmLink()"><i class="fa fa-link"></i> 建立链接</button>' +
        '</div>' +
        '<div style="font-size:12px;color:#888;margin-top:6px;">链接后从属 comic 重定向到主 comic，tags 自动合并，不在搜索结果展示。</div>' +
      '</div>';
  }

  /* ===== 并排预览 ===== */
  window.showPreview = function (fileName) {
    var panel = document.getElementById('preview-panel');
    panel.style.display = 'block';
    panel.innerHTML =
      '<div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:8px;">' +
        '<h3 style="margin:0;font-size:14px;">' + escapeHtml(fileName) + '</h3>' +
        '<button class="btn btn-secondary btn-sm" onclick="document.getElementById(\'preview-panel\').style.display=\'none\'">关闭</button>' +
      '</div>' +
      '<div style="display:flex;gap:12px;">' +
        '<div style="flex:1;text-align:center;">' +
          '<div style="color:#ed2553;font-weight:bold;font-size:12px;">CID ' + currentCID1 + '</div>' +
          '<img src="/galleries/' + currentCID1 + '/' + encodeURIComponent(fileName) + '" style="max-width:100%;max-height:300px;border-radius:4px;" />' +
        '</div>' +
        '<div style="flex:1;text-align:center;">' +
          '<div style="color:#f39c12;font-weight:bold;font-size:12px;">CID ' + currentCID2 + '</div>' +
          '<img src="/galleries/' + currentCID2 + '/' + encodeURIComponent(fileName) + '" style="max-width:100%;max-height:300px;border-radius:4px;" />' +
        '</div>' +
      '</div>';
  };

  /* ===== 建立链接 ===== */
  window.confirmLink = function () {
    var mainCID = parseInt(document.getElementById('link-main').value, 10);
    var subCID = parseInt(document.getElementById('link-sub').value, 10);
    if (!mainCID || !subCID || mainCID === subCID) {
      showAdminToast('请输入有效的主/从 CID');
      return;
    }
    if (!confirm('确认将从属 CID ' + subCID + ' 链接到主 CID ' + mainCID + '？\n操作可撤销。')) return;

    fetch('/api/admin/comic/link', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ main_cid: mainCID, sub_cid: subCID })
    })
      .then(function (resp) {
        if (!resp.ok) throw new Error('HTTP ' + resp.status);
        return resp.json();
      })
      .then(function () {
        showAdminToast('链接建立成功！CID ' + subCID + ' → ' + mainCID);
        loadLinks(mainCID, subCID);
      })
      .catch(function (err) {
        showAdminToast('建立链接失败: ' + err.message);
      });
  };

  /* ===== 取消链接 ===== */
  window.unlinkComic = function (subCID) {
    if (!confirm('确认取消 CID ' + subCID + ' 的从属关系？已合并的 tags 将保留。')) return;
    fetch('/api/admin/comic/unlink', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ main_cid: 0, sub_cid: subCID })
    })
      .then(function (resp) {
        if (!resp.ok) throw new Error('HTTP ' + resp.status);
        return resp.json();
      })
      .then(function () {
        showAdminToast('取消合并成功！CID ' + subCID + ' 已恢复独立。');
        loadLinks(currentCID1, currentCID2);
      })
      .catch(function (err) {
        showAdminToast('取消合并失败: ' + err.message);
      });
  };

  /* ===== 加载链接列表 ===== */
  function loadLinks(mainCID, subCID) {
    // 取全部链接
    fetch('/api/admin/comic/links?all=true')
      .then(function (resp) { return resp.json(); })
      .then(function (data) {
        renderLinksTable(data.links || [], mainCID, subCID);
      })
      .catch(function () {});
  }

  function renderLinksTable(links, mainCID, subCID) {
    var allHtml = '';
    var currentHtml = '';
    links.forEach(function (link) {
      var isCurrent = (link.main_cid === mainCID && link.sub_cid === subCID);
      var row =
        '<tr>' +
          '<td style="padding:4px 10px;"><strong style="color:#ed2553;">' + link.main_cid + '</strong></td>' +
          '<td style="padding:4px 10px;">' + escapeHtml(link.sub_title || '') + '</td>' +
          '<td style="padding:4px 10px;"><span style="color:#f39c12;">' + link.sub_cid + '</span></td>' +
          '<td style="padding:4px 10px;"><button class="btn btn-warning btn-sm" onclick="unlinkComic(' + link.sub_cid + ')"><i class="fa fa-unlink"></i> 取消合并</button></td>' +
        '</tr>';
      allHtml += row;
      if (isCurrent) currentHtml += row;
    });

    if (!currentHtml) currentHtml = '<tr><td colspan="4" style="padding:10px;text-align:center;color:#888;">无当前比较相关的链接</td></tr>';
    if (!allHtml) allHtml = '<tr><td colspan="4" style="padding:10px;text-align:center;color:#888;">暂无链接</td></tr>';

    document.getElementById('linked-table-container').innerHTML =
      '<table id="linked-table-current" class="link-table" style="width:100%;border-collapse:collapse;font-size:13px;">' +
        '<thead><tr style="background:#2a2a2a;"><th style="padding:6px 10px;">主 CID</th><th style="padding:6px 10px;">从属标题</th><th style="padding:6px 10px;">从属 CID</th><th style="padding:6px 10px;">操作</th></tr></thead>' +
        '<tbody>' + currentHtml + '</tbody>' +
      '</table>' +
      '<table id="linked-table-all" class="link-table" style="width:100%;border-collapse:collapse;font-size:13px;display:none;">' +
        '<thead><tr style="background:#2a2a2a;"><th style="padding:6px 10px;">主 CID</th><th style="padding:6px 10px;">从属标题</th><th style="padding:6px 10px;">从属 CID</th><th style="padding:6px 10px;">操作</th></tr></thead>' +
        '<tbody>' + allHtml + '</tbody>' +
      '</table>';
  }

  /* ===== 切换链接视图 ===== */
  window.switchLinksView = function (mode) {
    var cTable = document.getElementById('linked-table-current');
    var aTable = document.getElementById('linked-table-all');
    var cBtn = document.getElementById('btn-show-current');
    var aBtn = document.getElementById('btn-show-all');
    if (mode === 'current') {
      cTable.style.display = ''; aTable.style.display = 'none';
      cBtn.className = 'btn btn-primary btn-sm'; aBtn.className = 'btn btn-secondary btn-sm';
    } else {
      cTable.style.display = 'none'; aTable.style.display = '';
      cBtn.className = 'btn btn-secondary btn-sm'; aBtn.className = 'btn btn-primary btn-sm';
    }
  };

  /* ===== 页面加载时自动加载链接 ===== */
  document.addEventListener('DOMContentLoaded', function () {
    loadLinks(0, 0);
  });

  /* ===== 工具函数 ===== */
  function showAdminToast(msg) {
    var el = document.createElement('div');
    el.className = 'alert';
    el.textContent = msg;
    var msgContainer = document.getElementById('messages');
    if (msgContainer) {
      msgContainer.appendChild(el);
      setTimeout(function () { el.remove(); }, 3000);
    } else {
      alert(msg);
    }
  }

  function escapeHtml(str) {
    if (!str) return '';
    var div = document.createElement('div');
    div.appendChild(document.createTextNode(str));
    return div.innerHTML;
  }
})();
```

- [ ] **步骤 2：在 head.common.tpl 中引用 admin-compare.js**

在 `scripts.js` 之后添加：
```html
<script src="/static/custom/js/modules/admin-compare.js"></script>
```

- [ ] **步骤 3：构建验证**

运行：`cd D:/workdir/leon/cocomhub/cocom && go build ./...`
预期：无编译错误

- [ ] **步骤 4：Commit**

```bash
cd D:/workdir/leon/cocomhub/cocom
git add cmd/server/view/static/custom/js/modules/admin-compare.js cmd/server/view/static/tpl/head.common.tpl
git commit -m "feat: add admin-compare.js for comic compare tool frontend logic"
```

---

## 任务 16：添加 CSS 样式

**文件：** `cmd/server/view/static/custom/css/styles.css`

- [ ] **步骤 1：追加对比工具专用样式**

在 `styles.css` 末尾添加：

```css
/* ===== Admin 漫画对比工具 ===== */

.compare-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 13px;
}

.compare-table th {
    text-align: left;
    padding: 6px 10px;
    background: #2a2a2a;
    border-bottom: 2px solid #444;
    color: #aaa;
    font-weight: 400;
}

.compare-table td {
    padding: 4px 10px;
    border-bottom: 1px solid #2a2a2a;
}

.compare-table tr:hover {
    background: rgba(255, 255, 255, 0.03);
}

.compare-table .md5-cell {
    font-family: monospace;
    font-size: 11px;
    color: #666;
}

.compare-table .md5-mismatch {
    color: #f44336;
}

.compare-table .status-ok {
    color: #4caf50;
}

.compare-table .status-err {
    color: #f44336;
}

.compare-table .status-missing {
    color: #ffc107;
}

/* 推荐相关 */
.recommend-section {
    margin-bottom: 20px;
}

.recommend-header {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 10px;
}

.recommend-header h2 {
    margin: 0;
    font-size: 16px;
    font-weight: 400;
}

.recommend-refresh {
    cursor: pointer;
    background: transparent;
    border: 1px solid #555;
    color: #888;
    padding: 3px 8px;
    border-radius: 3px;
    font-size: 12px;
    transition: 0.15s;
}

.recommend-refresh:hover {
    color: #fff;
    border-color: #888;
}

.recommend-grid {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
}

/* 链接表格 */
.link-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 13px;
}

.link-table th {
    text-align: left;
    padding: 6px 10px;
    background: #2a2a2a;
    border-bottom: 2px solid #444;
    color: #aaa;
    font-weight: 400;
}

.link-table td {
    padding: 4px 10px;
    border-bottom: 1px solid #2a2a2a;
}

.link-table tr:hover {
    background: rgba(255, 255, 255, 0.03);
}
```

- [ ] **步骤 2：Commit**

```bash
cd D:/workdir/leon/cocomhub/cocom
git add cmd/server/view/static/custom/css/styles.css
git commit -m "feat: add admin compare tool and recommendation CSS styles"
```

---

## 验证方案

1. **多维度推荐**：
   - 启动 server，打开任意漫画 detail 页
   - 确认 5 个推荐栏在页面加载后出现
   - 检查推荐内容不重复、不含当前漫画
   - 点击刷新图标确认重新加载

2. **自动标签 ID**：
   - 通过 API 创建一个新 tag（ID=0），检查返回的 tag ID ≥ 1000000000
   - 连续创建确认 ID 递增

3. **漫画对比工具**：
   - 访问 `/admin` 页面
   - 输入两个 CID，点击对比
   - 确认文件列表、MD5 对比、匹配度统计正确
   - 点击不匹配行的「并排预览」确认图片显示

4. **建立/取消链接**：
   - 对比后点击「建立链接」
   - 访问从属 comic 确认 302 重定向到主 comic
   - 在 admin 页面点击「取消合并」，确认从属恢复独立

5. **搜索过滤**：
   - 建立链接后，搜索从属漫画标题确认不在结果中
   - 首页确认不显示从属漫画

6. **构建和测试**：
   - `go build ./cmd/server/...` 通过
   - `go test -tags=memory_storage_integration ./...` 现有测试不受影响