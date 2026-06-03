// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package view

import (
	"fmt"
	"image/jpeg"
	"image/png"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/pkg/errwrap"
	"github.com/disintegration/imaging"

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

// setCacheHeaders 设置图片缓存响应头
func setCacheHeaders(c *gin.Context, filePath string, extra string) {
	info, err := os.Stat(filePath)
	if err != nil {
		return
	}

	modTime := info.ModTime().Unix()
	name := info.Name()

	// Cache-Control: 1 year (immutable content after upload)
	c.Header("Cache-Control", "public, max-age=31536000")

	// ETag: based on filename + mtime (+ resize param if present)
	etag := fmt.Sprintf(`"%s-%d"`, name, modTime)
	if extra != "" {
		etag = fmt.Sprintf(`"%s-%s-%d"`, name, extra, modTime)
	}
	c.Header("ETag", etag)

	// Last-Modified
	c.Header("Last-Modified", info.ModTime().Format(time.RFC1123))
}

// resizeImage 运行时缩放图片并写入响应；返回 true 表示已处理
func resizeImage(c *gin.Context, filePath string, width int) bool {
	ext := strings.ToLower(filepath.Ext(filePath))

	// WebP 无法通过 imaging 解码，降级
	if ext == ".webp" {
		slog.WarnContext(c, "cannot resize webp image, serving original",
			slog.String("path", filePath))
		return false
	}

	srcImg, err := imaging.Open(filePath)
	if err != nil {
		slog.WarnContext(c, "imaging.Open failed, serving original",
			slog.String("path", filePath),
			slog.String("errmsg", err.Error()))
		return false
	}

	// 缩放到指定宽度（高度 0 = 自动按宽高比）
	dstImg := imaging.Resize(srcImg, width, 0, imaging.Lanczos)

	// 按原始格式编码输出（仅支持 JPEG / PNG，其余格式降级）
	switch ext {
	case ".jpg", ".jpeg":
		c.Header("Content-Type", "image/jpeg")
		if err := jpeg.Encode(c.Writer, dstImg, &jpeg.Options{Quality: 85}); err != nil {
			slog.ErrorContext(c, "jpeg encode failed", slog.String("errmsg", err.Error()))
			return false
		}
		return true
	case ".png":
		c.Header("Content-Type", "image/png")
		if err := png.Encode(c.Writer, dstImg); err != nil {
			slog.ErrorContext(c, "png encode failed", slog.String("errmsg", err.Error()))
			return false
		}
		return true
	default:
		slog.WarnContext(c, "unsupported format for resize encoding, serving original",
			slog.String("ext", ext))
		return false
	}
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

	filePath := info.PageSavePathByName(name)

	// 解析 ?w= 缩放参数
	widthParam := c.Query("w")
	var resizeWidth int
	var extra string
	if widthParam != "" {
		if w, err := strconv.Atoi(widthParam); err == nil && w >= 1 && w <= 4096 {
			resizeWidth = w
			extra = "w" + widthParam
		}
	}

	// 设置缓存头（先于响应体写入）
	setCacheHeaders(c, filePath, extra)

	// 有缩放参数时尝试运行时缩放
	if resizeWidth > 0 {
		if resizeImage(c, filePath, resizeWidth) {
			return // 缩放并写入响应成功
		}
		// 缩放失败（格式不支持 / 解码失败），降级到原始文件
	}

	c.File(filePath)
}