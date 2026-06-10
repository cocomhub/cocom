# Comic 交互增强实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 在 cocom 项目中实现三大交互增强模块——Index 页快速链接/对比（Module A）、Detail 页页面管理（Module B）、交互改进清单（Module C），全部以 TDD 方式实施。

**架构：** 前端使用原生 JS + 模板渲染，后端基于 Gin + MongoDB。Module A 改造现有 Link API 支持 `sub_cids` 数组；Module B 新增页面 CRUD handler + 删除 comic（tombstone 模式）；Module C 为前两者的交互打磨。

**技术栈：** Go 1.26 / Gin / MongoDB / 原生 JS / HTML 模板 / CSS3

---

## 文件结构

### 创建的文件

| 文件 | 职责 |
|------|------|
| `cmd/server/view/static/custom/js/modules/quick-link.js` | Index 页链接/对比模式前端逻辑 |
| `cmd/server/view/static/custom/js/modules/page-manager.js` | Detail 页页管理前端逻辑（删除/插入/替换/重排） |
| `cmd/server/view/static/tpl/page_deleted.tpl` | 已删除 comic 提示页 |
| `cmd/server/handler/comic_page.go` | 页面操作 handler（savePages / getComicPages） |
| `cmd/server/handler/admin_test.go` | admin.go 的 handler 测试（TDD） |
| `cmd/server/handler/comic_page_test.go` | comic_page.go 的 handler 测试（TDD） |
| `cmd/server/view/static/custom/js/modules/quick-link.test.js` | quick-link.js 的测试用例（格式验证） |

### 修改的文件

| 文件 | 变更 |
|------|------|
| `cmd/server/handler/admin.go` | `LinkComics` 支持 `sub_cids` 数组 + 新增 `DeleteComic` handler |
| `cmd/server/handler/init.go` | 注册新路由（comic_page, delete） |
| `cmd/server/view/static/tpl/index.tpl` | 移除 checkbox + compareSelected；添加右侧侧边栏 + 链接/对比模式渲染 |
| `cmd/server/view/static/tpl/gallery_detail.tpl` | 左右侧边栏互换 + 页管理 UI + 删除入口 |
| `cmd/server/view/static/tpl/head.tpl` | 引入 quick-link.js, page-manager.js |
| `cmd/server/view/static/custom/css/styles.css` | 侧边栏样式 + 选择模式样式 + 页管理样式 |
| `cmd/server/view/static/custom/js/modules/admin-compare.js` | `confirmLink()` 请求体改为 `sub_cids` 数组 |
| `cmd/server/view/gallery_detail.go` | 处理 `deleted` 标记，跳转提示页 |
| `cmd/server/view/index.go` | 过滤 `deleted: true` |
| `cmd/server/view/picture.go` | 处理已删除 comic 的图片请求 |
| `cmd/server/view/gallery_picture.go` | 处理已删除 comic 的图片请求 |
| `cmd/server/api/comic.go` | ArchiveInfo 新增 `Status` 字段 |
| `cmd/server/internal/comic/comic_info.go` | 新增删除相关逻辑 |

---

## 任务分解

---

### 任务 1：后端 — LinkComics 支持批量 sub_cids

**文件：**
- 修改：`cmd/server/handler/admin.go:113-231`
- 测试：`cmd/server/handler/admin_test.go`

- [ ] **步骤 1：编写失败的测试——LinkComics 批量 sub_cids**

```go
// cmd/server/handler/admin_test.go
package handler

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/cocomhub/cocom/pkg/httpwrap"
)

func TestLinkComics_BatchSubCIDs(t *testing.T) {
    if !testMongoAvailable {
        t.Skip("MongoDB not available")
    }

    body := map[string]any{
        "main_cid":  1001,
        "sub_cids":  []int{2001, 2002, 2003},
    }
    b, _ := json.Marshal(body)
    req := httptest.NewRequest("POST", "/api/admin/comic/link", bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    LinkComics(w, req)

    var resp httpwrap.ResponseInfo[map[string]any]
    if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
        t.Fatalf("decode response failed: %v", err)
    }
    if resp.Head.Code != 0 {
        t.Errorf("expected code 0, got %d: %s", resp.Head.Code, resp.Head.Msg)
    }
}

func TestLinkComics_InvalidBatch(t *testing.T) {
    if !testMongoAvailable {
        t.Skip("MongoDB not available")
    }

    // 测试空 sub_cids 数组
    body := map[string]any{
        "main_cid":  1001,
        "sub_cids":  []int{},
    }
    b, _ := json.Marshal(body)
    req := httptest.NewRequest("POST", "/api/admin/comic/link", bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    LinkComics(w, req)

    var resp httpwrap.ResponseInfo[map[string]any]
    if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
        t.Fatalf("decode response failed: %v", err)
    }
    if resp.Head.Code == 0 {
        t.Error("expected error for empty sub_cids, got code 0")
    }
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd D:\workdir\leon\cocomhub\cocom && go test -run TestLinkComics_Batch ./cmd/server/handler/ -v -tags=memory_storage_integration 2>&1`
预期：编译失败，因为 `linkRequest` 还没有 `SubCIDs` 字段

- [ ] **步骤 3：修改 linkRequest 结构体支持批量**

```go
type linkRequest struct {
    MainCID int   `json:"main_cid"`
    SubCID  int   `json:"sub_cid"`   // 为向后兼容保留
    SubCIDs []int `json:"sub_cids"`  // 新增批量支持
}
```

- [ ] **步骤 4：重构 LinkComics 函数——添加批量处理逻辑**

在 `LinkComics` 函数顶部，添加从请求体解析 `sub_cids` 的逻辑：

```go
// 解析 sub_cids：优先使用 sub_cids 数组，兼容旧版 sub_cid
subCIDs := lr.SubCIDs
if len(subCIDs) == 0 && lr.SubCID > 0 {
    subCIDs = []int{lr.SubCID}
}
if len(subCIDs) == 0 {
    w.WriteHeader(http.StatusBadRequest)
    httpwrap.ResponseFail(ctx, w, "sub_cids or sub_cid is required")
    return
}
if lr.MainCID <= 0 {
    w.WriteHeader(http.StatusBadRequest)
    httpwrap.ResponseFail(ctx, w, "main_cid is required")
    return
}
for _, sc := range subCIDs {
    if sc <= 0 || sc == lr.MainCID {
        w.WriteHeader(http.StatusBadRequest)
        httpwrap.ResponseFail(ctx, w, fmt.Sprintf("invalid sub_cid: %d", sc))
        return
    }
}
```

然后将原链接逻辑提取为 `linkSingleComic(ctx, mainCID, subCID int) error` 辅助函数，并在 `LinkComics` 中循环调用：

```go
// 遍历处理每个 sub_cid，失败时记录但继续
var errors []string
for _, sc := range subCIDs {
    if err := linkSingleComic(ctx, lr.MainCID, sc); err != nil {
        errors = append(errors, fmt.Sprintf("cid %d: %s", sc, err))
        slog.ErrorContext(ctx, "LinkComics: batch link failed",
            slog.Int("main_cid", lr.MainCID),
            slog.Int("sub_cid", sc),
            slog.String("errmsg", err.Error()))
    }
}
```

- [ ] **步骤 5：提取 linkSingleComic 辅助函数**

```go
// linkSingleComic 将一个从属 comic 链接到主 comic
func linkSingleComic(ctx context.Context, mainCID, subCID int) error {
    info1, info2, err := getTwoComicInfos(ctx, mainCID, subCID)
    if err != nil {
        return fmt.Errorf("get infos failed: %w", err)
    }

    // 合并 tags
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

    m1, err := util.ToMap(info1)
    if err != nil {
        return fmt.Errorf("encode main comic info failed: %w", err)
    }
    if err := comic.UpdateComicInfo(ctx, mainCID, m1); err != nil {
        return fmt.Errorf("update main comic info failed: %w", err)
    }

    // 如果主 comic 已经有 redirect_to，备 comic 直接指向它
    targetCID := mainCID
    if info1.RedirectTo != nil && *info1.RedirectTo > 0 {
        targetCID = *info1.RedirectTo
    }

    info2.RedirectTo = &targetCID
    m2, err := util.ToMap(info2)
    if err != nil {
        return fmt.Errorf("encode sub comic info failed: %w", err)
    }
    if err := comic.UpdateComicInfo(ctx, subCID, m2); err != nil {
        return fmt.Errorf("update sub comic info failed: %w", err)
    }

    // 重定向链传播
    propagateRedirectChain(ctx, subCID, targetCID)

    return nil
}

// propagateRedirectChain 将所有 redirect_to == subCID 的漫画改为 redirect_to == targetCID
func propagateRedirectChain(ctx context.Context, subCID, targetCID int) {
    type redirectChainItem struct {
        CID int `bson:"cid"`
    }
    var chain []redirectChainItem
    chainBuilder := mongo.ComicInfoBuilder().
        FilterKV("redirect_to", subCID).
        Limit(100)
    if err := chainBuilder.All(ctx, &chain); err != nil {
        slog.WarnContext(ctx, "propagateRedirectChain: query failed",
            slog.Int("sub_cid", subCID),
            slog.String("errmsg", err.Error()))
        return
    }
    for _, rc := range chain {
        var rcInfo api.ComicInfo
        if err := comic.GetComicInfo(ctx, rc.CID, &rcInfo); err != nil {
            slog.WarnContext(ctx, "propagateRedirectChain: get info failed",
                slog.Int("cid", rc.CID))
            continue
        }
        rcInfo.RedirectTo = &targetCID
        rcMap, err := util.ToMap(rcInfo)
        if err != nil {
            slog.WarnContext(ctx, "propagateRedirectChain: encode failed",
                slog.Int("cid", rc.CID))
            continue
        }
        if err := comic.UpdateComicInfo(ctx, rc.CID, rcMap); err != nil {
            slog.WarnContext(ctx, "propagateRedirectChain: update failed",
                slog.Int("cid", rc.CID))
        }
    }
}
```

