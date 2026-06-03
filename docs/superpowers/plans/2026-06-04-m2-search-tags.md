# M2：搜索与标签体验升级 — 实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 实现搜索自动补全、搜索结果高亮、标签批量操作增量聚合、标签 API 分页增强

**架构：** 新增一个轻量级自动补全 API + 前端搜索框绑定的混合下拉组件；搜索结果高亮为服务端渲染（SSR）模板函数；标签批量操作从全量聚合改为增量更新

**技术栈：** Go 1.26 + Gin + MongoDB + Vanilla JS

---

## 涉及文件

| 文件 | 操作 | 职责 |
|------|------|------|
| `cmd/server/api/comic.go` | 修改 | 添加 `AutocompleteResponse` 等新类型 |
| `cmd/server/handler/search.go` | 创建 | 处理 `GET /api/search/autocomplete` 请求 |
| `cmd/server/handler/init.go` | 修改 | 注册新路由 |
| `cmd/server/view/static/custom/js/modules/search-autocomplete.js` | 创建 | 搜索框绑定、下拉渲染、键盘导航 |
| `cmd/server/view/static/tpl/head.tpl` | 修改 | 引入新 script |
| `cmd/server/view/static/custom/js/scripts.js` | 修改 | `initGalleryPage` 中调用自动补全 |
| `cmd/server/view/static/custom/css/styles.css` | 修改 | 自动补全下拉浮层样式 + `<mark>` 高亮样式 |
| `cmd/server/view/index.go` | 修改 | 添加 `HighlightKeyword` 模板函数 |
| `cmd/server/view/static/tpl/index.tpl` | 修改 | 标题区域调用高亮函数 |
| `cmd/server/view/static/tpl/tag_list_result.tpl` | 修改 | 标签名高亮 |
| `cmd/server/handler/tag_edit.go` | 修改 | 批量添加后增量更新而非全量聚合 |
| `cmd/server/handler/tags_agg.go` | 修改 | 增加 `?page=`/`?page_size=` 分页参数 |
| `cmd/server/handler/search_test.go` | 创建 | 自动补全端点测试 |
| `cmd/server/handler/tags_search_test.go` | 创建 | 标签搜索端点测试 |

---

### 任务 1：自动补全 API 端点

**文件：**
- 创建：`cmd/server/handler/search.go`
- 修改：`cmd/server/api/comic.go`
- 修改：`cmd/server/handler/init.go`

- [ ] **步骤 1：在 `api/comic.go` 中添加自动补全相关类型**

在 `api/comic.go` 末尾追加：

```go
// AutocompleteComic 自动补全中漫画的轻量信息
type AutocompleteComic struct {
	CID   int    `json:"cid"`
	Title string `json:"title"`
}

// AutocompleteResponse 自动补全响应
type AutocompleteResponse struct {
	Comics []*AutocompleteComic `json:"comics"`
	Tags   []*TagInfo           `json:"tags"`
}
```

- [ ] **步骤 2：创建 `cmd/server/handler/search.go`**

```go
// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"log/slog"
	"net/http"
	"regexp"
	"strconv"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/cmd/server/internal/tag"
	"github.com/cocomhub/cocom/pkg/httpwrap"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SearchAutocomplete GET /api/search/autocomplete?q=xxx&limit=5
// 返回匹配的漫画标题和标签（混合下拉）
func SearchAutocomplete(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	query := req.URL.Query().Get("q")
	if query == "" {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "query q is required")
		return
	}

	limit := int64(5)
	if l := req.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.ParseInt(l, 10, 64); err == nil && v > 0 && v <= 20 {
			limit = v
		}
	}

	// 搜索漫画标题
	escapedQuery := primitive.Regex{Pattern: regexp.QuoteMeta(query), Options: "i"}
	infos, err := comic.GetRangeComicInfos(ctx, limit, 0,
		"$or", []bson.M{
			{"title.english": bson.M{"$regex": escapedQuery}},
			{"title.japanese": bson.M{"$regex": escapedQuery}},
			{"title.pretty": bson.M{"$regex": escapedQuery}},
		})
	if err != nil {
		slog.ErrorContext(ctx, "search autocomplete comics failed", slog.String("errmsg", err.Error()))
		// 不中断，只返回空漫画列表
		infos = nil
	}

	comics := make([]*api.AutocompleteComic, 0, len(infos))
	for _, info := range infos {
		title := info.Title.English
		if title == "" {
			title = info.Title.Pretty
		}
		if title == "" {
			title = info.Title.Japanese
		}
		comics = append(comics, &api.AutocompleteComic{
			CID:   info.CID,
			Title: title,
		})
	}

	// 搜索标签名
	tags, err := tag.SearchTags(ctx, "", query, limit)
	if err != nil {
		slog.ErrorContext(ctx, "search autocomplete tags failed", slog.String("errmsg", err.Error()))
		tags = nil
	}

	httpwrap.ResponseSucc(ctx, w, api.AutocompleteResponse{
		Comics: comics,
		Tags:   tags,
	})
}
```

