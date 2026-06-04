# M3：图片浏览与性能优化 — 实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 实现运行时图片缩放（响应式图片服务）+ 浏览器缓存策略

**架构：** 在 `Picture` handler 中增加 `?w=` 参数解析，使用 `github.com/disintegration/imaging` 库运行时缩放图片并直接编码到 ResponseWriter；对不可缩放格式（如 WebP）降级返回原始图片。同时设置 Cache-Control / ETag / Last-Modified 响应头。

**技术栈：** Go 1.26 + Gin + `github.com/disintegration/imaging`

---

## 涉及文件

| 文件 | 操作 | 职责 |
|------|------|------|
| `cmd/server/view/picture.go` | 修改 | `Picture` handler 增加 `w` 参数处理和缓存头 |

---

### 任务 1：响应式图片服务 + 缓存策略

**文件：**
- 修改：`cmd/server/view/picture.go`

- [ ] **步骤 1：读取当前文件并确认代码结构**

当前 `cmd/server/view/picture.go`：
```go
package view

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/pkg/errwrap"

	"github.com/gin-gonic/gin"
)

func parsePictureArgs(c *gin.Context) (cid int, name string, err error) {
	cid, err = strconv.Atoi(c.Param("cid"))
	if err != nil {
		err = errwrap.ErrInvalidArgs.SetIErrF("request parse cid failed: %s", err)
		return
	}
	name = c.Param("name")
	if len(name) == 0 {
		err = errwrap.ErrInvalidArgs.SetIErrF("picture name not found")
		return
	}
	return
}

func Picture(c *gin.Context) {
	cid, name, err := parsePictureArgs(c)
	if err != nil {
		slog.ErrorContext(c, "parsePictureArgs failed",
			slog.String("errmsg", err.Error()))
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	info := api.ComicInfo{}
	err = comic.GetComicInfo(c, cid, &info)
	if err != nil {
		slog.ErrorContext(c, "comic.GetComicInfo failed",
			slog.String("errmsg", err.Error()))
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	c.File(info.PageSavePathByName(name))
}
```

- [ ] **步骤 2：实现完整的 `Picture` handler**

将 `cmd/server/view/picture.go` 整体替换为：

```go
// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package view

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/pkg/errwrap"
	"github.com/disintegration/imaging"

	"github.com/gin-gonic/gin"
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
)

func parsePictureArgs(c *gin.Context) (cid int, name string, err error) {
	cid, err = strconv.Atoi(c.Param("cid"))
	if err != nil {
		err = errwrap.ErrInvalidArgs.SetIErrF("request parse cid failed: %s", err)
		return
	}

	name = c.Param("name")
	if len(name) == 0 {
		err = errwrap.ErrInvalidArgs.SetIErrF("picture name not found")
		return
	}

	return
}

// setCacheHeaders 设置图片缓存响应头
func setCacheHeaders(c *gin.Context, filePath string, extra string) {
	info, err := os.Stat(filePath)
	if err != nil {
		return
	}

	modTime := info.ModTime().Unix()
	name := info.Name()

	// Cache-Control: 1 year (immutable content — images don't change after upload)
	c.Header("Cache-Control", "public, max-age=31536000")

	// ETag: based on filename, modification time, and optional extra (e.g., resize params)
	etag := fmt.Sprintf(`"%s-%d"`, name, modTime)
	if extra != "" {
		etag = fmt.Sprintf(`"%s-%s-%d"`, name, extra, modTime)
	}
	c.Header("ETag", etag)

	// Last-Modified
	c.Header("Last-Modified", info.ModTime().Format(time.RFC1123))
}

// resizeImage 运行时缩放图片并写入响应
// 返回 true 表示已处理（写入响应），false 表示需要降级到原始文件
func resizeImage(c *gin.Context, filePath string, width int) bool {
	// 检查文件扩展名，WebP 等格式可能无法解码
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == ".webp" {
		slog.WarnContext(c, "cannot resize webp image, serving original",
			slog.String("path", filePath))
		return false
	}

	// 使用 imaging 库打开图片
	srcImg, err := imaging.Open(filePath)
	if err != nil {
		slog.WarnContext(c, "imaging.Open failed, serving original",
			slog.String("path", filePath),
			slog.String("errmsg", err.Error()))
		return false
	}

	// 缩放到指定宽度（保持宽高比），高度传 0 自动按比例
	dstImg := imaging.Resize(srcImg, width, 0, imaging.Lanczos)

	// 根据原始格式编码输出
	c.Header("Content-Type", c.GetHeader("Content-Type"))
	switch ext {
	case ".jpg", ".jpeg":
		c.Header("Content-Type", "image/jpeg")
		if err := jpeg.Encode(c.Writer, dstImg, &jpeg.Options{Quality: 85}); err != nil {
			slog.ErrorContext(c, "jpeg encode failed", slog.String("errmsg", err.Error()))
			return false
		}
	case ".png":
		c.Header("Content-Type", "image/png")
		if err := png.Encode(c.Writer, dstImg); err != nil {
			slog.ErrorContext(c, "png encode failed", slog.String("errmsg", err.Error()))
			return false
		}
	case ".gif":
		c.Header("Content-Type", "image/gif")
		if err := imaging.Encode(c.Writer, dstImg, imaging.GIF); err != nil {
			slog.ErrorContext(c, "gif encode failed", slog.String("errmsg", err.Error()))
			return false
		}
	default:
		slog.WarnContext(c, "unsupported format for resize, serving original",
			slog.String("ext", ext))
		return false
	}

	return true
}
```