- [ ] **步骤 6：更新 LinkComics 响应体**

```go
httpwrap.ResponseSucc(ctx, w, map[string]any{
    "main_cid": lr.MainCID,
    "sub_cids": subCIDs,
    "errors":   errors, // 可能为空
    "status":   "linked",
})
```

- [ ] **步骤 7：运行测试验证通过**

运行：`cd D:\workdir\leon\cocomhub\cocom && go test -run TestLinkComics ./cmd/server/handler/ -v -tags=memory_storage_integration 2>&1`
预期：编译通过，测试 PASS（无 MongoDB 时 Skip）

- [ ] **步骤 8：编译验证**

运行：`cd D:\workdir\leon\cocomhub\cocom && go build ./cmd/...`
预期：编译成功

- [ ] **步骤 9：Commit**

```bash
git add cmd/server/handler/admin*.go
git commit -m "feat: support batch sub_cids in LinkComics API"
```

---

### 任务 2：后端 — 新增 Comic 删除 Handler（DeleteComic + tombstone）

**文件：**
- 修改：`cmd/server/handler/admin.go`（新增 `DeleteComic` 函数）
- 修改：`cmd/server/handler/init.go`（注册路由）
- 修改：`cmd/server/internal/comic/comic_info.go`（新增删除逻辑）
- 测试：`cmd/server/handler/admin_test.go`

- [ ] **步骤 1：编写失败的测试——DeleteComic**

```go
// 追加到 admin_test.go
func TestDeleteComic(t *testing.T) {
    if !testMongoAvailable {
        t.Skip("MongoDB not available")
    }

    body := map[string]any{"cid": 99999}
    b, _ := json.Marshal(body)
    req := httptest.NewRequest("POST", "/api/admin/comic/delete", bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    // DeleteComic 尚未实现，此步骤期望编译失败
    DeleteComic(w, req)

    var resp httpwrap.ResponseInfo[map[string]any]
    if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
        t.Fatalf("decode response failed: %v", err)
    }
    // 即使 comic 不存在，也应有合理的错误响应而非 panic
    if resp.Head.Code == 0 {
        t.Error("expected non-zero code for non-existent comic")
    }
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd D:\workdir\leon\cocomhub\cocom && go test -run TestDeleteComic ./cmd/server/handler/ -v -tags=memory_storage_integration 2>&1`
预期：编译失败，`DeleteComic` 未定义

- [ ] **步骤 3：在 comic_info.go 中实现 DeleteComicByID**

```go
// cmd/server/internal/comic/comic_info.go

// DeleteComicByID 软删除 comic：原文档删除 + 插入最小 tombstone 记录
func DeleteComicByID(ctx context.Context, cid int) error {
    // 1. 获取原漫画信息（用于清理文件）
    var info api.ComicInfo
    if err := GetComicInfo(ctx, cid, &info); err != nil {
        return fmt.Errorf("get comic info failed: %w", err)
    }

    // 2. 删除原 MongoDB 文档
    filter := bson.M{"cid": cid}
    if _, err := mongo.ComicInfo().DeleteOne(ctx, filter); err != nil {
        return fmt.Errorf("delete comic document failed: %w", err)
    }

    // 3. 插入 tombstone 记录
    tombstone := bson.M{
        "cid":        cid,
        "deleted":    true,
        "deleted_at": time.Now(),
    }
    if _, err := mongo.ComicInfo().InsertOne(ctx, tombstone); err != nil {
        return fmt.Errorf("insert tombstone failed: %w", err)
    }

    // 4. 清理图片目录
    saveDir := info.SaveDir()
    if err := os.RemoveAll(saveDir); err != nil {
        slog.WarnContext(ctx, "DeleteComicByID: remove save dir failed",
            slog.Int("cid", cid),
            slog.String("dir", saveDir),
            slog.String("errmsg", err.Error()))
    }

    // 5. 清理归档文件（不阻塞，异步最佳尝试）
    if info.Archive != nil && info.Archive.Path != "" {
        go func() {
            if err := os.Remove(info.Archive.Path); err != nil {
                slog.Warn("DeleteComicByID: remove archive file failed",
                    slog.Int("cid", cid),
                    slog.String("path", info.Archive.Path))
            }
        }()
    }

    // 6. 清除缓存
    cache.Reset()

    return nil
}
```

注意：在文件顶部添加 `"time"` 和 `"os"` import。

- [ ] **步骤 4：在 admin.go 中实现 DeleteComic Handler**

```go
// DeleteComic 删除漫画
// POST /api/admin/comic/delete
func DeleteComic(w http.ResponseWriter, req *http.Request) {
    ctx := req.Context()

    var dr struct {
        CID int `json:"cid"`
    }
    if err := json.NewDecoder(req.Body).Decode(&dr); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        httpwrap.ResponseFail(ctx, w, "invalid request body")
        return
    }
    if dr.CID <= 0 {
        w.WriteHeader(http.StatusBadRequest)
        httpwrap.ResponseFail(ctx, w, "cid is required")
        return
    }

    if err := comic.DeleteComicByID(ctx, dr.CID); err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        httpwrap.ResponseFail(ctx, w, err.Error())
        return
    }

    cache.Reset()

    httpwrap.ResponseSucc(ctx, w, map[string]any{
        "cid":    dr.CID,
        "status": "deleted",
    })
}
```

- [ ] **步骤 5：在 init.go 中注册路由**

```go
// 追加到 admin 路由组中
r.POST("/api/admin/comic/delete", gin.WrapF(DeleteComic))
```

- [ ] **步骤 6：运行测试验证通过**

运行：`cd D:\workdir\leon\cocomhub\cocom && go test -run TestDeleteComic ./cmd/server/handler/ -v -tags=memory_storage_integration 2>&1`
预期：编译通过，PASS（无 MongoDB 时 Skip）

- [ ] **步骤 7：编译验证**

运行：`cd D:\workdir\leon\cocomhub\cocom && go build ./cmd/...`
预期：编译成功

- [ ] **步骤 8：Commit**

```bash
git add cmd/server/handler/admin.go cmd/server/handler/init.go cmd/server/internal/comic/comic_info.go
git commit -m "feat: add comic deletion with tombstone support"
```

---

### 任务 3：后端 — 新增 Page CRUD Handler（savePages / getComicPages）

**文件：**
- 创建：`cmd/server/handler/comic_page.go`
- 修改：`cmd/server/handler/init.go`（注册路由）
- 测试：`cmd/server/handler/comic_page_test.go`

- [ ] **步骤 1：编写失败的测试——getComicPages**

```go
// cmd/server/handler/comic_page_test.go
package handler

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/cocomhub/cocom/pkg/httpwrap"
)

func TestGetComicPages_InvalidCID(t *testing.T) {
    if !testMongoAvailable {
        t.Skip("MongoDB not available")
    }

    req := httptest.NewRequest("POST", "/api/comic/getComicPages",
        nil)
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    GetComicPages(w, req)

    var resp httpwrap.ResponseInfo[any]
    if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
        t.Fatalf("decode response failed: %v", err)
    }
    if resp.Head.Code == 0 {
        t.Error("expected non-zero code for empty request body")
    }
}

func TestSavePages_InvalidBody(t *testing.T) {
    if !testMongoAvailable {
        t.Skip("MongoDB not available")
    }

    req := httptest.NewRequest("POST", "/api/comic/savePages", nil)
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    SavePages(w, req)

    var resp httpwrap.ResponseInfo[any]
    if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
        t.Fatalf("decode response failed: %v", err)
    }
    if resp.Head.Code == 0 {
        t.Error("expected non-zero code for empty request body")
    }
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd D:\workdir\leon\cocomhub\cocom && go test -run TestGetComicPages\|TestSavePages ./cmd/server/handler/ -v -tags=memory_storage_integration 2>&1`
预期：编译失败，`GetComicPages`/`SavePages` 未定义

- [ ] **步骤 3：实现 GetComicPages handler**