- [ ] **步骤 3：在 `handler/init.go` 中注册新路由**

在 `r.POST("/api/settings", ...)` 之前添加：

```go
r.GET("/api/search/autocomplete", gin.WrapF(SearchAutocomplete))
```

- [ ] **步骤 4：编译验证**

运行：
```bash
cd D:\workdir\leon\cocomhub\cocom
make build
```
预期：编译成功，无错误

- [ ] **步骤 5：Commit**

```bash
git add cmd/server/api/comic.go cmd/server/handler/search.go cmd/server/handler/init.go
git commit -m "feat(search): 新增自动补全 API 端点 /api/search/autocomplete"
```

---

### 任务 2：前端搜索自动补全下拉

**文件：**
- 创建：`cmd/server/view/static/custom/js/modules/search-autocomplete.js`
- 修改：`cmd/server/view/static/tpl/head.tpl`
- 修改：`cmd/server/view/static/custom/js/scripts.js`
- 修改：`cmd/server/view/static/custom/css/styles.css`

- [ ] **步骤 1：创建 `search-autocomplete.js`**

```js
/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Search autocomplete dropdown — mixed comic titles + tags.
 */
(function() {
  'use strict';

  function esc(s) {
    return String(s).replace(/[&<>"']/g, function(c) {
      return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c];
    });
  }

  window.initSearchAutocomplete = function initSearchAutocomplete() {
    var input = document.querySelector('input[type="search"]');
    if (!input) return;

    var dropdown = document.createElement('div');
    dropdown.className = 'search-autocomplete-dropdown';
    dropdown.style.display = 'none';
    input.parentNode.appendChild(dropdown);

    var timeout = null;

    function hideDropdown() {
      dropdown.style.display = 'none';
      dropdown.innerHTML = '';
    }

    function handleResponse(comics, tags) {
      dropdown.innerHTML = '';

      var html = '';

      // Comics section
      if (comics && comics.length > 0) {
        html += '<div class="autocomplete-section"><div class="autocomplete-section-title">漫画</div>';
        comics.forEach(function(c) {
          html += '<div class="autocomplete-item autocomplete-comic" data-cid="' + c.cid + '">' +
            '<i class="fa fa-book" style="margin-right:6px;color:#888;"></i>' +
            '<span class="autocomplete-title">' + esc(c.title) + '</span>' +
            '<span class="autocomplete-cid">#' + c.cid + '</span></div>';
        });
        html += '</div>';
      }

      // Tags section
      if (tags && tags.length > 0) {
        html += '<div class="autocomplete-section"><div class="autocomplete-section-title">标签</div>';
        tags.forEach(function(t) {
          html += '<div class="autocomplete-item autocomplete-tag" data-type="' + esc(t.type) + '" data-name="' + esc(t.name) + '" data-url="' + esc(t.url) + '">' +
            '<i class="fa fa-tag" style="margin-right:6px;color:#888;"></i>' +
            '<span class="autocomplete-tag-type">[' + esc(t.type) + ']</span> ' +
            '<span class="autocomplete-tag-name">' + esc(t.name) + '</span>' +
            '<span class="autocomplete-tag-count">' + t.count + '</span></div>';
        });
        html += '</div>';
      }

      if (!html) {
        hideDropdown();
        return;
      }

      dropdown.innerHTML = html;
      dropdown.style.display = 'block';

      // Bind item click
      dropdown.querySelectorAll('.autocomplete-item').forEach(function(item) {
        item.addEventListener('click', function() {
          if (item.classList.contains('autocomplete-comic')) {
            var cid = item.getAttribute('data-cid');
            window.location.href = '/g/' + cid + '/';
          } else if (item.classList.contains('autocomplete-tag')) {
            var url = item.getAttribute('data-url');
            window.location.href = '/tag' + url;
          }
        });
      });
    }

    input.addEventListener('input', function() {
      var q = this.value.trim();
      if (timeout) clearTimeout(timeout);
      if (q.length < 2) {
        hideDropdown();
        return;
      }
      timeout = setTimeout(function() {
        var xhr = new XMLHttpRequest();
        xhr.open('GET', '/api/search/autocomplete?q=' + encodeURIComponent(q) + '&limit=5');
        xhr.onload = function() {
          if (xhr.status !== 200) return;
          try {
            var resp = JSON.parse(xhr.responseText);
            handleResponse(resp.body.comics, resp.body.tags);
          } catch (e) { /* ignore parse errors */ }
        };
        xhr.send();
      }, 200);
    });

    // Keyboard navigation (reuse bindAutocompleteKeys)
    if (typeof window.bindAutocompleteKeys === 'function') {
      window.bindAutocompleteKeys(input, dropdown, function() {
        // Default Enter: submit form if no selection
        input.closest('form').submit();
      });
    }

    // Close on blur (with delay for click)
    input.addEventListener('blur', function() {
      setTimeout(hideDropdown, 200);
    });

    // Close on Escape
    input.addEventListener('keydown', function(e) {
      if (e.key === 'Escape') {
        hideDropdown();
      }
    });
  };
})();
```

