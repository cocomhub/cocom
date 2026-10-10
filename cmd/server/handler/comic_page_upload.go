// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"fmt"
	"image"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	// 注册各格式的解码器，使 image.DecodeConfig 能校验上传内容确实是图片。
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/pkg/httpwrap"
	_ "github.com/cocomhub/cocom/pkg/imaging/webp"
)

// maxPageUploadBytes 单个页面文件的上传上限。
// 漫画页通常几 MB，32MiB 足够，同时避免磁盘/内存被单请求放大。
const maxPageUploadBytes = 32 << 20

// maxPageUploadMemory 留给 multipart 解析的内存上限，超出部分落临时文件。
const maxPageUploadMemory = 4 << 20

// UploadPage 接收一张页面图片，落为漫画目录下的暂存文件并返回其文件名。
// 该文件名随后由 POST /api/comic/savePages 的 replace/insert 的 file 字段引用。
//
// POST /api/comic/uploadPage   multipart/form-data: cid, file
//
// 安全约束：
//   - 文件名由服务端生成（客户端不参与），固定加 pageUploadPrefix 前缀并放在漫画目录内，
//     因此 savePages 只可能引用本接口产生的文件；
//   - 后缀走白名单，且必须通过 image.DecodeConfig 的内容校验（防止上传任意文件）；
//   - 请求体有上限，避免无界落盘。
func UploadPage(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	req.Body = http.MaxBytesReader(w, req.Body, maxPageUploadBytes)
	if err := req.ParseMultipartForm(maxPageUploadMemory); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "invalid multipart form: "+err.Error())
		return
	}
	// 请求体可能很大，确保临时文件被清理。
	defer func() {
		if req.MultipartForm != nil {
			_ = req.MultipartForm.RemoveAll()
		}
	}()

	cid, err := strconv.Atoi(strings.TrimSpace(req.FormValue("cid")))
	if err != nil || cid <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "cid is required")
		return
	}

	file, header, err := req.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "file is required")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !isPageImageExt(ext) {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("unsupported image extension %q (allowed: %s)",
			ext, strings.Join(pageImageExts, ", ")))
		return
	}

	var info api.ComicInfo
	if getErr := comic.GetComicInfo(ctx, cid, &info); getErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("get comic info failed: %s", getErr))
		return
	}
	saveDir := info.SaveDir()
	if mkErr := os.MkdirAll(saveDir, 0o755); mkErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("create save dir failed: %s", mkErr))
		return
	}

	// 先写临时文件、校验通过后再改名成正式暂存名，避免留下半截文件被 savePages 引用。
	name := fmt.Sprintf("%s%d%s", pageUploadPrefix, time.Now().UnixNano(), ext)
	dst := filepath.Join(saveDir, name)

	tmp, err := os.CreateTemp(saveDir, pageUploadPrefix+"*.part")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("create temp file failed: %s", err))
		return
	}
	tmpPath := tmp.Name()
	cleanupTmp := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}

	n, err := io.Copy(tmp, file)
	if err != nil {
		cleanupTmp()
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "write upload failed: "+err.Error())
		return
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("close temp file failed: %s", err))
		return
	}

	if err := validatePageImageFile(tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, err.Error())
		return
	}

	if err := os.Rename(tmpPath, dst); err != nil {
		_ = os.Remove(tmpPath)
		w.WriteHeader(http.StatusInternalServerError)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("commit upload failed: %s", err))
		return
	}

	httpwrap.ResponseSucc(ctx, w, map[string]any{
		"name": name,
		"size": n,
	})
}

// validatePageImageFile 打开文件并解码头，确认它是可识别的图片。
// 解不出图直接拒收：否则会变成「保存成功但画廊打不开」的静默错误。
func validatePageImageFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open upload failed: %w", err)
	}
	defer f.Close()

	cfg, format, err := image.DecodeConfig(f)
	if err != nil {
		return fmt.Errorf("上传内容不是可识别的图片: %w", err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return fmt.Errorf("上传图片尺寸非法: %dx%d", cfg.Width, cfg.Height)
	}
	if cfg.Width > maxUploadedPagePixels || cfg.Height > maxUploadedPagePixels {
		return fmt.Errorf("上传图片尺寸超出上限 %d: %dx%d", maxUploadedPagePixels, cfg.Width, cfg.Height)
	}
	if format == "" {
		return fmt.Errorf("无法识别上传图片格式")
	}
	return nil
}

// maxUploadedPagePixels 单边像素上限（约 1 亿像素的放大攻击防线）。
const maxUploadedPagePixels = 100000