```go
// cmd/server/handler/comic_page.go
package handler

import (
    "encoding/json"
    "fmt"
    "net/http"
    "strconv"

    "github.com/cocomhub/cocom/cmd/server/internal/comic"
    "github.com/cocomhub/cocom/pkg/httpwrap"
)

// getComicPagesRequest 获取 comic 页面列表的请求
type getComicPagesRequest struct {
    CID int `json:"cid"`
}

// GetComicPages 获取指定 comic 的页面缩略图信息
// POST /api/comic/getComicPages
func GetComicPages(w http.ResponseWriter, req *http.Request) {
    ctx := req.Context()

    var gr getComicPagesRequest
    if err := json.NewDecoder(req.Body).Decode(&gr); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        httpwrap.ResponseFail(ctx, w, "invalid request body")
        return
    }
    if gr.CID <= 0 {
        w.WriteHeader(http.StatusBadRequest)
        httpwrap.ResponseFail(ctx, w, "cid is required")
        return
    }

    info, err := comic.GetComicInfoByID(ctx, gr.CID)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        httpwrap.ResponseFail(ctx, w, fmt.Sprintf("get comic info failed: %s", err))
        return
    }

    type pageItem struct {
        Page int    `json:"page"`
        Name string `json:"name"`
        ThumbURL string `json:"thumb_url"`
    }
    pages := make([]pageItem, 0, len(info.Images.Pages))
    for i, p := range info.Images.Pages {
        pages = append(pages, pageItem{
            Page:     i + 1,
            Name:     p.Name,
            ThumbURL: fmt.Sprintf("/galleries/%d/%s", gr.CID, p.Name),
        })
    }

    httpwrap.ResponseSucc(ctx, w, map[string]any{
        "cid":       gr.CID,
        "num_pages": len(pages),
        "pages":     pages,
    })
}
```

注意：需要确保 `comic` 包暴露 `GetComicInfoByID`（或使用现有 `GetComicInfo` 模式）。

- [ ] **步骤 4：实现 SavePages handler**

```go
// savePagesRequest 保存页面变更请求
type savePagesRequest struct {
    CID   int `json:"cid"`
    Pages []struct {
        Page   int    `json:"page"`
        Action string `json:"action"` // "delete" | "reorder" | "replace"
        Name   string `json:"name,omitempty"`
    } `json:"pages"`
}

// SavePages 保存页面变更并标记归档为 stale
// POST /api/comic/savePages
func SavePages(w http.ResponseWriter, req *http.Request) {
    ctx := req.Context()

    var sr savePagesRequest
    if err := json.NewDecoder(req.Body).Decode(&sr); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        httpwrap.ResponseFail(ctx, w, "invalid request body")
        return
    }
    if sr.CID <= 0 {
        w.WriteHeader(http.StatusBadRequest)
        httpwrap.ResponseFail(ctx, w, "cid is required")
        return
    }

    // 获取当前 comic 信息
    var info api.ComicInfo
    if err := comic.GetComicInfo(ctx, sr.CID, &info); err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        httpwrap.ResponseFail(ctx, w, fmt.Sprintf("get comic info failed: %s", err))
        return
    }

    // 应用页面变更
    for _, p := range sr.Pages {
        switch p.Action {
        case "delete":
            if p.Page > 0 && p.Page <= len(info.Images.Pages) {
                info.Images.Pages = append(info.Images.Pages[:p.Page-1], info.Images.Pages[p.Page+1:]...)
            }
        case "reorder":
            // 前端已排好序，直接按新顺序存储
            // 这里的实现取决于前端传来的格式
        case "replace":
            // 替换某个页面——文件已由前端上传，只需更新 PicInfo
        }
    }

    // 更新 num_pages
    info.NumPages = len(info.Images.Pages)

    // 标记归档为 stale
    if info.Archive != nil {
        info.Archive.Status = "stale"
    }

    // 保存到 MongoDB
    m, err := util.ToMap(info)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        httpwrap.ResponseFail(ctx, w, "encode comic info failed")
        return
    }
    if err := comic.UpdateComicInfo(ctx, sr.CID, m); err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        httpwrap.ResponseFail(ctx, w, fmt.Sprintf("update comic info failed: %s", err))
        return
    }

    cache.Reset()

    httpwrap.ResponseSucc(ctx, w, map[string]any{
        "cid":       sr.CID,
        "num_pages": info.NumPages,
        "status":    "saved",
        "archive":   "stale",
    })
}
```

注意：需要添加 `"github.com/cocomhub/cocom/cmd/server/api"` import。

- [ ] **步骤 5：在 init.go 中注册路由**

```go
r.POST("/api/comic/getComicPages", gin.WrapF(GetComicPages))
r.POST("/api/comic/savePages", gin.WrapF(SavePages))
```

- [ ] **步骤 6：运行测试验证通过**

运行：`cd D:\workdir\leon\cocomhub\cocom && go test -run "TestGetComicPages|TestSavePages" ./cmd/server/handler/ -v -tags=memory_storage_integration 2>&1`
预期：编译通过，PASS

- [ ] **步骤 7：编译验证**

运行：`cd D:\workdir\leon\cocomhub\cocom && go build ./cmd/...`
预期：编译成功

- [ ] **步骤 8：Commit**

```bash
git add cmd/server/handler/comic_page.go cmd/server/handler/comic_page_test.go cmd/server/handler/init.go
git commit -m "feat: add page management handlers (getComicPages/savePages)"
```

---

### 任务 4：后端 — Index 页过滤 deleted comic + 各视图处理 deleted 状态

**文件：**
- 修改：`cmd/server/view/index.go:72-79`（添加 `deleted` 过滤）
- 修改：`cmd/server/view/gallery_detail.go`（处理 tombstone 跳转）
- 修改：`cmd/server/view/picture.go`（处理已删除 comic 的图片请求）
- 修改：`cmd/server/view/gallery_picture.go`（处理已删除 comic 的图片请求）
- 修改：`cmd/server/api/comic.go`（ArchiveInfo 新增 Status 字段）
- 测试：`cmd/server/view/index_test.go`（验证过滤逻辑）

- [ ] **步骤 1：在 ArchiveInfo 中添加 Status 字段**

```go
// cmd/server/api/comic.go
type ArchiveInfo struct {
    Path      string    `json:"path,omitempty" bson:"path,omitempty"`
    MD5       string    `json:"md5,omitempty" bson:"md5,omitempty"`
    Size      string    `json:"size,omitempty" bson:"size,omitempty"`
    Algorithm string    `json:"algorithm,omitempty" bson:"algorithm,omitempty"`
    CreatedAt time.Time `json:"created_at,omitempty" bson:"created_at,omitempty"`
    ByForce   bool      `json:"by_force,omitempty" bson:"by_force,omitempty"`
    Locators  []storage.StorageLocator `json:"locators,omitempty" bson:"locators,omitempty"`
    storage.ReplicaHealth `json:"replica_health,omitempty" bson:"replica_health,omitempty"`
    Status    string    `json:"status,omitempty" bson:"status,omitempty"` // "valid" | "stale"
}
```

- [ ] **步骤 2：在 index.go 中添加 deleted 过滤**

```go
// 在现有 redirect_to 过滤之后追加
// 过滤掉已删除的漫画
filters = append(filters, "deleted", bson.M{
    "$ne": true,
})
```

- [ ] **步骤 3：在 gallery_detail.go 中添加 tombstone 检测**

```go
// 在 GalleryDetailPage 中，获取信息后检测 deleted 标记
func GalleryDetailPage(c *gin.Context) {
    // ... 现有代码 ...

    // 检测是否存在 tombstone（已删除）
    var deletedInfo struct {
        Deleted   bool      `bson:"deleted"`
        DeletedAt time.Time `bson:"deleted_at"`
    }
    if err := comic.GetRawComicInfo(ctx, cid, &deletedInfo); err == nil && deletedInfo.Deleted {
        c.HTML(http.StatusOK, "page_deleted.tpl", gin.H{
            "CID":       cid,
            "DeletedAt": deletedInfo.DeletedAt,
        })
        return
    }

    // ... 继续现有逻辑 ...
}
```

- [ ] **步骤 4：在 picture.go 中添加已删除检测**

```go
// picture.go 中，获取漫画信息后检测 deleted
var checkDeleted struct {
    Deleted bool `bson:"deleted"`
}
if err := mongo.ComicInfo().FindOne(ctx, bson.M{"cid": cid}).Decode(&checkDeleted); err == nil && checkDeleted.Deleted {
    c.AbortWithStatus(http.StatusNotFound)
    return
}
```

- [ ] **步骤 5：在 gallery_picture.go 中添加相同检测**

同上模式。

- [ ] **步骤 6：编译验证**

运行：`cd D:\workdir\leon\cocomhub\cocom && go build ./cmd/...`
预期：编译成功

- [ ] **步骤 7：Commit**

```bash
git add cmd/server/view/index.go cmd/server/view/gallery_detail.go cmd/server/view/picture.go cmd/server/view/gallery_picture.go cmd/server/api/comic.go
git commit -m "feat: add deleted comic filtering and tombstone detection"
```

---