- [ ] **步骤 2：在 `head.tpl` 中引入 `search-autocomplete.js`**

在 `navigation.js` 行后追加：

```html
<script src="/static/custom/js/modules/search-autocomplete.js"></script>
```

- [ ] **步骤 3：在 `scripts.js` 的 `initGalleryPage` 中添加 `initSearchAutocomplete` 调用**

```js
function initGalleryPage() {
    if (typeof initThumbnailZoom === 'function') initThumbnailZoom();
    if (typeof initLargeModeToggle === 'function') initLargeModeToggle();
    if (typeof initKeyboardShortcuts === 'function') initKeyboardShortcuts();
    if (typeof initSearchAutocomplete === 'function') initSearchAutocomplete();
}
```

- [ ] **步骤 4：在 `styles.css` 中添加自动补全下拉样式 + 搜索高亮样式**

在末尾添加：

```css
/* ====== Search Autocomplete Dropdown ====== */
.search-autocomplete-dropdown {
    position: absolute;
    top: 100%;
    left: 0;
    right: 0;
    background: #2a2a2a;
    border: 1px solid #555;
    border-radius: var(--radius-md, 4px);
    max-height: 400px;
    overflow-y: auto;
    z-index: 9999;
    box-shadow: 0 4px 12px rgba(0,0,0,0.4);
}

.search-autocomplete-dropdown .autocomplete-section-title {
    padding: 4px 10px;
    font-size: 11px;
    color: #888;
    text-transform: uppercase;
    letter-spacing: 1px;
    background: #222;
    border-bottom: 1px solid #444;
}

.search-autocomplete-dropdown .autocomplete-item {
    padding: 8px 10px;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 4px;
    border-bottom: 1px solid #333;
    transition: background 0.15s;
}

.search-autocomplete-dropdown .autocomplete-item:hover,
.search-autocomplete-dropdown .autocomplete-item.keyboard-selected {
    background: #444 !important;
}

.search-autocomplete-dropdown .autocomplete-item:last-child {
    border-bottom: none;
}

.search-autocomplete-dropdown .autocomplete-cid {
    margin-left: auto;
    color: #666;
    font-size: 12px;
}

.search-autocomplete-dropdown .autocomplete-tag-type {
    color: #888;
    font-size: 12px;
}

.search-autocomplete-dropdown .autocomplete-tag-count {
    margin-left: auto;
    color: #666;
    font-size: 12px;
}

/* ====== Search Highlight ====== */
mark.search-highlight {
    background: #ffc107;
    color: #000;
    padding: 0 2px;
    border-radius: 2px;
}
```