需要在 `picture.go` 顶部加 import：
```go
import (
    "path/filepath"
)
```

注意：`golang.org/x/image/bmp` 和 `golang.org/x/image/tiff` 可能不需要 import（如果 Go 标准库已注册 BMP/TIFF decoder）。仅导入实际用到的包即可。

**核心逻辑：**

```go
func Picture(c *gin.Context) {
	cid, name, err := parsePictureArgs(c)
	if err != nil {
		slog.ErrorContext(c, "parsePictureArgs failed",
			slog.String("errmsg", err.Error()))
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	info := api.ComicInfo{}
	err = comic.GetComicInfo(c, cid, &info)
	if err != nil {
		slog.ErrorContext(c, "comic.GetComicInfo failed",
			slog.String("errmsg", err.Error()))
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	filePath := info.PageSavePathByName(name)

	// 设置缓存头
	widthParam := c.Query("w")
	var extra string
	var resizeWidth int
	if widthParam != "" {
		if w, err := strconv.Atoi(widthParam); err == nil && w >= 1 && w <= 4096 {
			resizeWidth = w
			extra = "w" + widthParam
		}
	}
	setCacheHeaders(c, filePath, extra)

	// 处理缩放请求
	if resizeWidth > 0 {
		if resizeImage(c, filePath, resizeWidth) {
			return // 缩放成功，已写入响应
		}
		// 缩放失败（降级），继续执行 c.File()
	}

	c.File(filePath)
}
```

- [ ] **步骤 3：编译验证**

运行：
```bash
cd D:\workdir\leon\cocomhub\cocom
go build ./...
```

预期：编译成功，无错误

如果出现 `undefined: filepath`，添加 `"path/filepath"` import。

- [ ] **步骤 4：Commit**

```bash
git add cmd/server/view/picture.go
git commit -m "feat(server): 响应式图片服务 ?w= 运行时缩放 + 浏览器缓存策略"
```

---

## 验收标准

- [ ] `GET /galleries/{cid}/1.jpg?w=200` 返回宽度为 200px 的缩放图片
- [ ] `GET /galleries/{cid}/1.jpg` 返回原始图片（未缩放）
- [ ] 图片响应包含 `Cache-Control: public, max-age=31536000` 头
- [ ] 图片响应包含 `ETag` 和 `Last-Modified` 头
- [ ] WebP 等不可缩放格式返回原始图片并打印 warn 日志
- [ ] `go build ./...` 编译通过