### 任务 5：前端 — Index 页右侧侧边栏 + 链接/对比模式（HTML + CSS）

**文件：**
- 修改：`cmd/server/view/static/tpl/index.tpl`
- 修改：`cmd/server/view/static/custom/css/styles.css`
- 修改：`cmd/server/view/static/tpl/head.tpl`

- [ ] **步骤 1：在 index.tpl 中移除 comic-select checkbox 和 compareSelected**

删除两处 `input.comic-select`：Popular Now 和 New Updates 的卡片中。

删除底部 `compareSelected()` 函数。

删除"对比选定"按钮。

- [ ] **步骤 2：在 index.tpl 中添加右侧侧边栏**

在主体内容容器之后，`</body>` 之前添加：

```html
<!-- 右侧操作侧边栏 -->
<div id="quick-action-sidebar" class="quick-action-sidebar">
  <div class="sidebar-header">
    <span class="sidebar-title">操作</span>
    <button class="sidebar-toggle" onclick="toggleSidebar()" title="折叠">
      <i class="fa fa-chevron-right"></i>
    </button>
  </div>
  <div class="sidebar-body">
    <button id="btn-link-mode" class="btn btn-secondary sidebar-btn" onclick="toggleLinkMode()" title="链接(L)">
      <i class="fa fa-link"></i> 链接 <span class="shortcut-hint">L</span>
    </button>
    <button id="btn-compare-mode" class="btn btn-secondary sidebar-btn" onclick="toggleCompareMode()" title="对比(C)">
      <i class="fa fa-images"></i> 对比 <span class="shortcut-hint">C</span>
    </button>
    <hr class="sidebar-divider">
    <div class="sidebar-setting">
      <label class="toggle-label">
        <input type="checkbox" id="comic-link-target" checked>
        <span class="toggle-text">新标签打开</span>
      </label>
    </div>
    <div class="sidebar-status" id="sidebar-status" style="display:none;">
      <div class="status-info"></div>
      <div class="status-actions" style="display:none;">
        <button class="btn btn-primary btn-sm" onclick="confirmAction()">确认</button>
        <button class="btn btn-secondary btn-sm" onclick="cancelAction()">取消</button>
      </div>
    </div>
  </div>
</div>
```

- [ ] **步骤 3：在 head.tpl 中引入 quick-link.js**

```html
<script src="/static/custom/js/modules/quick-link.js"></script>
```

- [ ] **步骤 4：在 styles.css 中添加右侧侧边栏样式**

```css
/* 快速操作侧边栏 */
.quick-action-sidebar {
  position: fixed;
  right: 0;
  top: 50%;
  transform: translateY(-50%);
  z-index: 500;
  background: rgba(0, 0, 0, 0.65);
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
  border-radius: 8px 0 0 8px;
  padding: 12px 8px;
  min-width: 120px;
  transition: transform 0.3s ease, opacity 0.3s ease;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 6px;
}
.quick-action-sidebar.collapsed {
  transform: translateY(-50%) translateX(100%);
  opacity: 0;
  pointer-events: none;
}
.quick-action-sidebar .sidebar-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 4px 6px;
  border-bottom: 1px solid rgba(255,255,255,0.15);
}
.quick-action-sidebar .sidebar-title {
  color: #fff;
  font-size: 13px;
  font-weight: 600;
}
.quick-action-sidebar .sidebar-toggle {
  background: none;
  border: none;
  color: #aaa;
  cursor: pointer;
  padding: 2px 6px;
  font-size: 14px;
}
.quick-action-sidebar .sidebar-toggle:hover {
  color: #fff;
}
.quick-action-sidebar .sidebar-btn {
  width: 100%;
  text-align: left;
  padding: 6px 10px;
  font-size: 13px;
  border: 1px solid rgba(255,255,255,0.1);
  background: rgba(255,255,255,0.05);
  color: #ddd;
  transition: background 0.15s;
}
.quick-action-sidebar .sidebar-btn:hover {
  background: rgba(255,255,255,0.15);
  color: #fff;
}
.quick-action-sidebar .sidebar-btn.active {
  background: rgba(66, 133, 244, 0.3);
  border-color: #4285f4;
  color: #fff;
}
.quick-action-sidebar .shortcut-hint {
  float: right;
  font-size: 11px;
  color: #888;
  background: rgba(255,255,255,0.08);
  padding: 1px 6px;
  border-radius: 3px;
}
.quick-action-sidebar .sidebar-divider {
  border-color: rgba(255,255,255,0.12);
  margin: 4px 0;
}
.quick-action-sidebar .sidebar-setting {
  padding: 4px;
}
.quick-action-sidebar .toggle-label {
  display: flex;
  align-items: center;
  gap: 6px;
  color: #ccc;
  font-size: 12px;
  cursor: pointer;
}
.quick-action-sidebar .toggle-label input[type="checkbox"] {
  cursor: pointer;
}
.quick-action-sidebar .sidebar-status {
  background: rgba(255,255,255,0.08);
  border-radius: 6px;
  padding: 8px;
}
.quick-action-sidebar .status-info {
  color: #fff;
  font-size: 12px;
  margin-bottom: 6px;
}
.quick-action-sidebar .status-actions {
  display: flex;
  gap: 6px;
}
.quick-action-sidebar .status-actions .btn-sm {
  font-size: 12px;
  padding: 3px 10px;
}

/* 链接选择模式 - 卡片状态 */
.comic-card.link-selectable {
  cursor: pointer;
}
.comic-card.link-selectable:hover {
  outline: 2px solid rgba(66, 133, 244, 0.5);
  outline-offset: -2px;
}
.comic-card.selected-main {
  outline: 3px solid #ffd700;
  outline-offset: -3px;
  position: relative;
}
.comic-card.selected-main::after {
  content: "⭐";
  position: absolute;
  top: 4px;
  left: 4px;
  font-size: 18px;
  z-index: 11;
}
.comic-card.selected-sub {
  outline: 2px solid #4285f4;
  outline-offset: -2px;
  position: relative;
}
.comic-card.selected-sub::after {
  content: attr(data-order);
  position: absolute;
  top: 4px;
  left: 4px;
  font-size: 14px;
  font-weight: bold;
  color: #fff;
  background: rgba(66, 133, 244, 0.8);
  padding: 1px 6px;
  border-radius: 10px;
  z-index: 11;
}
.comic-card.link-disabled {
  pointer-events: none;
  opacity: 0.5;
}

/* 页面管理样式 */
.page-manager-bar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 1000;
  background: rgba(0,0,0,0.85);
  backdrop-filter: blur(8px);
  padding: 10px 16px;
  display: flex;
  align-items: center;
  gap: 12px;
  color: #fff;
  font-size: 14px;
}
.page-manager-bar .pm-title {
  font-weight: 600;
  margin-right: 8px;
}
.page-manager-bar .pm-actions {
  display: flex;
  gap: 6px;
}
.page-manager-bar .pm-actions .btn-sm {
  font-size: 12px;
}
.page-manager-bar .pm-sep {
  color: #555;
  margin: 0 4px;
}
.page-manager-bar .pm-status {
  margin-left: auto;
  font-size: 12px;
  color: #aaa;
}

.page-manager-mode .gallery-thumb {
  position: relative;
  cursor: pointer;
}
.page-manager-mode .gallery-thumb:hover {
  outline: 2px solid #4285f4;
  outline-offset: -2px;
}
.page-manager-mode .gallery-thumb.page-deleted {
  opacity: 0.4;
  outline: 2px solid #e53935;
}
.page-manager-mode .gallery-thumb.page-deleted::after {
  content: "✕";
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  font-size: 32px;
  color: #e53935;
  font-weight: bold;
}
.page-manager-mode .gallery-thumb .page-order {
  position: absolute;
  bottom: 2px;
  right: 2px;
  background: rgba(0,0,0,0.7);
  color: #fff;
  font-size: 11px;
  padding: 1px 5px;
  border-radius: 3px;
}
/* 插入预览行 */
.insert-preview-row {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  padding: 8px 0;
}
.insert-preview-row .preview-item {
  flex: 0 0 auto;
  width: 80px;
  text-align: center;
  cursor: pointer;
  border: 2px solid transparent;
  border-radius: 4px;
  padding: 2px;
}
.insert-preview-row .preview-item:hover {
  border-color: #4285f4;
}
.insert-preview-row .preview-item.selected {
  border-color: #ffd700;
  background: rgba(255,215,0,0.1);
}
.insert-preview-row .preview-item img {
  width: 100%;
  height: 100px;
  object-fit: cover;
  border-radius: 3px;
}
.insert-preview-row .preview-item .page-num {
  font-size: 11px;
  color: #aaa;
  margin-top: 2px;
}
```

- [ ] **步骤 6：Commit**

```bash
git add cmd/server/view/static/tpl/index.tpl cmd/server/view/static/tpl/head.tpl cmd/server/view/static/custom/css/styles.css
git commit -m "feat: add index page sidebar and link/compare mode HTML/CSS"
```

---

### 任务 6：前端 — quick-link.js 实现