- [ ] **步骤 5：Commit**

```bash
git add cmd/server/view/static/custom/js/modules/search-autocomplete.js cmd/server/view/static/tpl/head.tpl cmd/server/view/static/custom/js/scripts.js cmd/server/view/static/custom/css/styles.css
git commit -m "feat(ui): 搜索框混合自动补全（标题+标签）"
```

---

### 任务 3：搜索结果高亮（SSR）

**文件：**
- 修改：`cmd/server/view/index.go`
- 修改：`cmd/server/view/static/tpl/index.tpl`
- 修改：`cmd/server/view/static/tpl/tag_list_result.tpl`

- [ ] **步骤 1：在 `view/index.go` 中添加 `HighlightKeyword` 模板函数**

在 `GalleryIndexPage` 结构体上添加方法：

```go
import "html/template"

// HighlightKeyword 将 text 中的 keyword 子串替换为 <mark> 标签包裹
func (p *GalleryIndexPage) HighlightKeyword(text, keyword string) template.HTML {
	if keyword == "" || text == "" {
		return template.HTML(template.HTMLEscapeString(text))
	}
	escaped := template.HTMLEscapeString(text)
	kwEscaped := template.HTMLEscapeString(keyword)
	// 不区分大小写，用正则替换
	re := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(kwEscaped))
	result := re.ReplaceAllString(escaped, `<mark class="search-highlight">$&</mark>`)
	return template.HTML(result)
}
```

需要在 `view/index.go` 头部添加 import：
```go
import (
    "html/template"
    "regexp"
)
```

- [ ] **步骤 2：修改 `index.tpl` 中标题区域使用高亮函数**

找到第 74 行的 `.Caption` 和 89 行的 `.Caption` 修改：

```html
<div class="caption">{{$.HighlightKeyword $detail.Title.English $.SearchQuery}}</div>
```

将搜索标题两处 `{{$detail.Title.English}}` 改为 `{{$.HighlightKeyword $detail.Title.English $.SearchQuery}}`。

仅在 `.SearchQuery` 非空时高亮。还需要在标题区域添加条件判断。

修改第 74 行（Popular Now 区域）：
```html
<div class="caption">{{if $.SearchQuery}}{{$.HighlightKeyword $detail.Title.English $.SearchQuery}}{{else}}{{$detail.Title.English}}{{end}}</div>
```

修改第 89 行（New Uploads 区域）：
```html
<div class="caption">{{if $.SearchQuery}}{{$.HighlightKeyword $detail.Title.English $.SearchQuery}}{{else}}{{$detail.Title.English}}{{end}}</div>
```

- [ ] **步骤 3：修改 `tag_list_result.tpl` 中标签名高亮**

暂无高亮需求（标签列表页按 A-Z 字母索引或按热门排序，展示的是标签名本身，没有搜索词上下文），跳过。

- [ ] **步骤 4：编译验证**

运行：
```bash
cd D:\workdir\leon\cocomhub\cocom
make build
```
预期：编译成功

- [ ] **步骤 5：Commit**

```bash
git add cmd/server/view/index.go cmd/server/view/static/tpl/index.tpl
git commit -m "feat(ui): 搜索列表页标题关键词高亮（SSR <mark>）"
```

---

### 任务 4：标签批量操作增量聚合

**文件：**
- 修改：`cmd/server/handler/tag_edit.go`（`BatchAddTagToComics` 函数）

- [ ] **步骤 1：将全量聚合改为增量更新**

将 `BatchAddTagToComics` 末尾的以下代码：

```go
// 批量操作结束后，全量重聚合以修正 count 并清空缓存
if err := tag.AggregateTags(ctx); err != nil {
    slog.ErrorContext(ctx, "re-aggregate tags after batch add failed", slog.String("errmsg", err.Error()))
}
if err := cache.Reset(); err != nil {
    slog.WarnContext(ctx, "cache reset after batch add failed", slog.String("errmsg", err.Error()))
}
```

替换为：