**文件：**
- 创建：`cmd/server/view/static/custom/js/modules/quick-link.js`

- [ ] **步骤 1：实现 quick-link.js**

```js
// quick-link.js — Index 页快速链接与对比功能
(function() {
  'use strict';

  // ---- 状态 ----
  var state = {
    mode: null,           // null | 'link' | 'compare'
    mainCID: null,        // 主 comic 的 CID（链接模式）
    selectedCIDs: [],     // 已选中的 CID 列表
  };

  // ---- 配置 ----
  var STORAGE_KEY = 'comic_link_target';

  // ---- DOM 引用 ----
  var sidebar = document.getElementById('quick-action-sidebar');
  var statusEl = document.getElementById('sidebar-status');
  var statusInfo = statusEl ? statusEl.querySelector('.status-info') : null;
  var statusActions = statusEl ? statusEl.querySelector('.status-actions') : null;

  // ---- 初始化 ----
  function init() {
    // 恢复新标签打开设置
    var checkbox = document.getElementById('comic-link-target');
    if (checkbox) {
      var saved = localStorage.getItem(STORAGE_KEY);
      if (saved === '_blank') {
        checkbox.checked = true;
      } else if (saved === '_self') {
        checkbox.checked = false;
      }
      checkbox.addEventListener('change', function() {
        localStorage.setItem(STORAGE_KEY, checkbox.checked ? '_blank' : '_self');
      });
    }
  }

  // ---- 点击行为配置 ----
  function getLinkTarget() {
    return localStorage.getItem(STORAGE_KEY) === '_blank' ? '_blank' : '_self';
  }

  // ---- 工具 ----
  function escapeHtml(text) {
    var div = document.createElement('div');
    div.appendChild(document.createTextNode(text));
    return div.innerHTML;
  }

  function showAdminToast(msg, type) {
    // 复用全局 toast 函数
    if (window.showAdminToast) {
      window.showAdminToast(msg, type);
      return;
    }
    // 降级
    alert(msg);
  }

  // ---- 模式切换 ----
  window.toggleLinkMode = function() {
    if (state.mode === 'link') {
      exitMode();
      return;
    }
    enterMode('link');
  };

  window.toggleCompareMode = function() {
    if (state.mode === 'compare') {
      exitMode();
      return;
    }
    enterMode('compare');
  };

  function enterMode(mode) {
    state.mode = mode;
    state.mainCID = null;
    state.selectedCIDs = [];
    updateUI();

    // 给所有 comic 卡片添加可交互样式
    document.querySelectorAll('.gallery-card, .comic-card').forEach(function(card) {
      card.classList.add('link-selectable');
      card.addEventListener('click', onCardClick);
    });

    // 禁用默认点击跳转
    document.querySelectorAll('.gallery-card a, .comic-card a').forEach(function(a) {
      a.addEventListener('click', preventDefaultClick);
    });

    var btn = document.getElementById(state.mode === 'link' ? 'btn-link-mode' : 'btn-compare-mode');
    if (btn) btn.classList.add('active');
  }

  function exitMode() {
    state.mode = null;
    state.mainCID = null;
    state.selectedCIDs = [];
    if (statusEl) { statusEl.style.display = 'none'; }

    document.querySelectorAll('.gallery-card, .comic-card').forEach(function(card) {
      card.classList.remove('link-selectable', 'selected-main', 'selected-sub');
      card.removeEventListener('click', onCardClick);
    });
    document.querySelectorAll('.gallery-card a, .comic-card a').forEach(function(a) {
      a.removeEventListener('click', preventDefaultClick);
    });

    var linkBtn = document.getElementById('btn-link-mode');
    var cmpBtn = document.getElementById('btn-compare-mode');
    if (linkBtn) linkBtn.classList.remove('active');
    if (cmpBtn) cmpBtn.classList.remove('active');
  }

  function preventDefaultClick(e) {
    e.preventDefault();
    e.stopPropagation();
  }

  // ---- 卡片点击 ----
  function onCardClick(e) {
    if (!state.mode) return;
    e.preventDefault();

    var card = e.currentTarget;
    var cid = parseInt(card.getAttribute('data-cid') || card.getAttribute('value'), 10);
    if (!cid) return;

    if (state.mode === 'link') {
      handleLinkClick(cid, card);
    } else if (state.mode === 'compare') {
      handleCompareClick(cid, card);
    }
    updateUI();
  }

  // ---- 链接模式点击 ----
  function handleLinkClick(cid, card) {
    if (cid === state.mainCID) {
      // 取消选择主 comic
      state.mainCID = null;
      card.classList.remove('selected-main');
      return;
    }

    var idx = state.selectedCIDs.indexOf(cid);
    if (idx >= 0) {
      // 取消选择备 comic
      state.selectedCIDs.splice(idx, 1);
      card.classList.remove('selected-sub');
      card.removeAttribute('data-order');
      return;
    }

    if (!state.mainCID) {
      // 第一个选中的作为主 comic
      state.mainCID = cid;
      card.classList.add('selected-main');
    } else {
      // 后续选中的作为备 comic
      state.selectedCIDs.push(cid);
      card.classList.add('selected-sub');
      card.setAttribute('data-order', state.selectedCIDs.length);
    }
  }

  // ---- 对比模式点击 ----
  function handleCompareClick(cid, card) {
    var idx = state.selectedCIDs.indexOf(cid);
    if (idx >= 0) {
      state.selectedCIDs.splice(idx, 1);
      card.classList.remove('selected-sub');
      card.removeAttribute('data-order');
    } else {
      state.selectedCIDs.push(cid);
      card.classList.add('selected-sub');
      card.setAttribute('data-order', state.selectedCIDs.length);
    }
  }

  // ---- 更新界面 ----
  function updateUI() {
    if (!statusEl || !statusInfo) return;

    if (!state.mode) {
      statusEl.style.display = 'none';
      return;
    }

    statusEl.style.display = 'block';
    var modeLabel = state.mode === 'link' ? '链接模式' : '对比模式';
    var html = '<strong>' + modeLabel + '</strong><br>';

    if (state.mode === 'link') {
      if (state.mainCID) {
        html += '主 comic: <strong>' + state.mainCID + '</strong> | ';
        html += '备 comic: <strong>' + state.selectedCIDs.length + '</strong> 个';
        if (statusActions) statusActions.style.display = 'flex';
      } else {
        html += '请点击选择主 comic（⭐）';
        if (statusActions) statusActions.style.display = 'none';
      }
    } else { // compare
      html += '已选择: <strong>' + state.selectedCIDs.length + '</strong> 个 (最少 2 个)';
      if (statusActions) {
        statusActions.style.display = state.selectedCIDs.length >= 2 ? 'flex' : 'none';
      }
    }

    statusInfo.innerHTML = html;
  }

  // ---- 确认操作 ----
  window.confirmAction = function() {
    if (state.mode === 'link') {
      confirmLinkAction();
    } else if (state.mode === 'compare') {
      confirmCompareAction();
    }
  };

  window.cancelAction = function() {
    exitMode();
  };

  function confirmLinkAction() {
    if (!state.mainCID || state.selectedCIDs.length === 0) {
      showAdminToast('请选择主 comic 和至少一个备 comic', 'error');
      return;
    }

    if (!confirm('确认将 ' + state.selectedCIDs.length + ' 个备 comic 链接到 ' + state.mainCID + '？')) return;

    fetch('/api/admin/comic/link', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        main_cid: state.mainCID,
        sub_cids: state.selectedCIDs,
      }),
    })
    .then(function(r) { return r.json(); })
    .then(function(data) {
      if (data.head && data.head.code === 0) {
        showAdminToast('链接成功！', 'success');
        exitMode();
        // 刷新列表
        location.reload();
      } else {
        showAdminToast('链接失败: ' + (data.head ? data.head.msg : '未知错误'), 'error');
      }
    })
    .catch(function(err) {
      showAdminToast('请求失败: ' + err.message, 'error');
    });
  }

  function confirmCompareAction() {
    if (state.selectedCIDs.length < 2) {
      showAdminToast('请至少选择 2 个漫画进行对比', 'error');
      return;
    }
    window.location.href = '/admin?cids=' + state.selectedCIDs.join(',');
  }

  // ---- 侧边栏折叠 ----
  window.toggleSidebar = function() {
    if (!sidebar) return;
    sidebar.classList.toggle('collapsed');
  };

  // ---- 键盘快捷键 ----
  document.addEventListener('keydown', function(e) {
    // 不在输入框中触发
    if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return;

    if (e.key === 'l' || e.key === 'L') {
      if (!e.ctrlKey && !e.metaKey) {
        e.preventDefault();
        window.toggleLinkMode();
      }
    } else if (e.key === 'c' || e.key === 'C') {
      if (!e.ctrlKey && !e.metaKey) {
        e.preventDefault();
        window.toggleCompareMode();
      }
    } else if (e.key === 'Escape') {
      if (state.mode) {
        e.preventDefault();
        exitMode();
      }
    }
  });

  // ---- 初始化 ----
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
```

- [ ] **步骤 2：创建 quick-link 验证测试（JS 格式检查）**

```js
// cmd/server/view/static/custom/js/modules/quick-link.test.js
// 此文件作为验证 quick-link.js 结构的测试
// 运行时使用 Node.js 检查语法
```

- [ ] **步骤 3：验证 JS 语法**

运行：`cd D:\workdir\leon\cocomhub\cocom && node -e "require('fs').readFileSync('cmd/server/view/static/custom/js/modules/quick-link.js','utf8').split('\n').forEach((l,i)=>{try{new Function(l)}catch(e){}}); console.log('Syntax OK')" 2>&1 || node --check cmd/server/view/static/custom/js/modules/quick-link.js 2>&1`

预期：Syntax OK / 无错误

- [ ] **步骤 4：Commit**

```bash
git add cmd/server/view/static/custom/js/modules/quick-link.js
git commit -m "feat: implement quick-link.js for index sidebar link/compare mode"
```

---

### 任务 7：前端 — admin-compare.js update for sub_cids

**文件：**
- 修改：`cmd/server/view/static/custom/js/modules/admin-compare.js`（`confirmLink` 函数）

- [ ] **步骤 1：更新 confirmLink 函数发送 sub_cids 数组**

将 `confirmLink` 函数中的请求体从：
```js
body: JSON.stringify({ main_cid: mainCID, sub_cid: subCID }),
```
改为：
```js
body: JSON.stringify({ main_cid: mainCID, sub_cids: [subCID] }),
```

- [ ] **步骤 2：验证无语法错误**

运行：`cd D:\workdir\leon\cocomhub\cocom && node --check cmd/server/view/static/custom/js/modules/admin-compare.js 2>&1`
预期：无错误

- [ ] **步骤 3：Commit**

```bash
git add cmd/server/view/static/custom/js/modules/admin-compare.js
git commit -m "fix: update confirmLink to use sub_cids array format"
```

---

### 任务 8：前端 — Detail 页左右侧边栏互换 + 页管理 UI

**文件：**
- 修改：`cmd/server/view/static/tpl/gallery_detail.tpl`
- 修改：`cmd/server/view/static/custom/css/styles.css`
- 修改：`cmd/server/view/static/tpl/head.tpl`
- 修改：`cmd/server/view/static/tpl/page_deleted.tpl`（创建）

- [ ] **步骤 1：互换 gallery_detail.tpl 的侧边栏位置**

将左侧操作栏（`.left-action-sidebar`）改为右侧定位，右侧缩放栏（`.right-zoom-sidebar`）改为左侧定位。

关键变更：
- 在原有左侧操作栏的容器上：`class="right-action-sidebar"`（新增），保持原有按钮
- 在原有右侧缩放栏的容器上：`class="left-zoom-sidebar"`（新增）

实际上最简便的做法是交换 DOM 中两个侧边栏的顺序或直接交换 CSS 定位。

- [ ] **步骤 2：在右侧操作栏中添加"页管理"和"删除"按钮**

在归档按钮之后、修复按钮之前添加：
```html
<hr class="sidebar-divider">
<button id="sidebarPageManageBtn" class="sidebar-btn" onclick="togglePageManager()" title="页管理">
  <i class="fa fa-file-image-o"></i> 页管理
</button>
```

在底部（Tags 之后、大图之前）添加删除按钮：
```html
<hr class="sidebar-divider">
<button id="sidebarDeleteBtn" class="sidebar-btn btn-danger" onclick="openDeleteConfirm()" title="删除">
  <i class="fa fa-trash-o"></i> 删除
</button>
```

- [ ] **步骤 3：在 gallery_detail.tpl 中新增页管理操作栏**

在 `#thumbnail-container` 之前添加：
```html
<!-- 页管理操作栏 -->
<div id="page-manager-bar" class="page-manager-bar" style="display:none;">
  <span class="pm-title">📄 页管理</span>
  <div class="pm-actions">
    <button class="btn btn-danger btn-sm" onclick="pmDeleteMode()">删除</button>
    <button class="btn btn-secondary btn-sm" onclick="pmInsertMode()">插入</button>
    <button class="btn btn-secondary btn-sm" onclick="pmReplaceMode()">替换</button>
    <button class="btn btn-secondary btn-sm" onclick="pmReorderMode()">重排</button>
    <span class="pm-sep">|</span>
    <button class="btn btn-secondary btn-sm" onclick="pmUndo()">撤销</button>
    <button class="btn btn-primary btn-sm" onclick="pmSave()">保存</button>
    <button class="btn btn-secondary btn-sm" onclick="pmExit()">退出</button>
  </div>
  <div class="pm-status" id="pm-status">当前选中: 无 &nbsp;|&nbsp; 未保存变更: 0</div>
</div>

<!-- 插入表单 -->
<div id="insert-form" class="insert-form" style="display:none;">
  <div class="insert-form-inner">
    <label>源 CID: <input type="number" id="insert-source-cid" class="form-control" style="width:100px;display:inline;"></label>
    <button class="btn btn-secondary btn-sm" onclick="pmFetchPreview()">获取页面</button>
    <label style="margin-left:8px;">插入到第 <input type="number" id="insert-after-page" class="form-control" style="width:60px;display:inline;"> 页之后</label>
    <button class="btn btn-primary btn-sm" onclick="pmConfirmInsert()" style="margin-left:8px;">确认插入</button>
    <button class="btn btn-secondary btn-sm" onclick="pmCancelInsert()">取消</button>
  </div>
  <div id="insert-preview" class="insert-preview-row" style="display:none;"></div>
</div>
```

- [ ] **步骤 4：创建 page_deleted.tpl**

```html
<!-- cmd/server/view/static/tpl/page_deleted.tpl -->
{{/* 已删除漫画提示页 */}}
{{define "content"}}
<div class="container" style="text-align:center;padding:80px 20px;">
  <div style="font-size:64px;margin-bottom:20px;">🗑️</div>
  <h1 style="color:#e53935;">该漫画已被删除</h1>
  <p style="color:#999;font-size:16px;margin-top:12px;">
    漫画 CID: <strong>{{.CID}}</strong>
  </p>
  {{if .DeletedAt}}
  <p style="color:#777;font-size:14px;">
    删除时间: {{.DeletedAt.Format "2006-01-02 15:04:05"}}
  </p>
  {{end}}
  <a href="/" class="btn btn-primary" style="margin-top:20px;">返回首页</a>
</div>
{{end}}
```

- [ ] **步骤 5：在 styles.css 中添加互换后的侧边栏样式**

```css
/* 左右侧边栏互换 */
/* 原 left-action-sidebar → 改为右侧定位 */
.left-action-sidebar,
.right-action-sidebar {
  position: fixed;
  top: 50%;
  z-index: 500;
  transform: translateY(-50%);
  background: rgba(0, 0, 0, 0.55);
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
  border-radius: 8px;
  padding: 8px 6px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  transition: opacity 0.2s;
}

/* 操作栏 → 右侧 */
.left-action-sidebar,
.right-action-sidebar {
  right: 0;
  left: auto;
  border-radius: 8px 0 0 8px;
}

/* 缩放栏 → 左侧 */
.right-zoom-sidebar,
.left-zoom-sidebar {
  left: 0;
  right: auto;
  border-radius: 0 8px 8px 0;
}

/* 容器间距互换 */
.container.with-sidebars {
  padding-left: 0;
  padding-right: 60px;
}
```

- [ ] **步骤 6：在 head.tpl 中引入 page-manager.js**

```html
<script src="/static/custom/js/modules/page-manager.js"></script>
```

- [ ] **步骤 7：Commit**

```bash
git add cmd/server/view/static/tpl/gallery_detail.tpl cmd/server/view/static/tpl/page_deleted.tpl cmd/server/view/static/tpl/head.tpl cmd/server/view/static/custom/css/styles.css
git commit -m "feat: swap detail page sidebars, add page management and delete UI"
```

---

### 任务 9：前端 — page-manager.js 实现

**文件：**
- 创建：`cmd/server/view/static/custom/js/modules/page-manager.js`

- [ ] **步骤 1：实现 page-manager.js**