```go
// 批量操作结束后，增量更新 comicTag count
t := batchReq.Tag
if err := tag.UpdateComicTagIncremental(ctx, t.Type, t.ID, t.Name, t.URL, resp.Updated); err != nil {
    slog.WarnContext(ctx, "incremental tag update failed, falling back to full aggregate",
        slog.String("errmsg", err.Error()))
    if err := tag.AggregateTags(ctx); err != nil {
        slog.ErrorContext(ctx, "re-aggregate tags after batch add failed", slog.String("errmsg", err.Error()))
    }
}
if err := cache.Reset(); err != nil {
    slog.WarnContext(ctx, "cache reset after batch add failed", slog.String("errmsg", err.Error()))
}
```

- [ ] **步骤 2：编译验证**

运行：
```bash
cd D:\workdir\leon\cocomhub\cocom
make build
```
预期：编译成功

- [ ] **步骤 3：Commit**

```bash
git add cmd/server/handler/tag_edit.go
git commit -m "perf(api): 批量添加 tag 改用增量聚合，避免全量重聚合"
```

---

### 任务 5：标签 API 分页增强

**文件：**
- 修改：`cmd/server/handler/tags_agg.go`（`GetTags` 函数）

- [ ] **步骤 1：增加 `?page=`/`?page_size=` 分页参数支持**

在 `GetTags` 函数中 `limit` 解析之后添加 `page`/`page_size` 参数解析逻辑：

```go
// page/page_size 参数支持（与 skip/limit 二选一，优先使用 page/page_size）
pageSize := int64(0)
if ps := req.URL.Query().Get("page_size"); ps != "" {
    if v, err := strconv.ParseInt(ps, 10, 64); err == nil && v > 0 {
        pageSize = v
        if pageSize > 100 {
            pageSize = 100
        }
    }
}
if p := req.URL.Query().Get("page"); p != "" {
    if v, err := strconv.ParseInt(p, 10, 64); err == nil && v >= 1 {
        if pageSize > 0 {
            // page + page_size 同时存在，覆盖 skip/limit
            skip = (v - 1) * pageSize
            limit = pageSize
        }
    }
}
```

这段代码插在 `limit` 解析（第 45-50 行）之后、`skip` 解析（第 51-56 行）之前：

```go
limit := int64(20)
if l := req.URL.Query().Get("limit"); l != "" {
    if v, err := strconv.ParseInt(l, 10, 64); err == nil && v > 0 {
        limit = v
    }
}
// [插入 page/page_size 解析]
skip := int64(0)
```

- [ ] **步骤 2：编译验证**

运行：
```bash
cd D:\workdir\leon\cocomhub\cocom
make build
```
预期：编译成功

- [ ] **步骤 3：Commit**

```bash
git add cmd/server/handler/tags_agg.go
git commit -m "feat(api): 标签列表 API 增加 page/page_size 分页参数"
```

---

### 任务 6：搜索 API 测试

**文件：**
- 创建：`cmd/server/handler/search_test.go`
- 创建：`cmd/server/handler/tags_search_test.go`

- [ ] **步骤 1：创建 `search_test.go`**

```go
// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/pkg/httpwrap"
)

func TestSearchAutocomplete_EmptyQuery(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/search/autocomplete?q=", nil)
	w := httptest.NewRecorder()
	SearchAutocomplete(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %s", err)
	}
	if resp.Head.Code != 0 {
		t.Errorf("expected code 0 (error), got %d", resp.Head.Code)
	}
}

func TestSearchAutocomplete_ShortQuery(t *testing.T) {
	// 单字符查询应返回空结果（实际是 API 仍会处理，但前端限制 2 字符）
	req := httptest.NewRequest(http.MethodGet, "/api/search/autocomplete?q=a", nil)
	w := httptest.NewRecorder()
	SearchAutocomplete(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp httpwrap.ResponseInfo[api.AutocompleteResponse]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %s", err)
	}
	// API 不限制短查询，返回的结果可能是空的
	if resp.Body.Comics == nil {
		t.Error("expected comics field (may be empty), got nil")
	}
	if resp.Body.Tags == nil {
		t.Error("expected tags field (may be empty), got nil")
	}
}

func TestSearchAutocomplete_ResponseStructure(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/search/autocomplete?q=test&limit=3", nil)
	w := httptest.NewRecorder()
	SearchAutocomplete(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp httpwrap.ResponseInfo[api.AutocompleteResponse]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %s", err)
	}

	// 验证类型
	if resp.Body.Comics == nil {
		t.Error("comics should be non-nil array")
	}
	if resp.Body.Tags == nil {
		t.Error("tags should be non-nil array")
	}

	// 验证 Total <= limit (3)
	if len(resp.Body.Tags) > 3 {
		t.Errorf("expected at most 3 tags, got %d", len(resp.Body.Tags))
	}
	if len(resp.Body.Comics) > 3 {
		t.Errorf("expected at most 3 comics, got %d", len(resp.Body.Comics))
	}
}
```

- [ ] **步骤 2：创建 `tags_search_test.go`**

```go
// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/pkg/httpwrap"
)

func TestSearchTags_EmptyQuery(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/comic/tags/search?type=tag&q=", nil)
	w := httptest.NewRecorder()
	SearchTags(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 (empty query returns top tags), got %d", w.Code)
	}

	var resp httpwrap.ResponseInfo[api.TagSearchResponse]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %s", err)
	}
	if resp.Body.Tags == nil {
		t.Error("expected tags field, got nil")
	}
}

func TestSearchTags_DefaultTypeAndLimit(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/comic/tags/search?q=love", nil)
	w := httptest.NewRecorder()
	SearchTags(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp httpwrap.ResponseInfo[api.TagSearchResponse]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %s", err)
	}
	// 默认 limit=20
	if len(resp.Body.Tags) > 20 {
		t.Errorf("expected at most 20 tags (default limit), got %d", len(resp.Body.Tags))
	}
	if resp.Body.Total != len(resp.Body.Tags) {
		t.Errorf("expected total=%d == len(tags)=%d", resp.Body.Total, len(resp.Body.Tags))
	}
}

func TestSearchTags_WithLimit(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/comic/tags/search?type=artist&q=a&limit=5", nil)
	w := httptest.NewRecorder()
	SearchTags(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp httpwrap.ResponseInfo[api.TagSearchResponse]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %s", err)
	}
	if len(resp.Body.Tags) > 5 {
		t.Errorf("expected at most 5 tags, got %d", len(resp.Body.Tags))
	}
}

func TestSearchTags_ExceedsMaxLimit(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/comic/tags/search?q=test&limit=500", nil)
	w := httptest.NewRecorder()
	SearchTags(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp httpwrap.ResponseInfo[api.TagSearchResponse]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %s", err)
	}
	// 上限 100
	if len(resp.Body.Tags) > 100 {
		t.Errorf("expected at most 100 tags (max limit), got %d", len(resp.Body.Tags))
	}
}

func TestSearchTags_TagInfoStructure(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/comic/tags/search?type=tag&q=love&limit=3", nil)
	w := httptest.NewRecorder()
	SearchTags(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp httpwrap.ResponseInfo[api.TagSearchResponse]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %s", err)
	}

	for _, tag := range resp.Body.Tags {
		if tag.Name == "" {
			t.Error("tag name should not be empty")
		}
		if tag.Type == "" {
			t.Error("tag type should not be empty")
		}
	}
}
```

- [ ] **步骤 3：验证测试编译**

运行：
```bash
cd D:\workdir\leon\cocomhub\cocom
go test -run TestSearch -tags=memory_storage_integration ./cmd/server/handler/ -v -count=1 -timeout 30s
```
预期：测试编译成功，运行结果可能因缺少 MongoDB 连接而失败（部分测试需要真实的存储后端），但所有测试框架验证通过

- [ ] **步骤 4：Commit**

```bash
git add cmd/server/handler/search_test.go cmd/server/handler/tags_search_test.go
git commit -m "test(api): 搜索与标签 API 端点单元测试"
```

---

## 验收标准

- [ ] 搜索框输入 2 个字符后 300ms 内展示混合下拉建议
- [ ] 搜索结果页标题中匹配文字有 `<mark class="search-highlight">` 高亮
- [ ] `GET /api/comic/tags` 支持 `?page=2&page_size=50` 参数
- [ ] 批量添加标签后不再触发全量 `AggregateTags`，改用增量更新
- [ ] `make build` 编译通过
- [ ] 所有测试文件至少可编译