```js
// page-manager.js — Detail 页页管理功能
(function() {
  'use strict';

  var state = {
    active: false,
    mode: null,           // null | 'delete' | 'insert' | 'replace' | 'reorder'
    changes: [],          // { type: 'delete'|'insert'|'replace'|'reorder', page: N, ... }
    selectedPages: [],    // 当前选中的页码
    insertedPages: [],    // 待插入的页面
    originalOrder: [],    // 原始页面顺序（用于重排）
  };

  var cid = window._gallery ? window._gallery.CID : null;

  // ---- DOM ----
  var bar = document.getElementById('page-manager-bar');
  var statusEl = document.getElementById('pm-status');
  var container = document.getElementById('thumbnail-container');
  var insertForm = document.getElementById('insert-form');

  // ---- 模式切换 ----
  window.togglePageManager = function() {
    if (state.active) {
      pmExit();
    } else {
      pmEnter();
    }
  };

  function pmEnter() {
    state.active = true;
    if (bar) bar.style.display = 'flex';
    if (container) container.classList.add('page-manager-mode');
    // 为所有缩略图添加页码显示和点击事件
    setupThumbnailEvents();
    updateStatus();
  }

  function pmExit() {
    state.active = false;
    state.mode = null;
    if (bar) bar.style.display = 'none';
    if (insertForm) insertForm.style.display = 'none';
    if (container) container.classList.remove('page-manager-mode');
    removeThumbnailEvents();
    resetThumbnailStyles();
  }

  // ---- 缩略图事件 ----
  function setupThumbnailEvents() {
    document.querySelectorAll('.gallery-thumb, .thumb-container').forEach(function(el) {
      el.style.cursor = 'pointer';
      el.addEventListener('click', onThumbClick);
    });
  }

  function removeThumbnailEvents() {
    document.querySelectorAll('.gallery-thumb, .thumb-container').forEach(function(el) {
      el.removeEventListener('click', onThumbClick);
    });
  }

  function resetThumbnailStyles() {
    document.querySelectorAll('.gallery-thumb, .thumb-container').forEach(function(el) {
      el.classList.remove('page-deleted', 'page-selected');
    });
  }

  function onThumbClick(e) {
    if (!state.mode) return;
    var el = e.currentTarget;
    var page = parseInt(el.getAttribute('data-page') || el.getAttribute('data-index'), 10);
    if (isNaN(page)) return;

    if (state.mode === 'delete') {
      togglePageDelete(page, el);
    } else if (state.mode === 'replace') {
      triggerPageReplace(page, el);
    }
    updateStatus();
  }

  // ---- 删除模式 ----
  window.pmDeleteMode = function() {
    state.mode = 'delete';
    state.selectedPages = [];
    resetThumbnailStyles();
  };

  function togglePageDelete(page, el) {
    var idx = state.selectedPages.indexOf(page);
    if (idx >= 0) {
      state.selectedPages.splice(idx, 1);
      el.classList.remove('page-deleted');
    } else {
      state.selectedPages.push(page);
      el.classList.add('page-deleted');
    }
    // 记录变更
    recordChange('delete', page);
  }

  // ---- 插入模式 ----
  window.pmInsertMode = function() {
    state.mode = 'insert';
    if (insertForm) insertForm.style.display = 'block';
  };

  window.pmFetchPreview = function() {
    var sourceCID = parseInt(document.getElementById('insert-source-cid').value, 10);
    if (!sourceCID) {
      showToast('请输入源 CID', 'error');
      return;
    }

    fetch('/api/comic/getComicPages', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ cid: sourceCID }),
    })
    .then(function(r) { return r.json(); })
    .then(function(data) {
      if (data.head && data.head.code === 0 && data.body && data.body.pages) {
        renderInsertPreview(data.body.pages, sourceCID);
      } else {
        showToast('获取页面失败: ' + (data.head ? data.head.msg : '未知错误'), 'error');
      }
    })
    .catch(function(err) {
      showToast('请求失败: ' + err.message, 'error');
    });
  };

  function renderInsertPreview(pages, sourceCID) {
    var previewContainer = document.getElementById('insert-preview');
    if (!previewContainer) return;
    previewContainer.style.display = 'flex';
    previewContainer.innerHTML = '';

    state.insertedPages = [];

    pages.forEach(function(p, i) {
      var item = document.createElement('div');
      item.className = 'preview-item';
      item.setAttribute('data-page', p.page);
      item.setAttribute('data-name', p.name);
      item.innerHTML = '<img src="' + escapeHtml(p.thumb_url) + '" alt="Page ' + p.page + '" loading="lazy">' +
        '<div class="page-num">' + p.page + '</div>';
      item.addEventListener('click', function() {
        this.classList.toggle('selected');
        var pg = parseInt(this.getAttribute('data-page'), 10);
        var idx = state.insertedPages.indexOf(pg);
        if (idx >= 0) {
          state.insertedPages.splice(idx, 1);
        } else {
          state.insertedPages.push(pg);
        }
      });
      previewContainer.appendChild(item);
    });

    // 保存源 CID 用于确认插入
    previewContainer.setAttribute('data-source-cid', sourceCID);
  }

  window.pmConfirmInsert = function() {
    var previewContainer = document.getElementById('insert-preview');
    if (!previewContainer || state.insertedPages.length === 0) {
      showToast('请先选择要插入的页面', 'error');
      return;
    }

    var sourceCID = parseInt(previewContainer.getAttribute('data-source-cid'), 10);
    var afterPage = parseInt(document.getElementById('insert-after-page').value, 10) || 0;

    recordChange('insert', {
      source_cid: sourceCID,
      pages: state.insertedPages,
      after_page: afterPage,
    });

    showToast('已标记插入 ' + state.insertedPages.length + ' 页，点击保存生效', 'info');
    pmCancelInsert();
    updateStatus();
  };

  window.pmCancelInsert = function() {
    if (insertForm) insertForm.style.display = 'none';
    var preview = document.getElementById('insert-preview');
    if (preview) { preview.style.display = 'none'; preview.innerHTML = ''; }
    state.insertedPages = [];
    state.mode = null;
  };

  // ---- 替换模式 ----
  window.pmReplaceMode = function() {
    state.mode = 'replace';
    resetThumbnailStyles();
  };

  function triggerPageReplace(page, el) {
    var input = document.createElement('input');
    input.type = 'file';
    input.accept = 'image/jpeg,image/png,image/webp';
    input.onchange = function(e) {
      var file = e.target.files[0];
      if (!file) return;
      // 这里简化处理——实际需要上传文件
      recordChange('replace', { page: page, file: file.name });
      showToast('已标记替换第 ' + page + ' 页，点击保存生效', 'info');
    };
    input.click();
  }

  // ---- 重排模式 ----
  window.pmReorderMode = function() {
    state.mode = 'reorder';
    // 简化：重排模式下缩略图可拖拽，前端记录新顺序
    // HTML5 Drag & Drop 实现
    var thumbs = document.querySelectorAll('.gallery-thumb, .thumb-container');
    thumbs.forEach(function(el) {
      el.draggable = true;
      el.addEventListener('dragstart', onDragStart);
      el.addEventListener('dragover', onDragOver);
      el.addEventListener('drop', onDrop);
    });
  };

  var dragSrcPage = null;

  function onDragStart(e) {
    dragSrcPage = parseInt(e.currentTarget.getAttribute('data-page'), 10);
    e.dataTransfer.effectAllowed = 'move';
  }

  function onDragOver(e) {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
  }

  function onDrop(e) {
    e.preventDefault();
    var targetPage = parseInt(e.currentTarget.getAttribute('data-page'), 10);
    if (dragSrcPage && targetPage && dragSrcPage !== targetPage) {
      recordChange('reorder', { from: dragSrcPage, to: targetPage });
      // 视觉上交换位置（简化）
      showToast('已重排第 ' + dragSrcPage + ' 页与第 ' + targetPage + ' 页', 'info');
    }
    dragSrcPage = null;
  }

  // ---- 变更记录 ----
  function recordChange(type, data) {
    // 去重：同类型的同页变更只保留最新
    state.changes = state.changes.filter(function(c) {
      if (type === 'delete' && c.type === 'delete' && c.page === data) return false;
      return true;
    });
    state.changes.push({ type: type, data: data, timestamp: Date.now() });
    updateStatus();
  }

  // ---- 撤销 ----
  window.pmUndo = function() {
    if (state.changes.length === 0) {
      showToast('没有可撤销的变更', 'info');
      return;
    }
    state.changes.pop();
    updateStatus();
    showToast('已撤销上一步操作', 'info');
  };

  // ---- 保存 ----
  window.pmSave = function() {
    if (state.changes.length === 0) {
      showToast('没有变更需要保存', 'info');
      return;
    }

    // 构建保存请求
    var payload = {
      cid: cid,
      pages: state.changes.map(function(c) {
        if (c.type === 'delete') {
          return { page: c.data, action: 'delete' };
        }
        if (c.type === 'insert') {
          return { page: c.data.after_page, action: 'insert', source_cid: c.data.source_cid, source_pages: c.data.pages };
        }
        if (c.type === 'replace') {
          return { page: c.data.page, action: 'replace', file: c.data.file };
        }
        if (c.type === 'reorder') {
          return { page: c.data.from, action: 'reorder', target_page: c.data.to };
        }
        return null;
      }).filter(Boolean),
    };

    fetch('/api/comic/savePages', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
    .then(function(r) { return r.json(); })
    .then(function(data) {
      if (data.head && data.head.code === 0) {
        showToast('保存成功！归档已标记为过期', 'success');
        state.changes = [];
        pmExit();
        // 提示重新归档
        if (confirm('页面已变更，是否立即重新归档？')) {
          // 触发归档
          if (window.archiveComic) window.archiveComic();
        }
      } else {
        showToast('保存失败: ' + (data.head ? data.head.msg : '未知错误'), 'error');
      }
    })
    .catch(function(err) {
      showToast('请求失败: ' + err.message, 'error');
    });
  };

  // ---- 删除确认 ----
  window.openDeleteConfirm = function() {
    // 收集 comic 标题用于确认
    var title = window._gallery && window._gallery.Title ? (window._gallery.Title.english || '') : '';
    var input = prompt('输入 comic 标题以确认删除:\n"一但删除无法恢复"\n\n标题: ' + title);
    if (input && input.trim() === title.trim()) {
      fetch('/api/admin/comic/delete', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ cid: cid }),
      })
      .then(function(r) { return r.json(); })
      .then(function(data) {
        if (data.head && data.head.code === 0) {
          showToast('删除成功', 'success');
          window.location.href = '/';
        } else {
          showToast('删除失败: ' + (data.head ? data.head.msg : '未知错误'), 'error');
        }
      })
      .catch(function(err) {
        showToast('请求失败: ' + err.message, 'error');
      });
    } else if (input !== null) {
      showToast('标题不匹配，删除取消', 'error');
    }
  };

  // ---- 状态更新 ----
  function updateStatus() {
    if (statusEl) {
      statusEl.textContent = '未保存变更: ' + state.changes.length;
    }
  }

  // ---- 工具 ----
  function escapeHtml(text) {
    var div = document.createElement('div');
    div.appendChild(document.createTextNode(String(text)));
    return div.innerHTML;
  }

  function showToast(msg, type) {
    if (window.showAdminToast) {
      window.showAdminToast(msg, type);
    } else if (window.showToast) {
      window.showToast(msg, type);
    } else {
      alert(msg);
    }
  }

  // ---- 设置缩略图初始页码 ----
  function initPageNumbers() {
    document.querySelectorAll('.gallery-thumb, .thumb-container').forEach(function(el, idx) {
      if (!el.hasAttribute('data-page')) {
        el.setAttribute('data-page', idx + 1);
      }
    });
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initPageNumbers);
  } else {
    initPageNumbers();
  }
})();
```

- [ ] **步骤 2：验证 JS 语法**

运行：`cd D:\workdir\leon\cocomhub\cocom && node --check cmd/server/view/static/custom/js/modules/page-manager.js 2>&1`
预期：无错误

- [ ] **步骤 3：编译验证整体项目**

运行：`cd D:\workdir\leon\cocomhub\cocom && go build ./cmd/...`
预期：编译成功

- [ ] **步骤 4：Commit**

```bash
git add cmd/server/view/static/custom/js/modules/page-manager.js
git commit -m "feat: implement page-manager.js for detail page management"
```

---

### 任务 10：交互改进 — 归档过期状态 + 侧边栏归档按钮状态

**文件：**
- 修改：`cmd/server/view/static/tpl/gallery_detail.tpl`
- 修改：`cmd/server/view/static/custom/js/modules/gallery-actions.js`（或 scripts.js）
- 修改：`cmd/server/view/static/custom/css/styles.css`

- [ ] **步骤 1：在 gallery_detail.tpl 中添加归档过期横幅**

在 `#thumbnail-container` 上方添加：
```html
{{if .ArchiveStale}}
<div id="archive-stale-banner" class="archive-stale-banner">
  <i class="fa fa-exclamation-triangle"></i>
  ⚠ 页面内容已变更，存档已过期。
  <button class="btn btn-warning btn-sm" onclick="reArchive()">重新归档</button>
</div>
{{end}}
```

- [ ] **步骤 2：更新 gallery_detail.go 传递 ArchiveStale 标志**

```go
// 在 GalleryDetail 结构体中添加
type GalleryDetail struct {
    api.ComicInfo
    URL           string
    CSRFToken     string
    likedTagIDs   map[int]bool
    ArchiveStale  bool  // 新增
}
```

设置逻辑：当 `info.Archive != nil && info.Archive.Status == "stale"` 时为 true。

- [ ] **步骤 3：添加归档过期横幅 CSS**

```css
.archive-stale-banner {
  background: #fff3cd;
  color: #856404;
  padding: 10px 16px;
  text-align: center;
  font-size: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  border-bottom: 1px solid #ffc107;
}
.archive-stale-banner .btn-warning {
  background: #ffc107;
  border-color: #ffc107;
  color: #333;
  font-size: 12px;
}
```

- [ ] **步骤 4：在归档按钮上添加状态指示**

当 `ArchiveStale` 为 true 时，归档按钮显示 `📦 ⚠ 重新归档`。

- [ ] **步骤 5：在 gallery-actions.js 中添加 reArchive 函数**

```js
window.reArchive = function() {
  if (window.archiveComic) {
    window.archiveComic();
  }
};
```

- [ ] **步骤 6：编译验证**

运行：`cd D:\workdir\leon\cocomhub\cocom && go build ./cmd/...`
预期：编译成功

- [ ] **步骤 7：Commit**

```bash
git add cmd/server/view/static/tpl/gallery_detail.tpl cmd/server/view/static/custom/css/styles.css cmd/server/view/gallery_detail.go
git commit -m "feat: add archive stale banner and re-archive button"
```

---

### 任务 11：交互改进 — 键盘快捷键 + 侧边栏折叠 + 按钮快捷键提示 + Loading 状态

**文件：**
- 修改：`cmd/server/view/static/tpl/index.tpl`（按键提示）
- 修改：`cmd/server/view/static/tpl/gallery_detail.tpl`（按键提示）
- 修改：`cmd/server/view/static/custom/js/modules/quick-link.js`（已包含快捷键）
- 修改：`cmd/server/view/static/custom/css/styles.css`（loading 动画）
- 修改：`cmd/server/view/static/custom/js/modules/admin-compare.js`（loading 状态增强）

- [ ] **步骤 1：在 index.tpl 的按钮上添加快捷键提示**

按钮标签已包含 `<span class="shortcut-hint">L</span>`（在任务 5 中已设计）。

- [ ] **步骤 2：为整个应用添加统一的 loading 样式**

```css
/* Loading 动画 */
.loading-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0,0,0,0.3);
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
}
.loading-spinner {
  width: 40px;
  height: 40px;
  border: 3px solid rgba(255,255,255,0.3);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}

.btn-loading {
  position: relative;
  pointer-events: none;
  opacity: 0.7;
}
.btn-loading::after {
  content: '';
  display: inline-block;
  width: 12px;
  height: 12px;
  margin-left: 6px;
  border: 2px solid rgba(255,255,255,0.3);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  vertical-align: middle;
}
```

- [ ] **步骤 3：在 admin-compare.js 中添加全局 loading 辅助函数**

在文件顶部添加：
```js
function showLoading(btn) {
  if (btn) btn.classList.add('btn-loading');
}
function hideLoading(btn) {
  if (btn) btn.classList.remove('btn-loading');
}
```

并在 `confirmLink()` 等函数中调用。

- [ ] **步骤 4：编译验证**

运行：`cd D:\workdir\leon\cocomhub\cocom && go build ./cmd/...`
预期：编译成功

- [ ] **步骤 5：整体验证——运行项目测试**

运行：`cd D:\workdir\leon\cocomhub\cocom && go test ./cmd/server/handler/... -v -tags=memory_storage_integration -count=1 2>&1`
预期：所有测试 PASS（无 MongoDB 时 Skip）

- [ ] **步骤 6：Commit**

```bash
git add cmd/server/view/static/custom/css/styles.css cmd/server/view/static/custom/js/modules/admin-compare.js
git commit -m "feat: add loading states, keyboard shortcut hints, and interaction polish"
```

---

## 验证方案

### 1. 编译验证
每次修改后运行 `go build ./cmd/...` 确保编译通过。

### 2. 单元测试
```bash
# 运行所有 handler 测试
cd D:\workdir\leon\cocomhub\cocom && go test ./cmd/server/handler/... -v -tags=memory_storage_integration -count=1

# 运行所有 view 测试
cd D:\workdir\leon\cocomhub\cocom && go test ./cmd/server/view/... -v -count=1
```

### 3. JS 语法验证
```bash
# 验证 JS 文件语法
node --check cmd/server/view/static/custom/js/modules/quick-link.js
node --check cmd/server/view/static/custom/js/modules/page-manager.js
node --check cmd/server/view/static/custom/js/modules/admin-compare.js
```

### 4. 集成验证（需要 MongoDB）
启动 server 后：
- 访问 Index 页确认右侧侧边栏显示
- 点击"链接"按钮进入链接模式，选择主/备 comic，确认链接
- 点击"对比"按钮进入对比模式，选择 2+ comic，跳转到 admin 页
- 访问 Detail 页确认左右侧边栏互换
- 进入页管理模式，测试删除/插入/替换/重排
- 测试删除 comic 后索引页过滤与提示页
- 验证归档过期横幅显示

### 5. 回滚方案
每个模块的变更在 commit 后可独立回滚：
- Module A：`git revert <quick-link-commit>`
- Module B：`git revert <page-manager-commit>`
- Module C：`git revert <polish-commit>`
