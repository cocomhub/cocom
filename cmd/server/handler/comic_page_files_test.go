// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/cocomhub/cocom/cmd/server/api"
	internalComic "github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/pkg/httpwrap"
)

// pageContent 返回第 orig 页的可区分内容，用于验证「文件是否跟着页号一起搬」。
func pageContent(orig int) string { return fmt.Sprintf("PAGE-%d", orig) }

func writePageFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// pageFiles 返回目录下所有页面图片文件（名字排序），名字 → 内容。
func pageFiles(t *testing.T, dir string) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}
		}
		t.Fatalf("read dir %s: %v", dir, err)
	}
	out := make(map[string]string)
	for _, de := range entries {
		if de.IsDir() || !isPageImageExt(filepath.Ext(de.Name())) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, de.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", de.Name(), err)
		}
		out[de.Name()] = string(b)
	}
	return out
}

func sortedNames(m map[string]string) []string {
	names := make([]string, 0, len(m))
	for k := range m {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// pngBytes 生成一张最小的合法 PNG（上传接口会做内容校验，不能用假数据）。
func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

// uploadTestPage 调上传接口把一张真图落为暂存文件，返回其文件名。
func uploadTestPage(t *testing.T, cid int, name string) string {
	t.Helper()
	return uploadTestPageBytes(t, cid, name, pngBytes(t))
}

func uploadTestPageBytes(t *testing.T, cid int, name string, content []byte) string {
	t.Helper()

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if err := mw.WriteField("cid", strconv.Itoa(cid)); err != nil {
		t.Fatalf("write cid field: %v", err)
	}
	fw, err := mw.CreateFormFile("file", name)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fw.Write(content); err != nil {
		t.Fatalf("write file content: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/comic/uploadPage", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	UploadPage(w, req)

	var resp struct {
		Head struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		} `json:"head"`
		Body struct {
			Name string `json:"name"`
		} `json:"body"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode upload response: %v (status %d)", err, w.Code)
	}
	if resp.Head.Code != 0 {
		t.Fatalf("upload failed: code=%d msg=%s status=%d", resp.Head.Code, resp.Head.Msg, w.Code)
	}
	if resp.Body.Name == "" {
		t.Fatal("upload 返回空文件名")
	}
	return resp.Body.Name
}

func savePagesExpectOK(t *testing.T, cid int, items ...map[string]any) {
	t.Helper()
	w := httptest.NewRecorder()
	SavePages(w, savePagesRequestFor(t, cid, items...))
	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode savePages response: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Fatalf("savePages code = %d (%s)", resp.Head.Code, resp.Head.Msg)
	}
}

// TestSavePages_DeleteRenumbersFiles 锁定核心不变量：删页后磁盘文件名必须跟着重编号。
// 只改元数据会让第 2 页显示被删页的图片（文件名由下标派生，且只被读取方使用）。
func TestSavePages_DeleteRenumbersFiles(t *testing.T) {
	const cid = 7101
	seedPagesComic(t, cid, 5)

	savePagesExpectOK(t, cid, map[string]any{"page": 2, "action": "delete"})

	dir := (&api.ComicInfo{CID: cid}).SaveDir()
	files := pageFiles(t, dir)
	if got, want := sortedNames(files), []string{"1.jpg", "2.jpg", "3.jpg", "4.jpg"}; len(got) != len(want) {
		t.Fatalf("文件集合 = %v, want %v", got, want)
	}
	// 原第 3 页的内容现在应位于 2.jpg。
	if files["2.jpg"] != pageContent(3) {
		t.Errorf("2.jpg 内容 = %q, want %q", files["2.jpg"], pageContent(3))
	}
	if files["4.jpg"] != pageContent(5) {
		t.Errorf("4.jpg 内容 = %q, want %q", files["4.jpg"], pageContent(5))
	}
}

func TestSavePages_MultipleDeletesRenumberFiles(t *testing.T) {
	const cid = 7102
	seedPagesComic(t, cid, 5)

	savePagesExpectOK(t, cid,
		map[string]any{"page": 2, "action": "delete"},
		map[string]any{"page": 3, "action": "delete"},
	)

	if got, want := savedPageNumbers(t, cid), []int{1, 4, 5}; !slices.Equal(got, want) {
		t.Errorf("元数据 = %v, want %v", got, want)
	}
	files := pageFiles(t, (&api.ComicInfo{CID: cid}).SaveDir())
	for i, want := range []string{pageContent(1), pageContent(4), pageContent(5)} {
		name := fmt.Sprintf("%d.jpg", i+1)
		if files[name] != want {
			t.Errorf("%s 内容 = %q, want %q", name, files[name], want)
		}
	}
	if len(files) != 3 {
		t.Errorf("文件数 = %d, want 3 (%v)", len(files), sortedNames(files))
	}
}

func TestSavePages_DeleteAllRemovesFiles(t *testing.T) {
	const cid = 7103
	seedPagesComic(t, cid, 3)

	savePagesExpectOK(t, cid,
		map[string]any{"page": 1, "action": "delete"},
		map[string]any{"page": 2, "action": "delete"},
		map[string]any{"page": 3, "action": "delete"},
	)

	if files := pageFiles(t, (&api.ComicInfo{CID: cid}).SaveDir()); len(files) != 0 {
		t.Errorf("全删后仍有文件: %v", sortedNames(files))
	}
}

// TestSavePages_SwapRenumbersFiles 验证 reorder 是「交换」并且磁盘文件同步交换。
func TestSavePages_SwapRenumbersFiles(t *testing.T) {
	const cid = 7104
	seedPagesComic(t, cid, 5)

	savePagesExpectOK(t, cid, map[string]any{"page": 2, "action": "reorder", "to_page": 4})

	files := pageFiles(t, (&api.ComicInfo{CID: cid}).SaveDir())
	want := map[string]string{
		"1.jpg": pageContent(1),
		"2.jpg": pageContent(4),
		"3.jpg": pageContent(3),
		"4.jpg": pageContent(2),
		"5.jpg": pageContent(5),
	}
	for name, content := range want {
		if files[name] != content {
			t.Errorf("%s 内容 = %q, want %q", name, files[name], content)
		}
	}
}

// TestSavePages_InsertFromSourceComicCopiesFiles 覆盖跨漫画插入：
// 目标漫画得到新页，源漫画的文件必须**保留**（复制而非搬走）。
func TestSavePages_InsertFromSourceComicCopiesFiles(t *testing.T) {
	const srcCID, dstCID = 7105, 7106
	seedPagesComic(t, srcCID, 2)
	seedPagesComic(t, dstCID, 2)

	savePagesExpectOK(t, dstCID, map[string]any{
		"action":       "insert",
		"page":         1, // after_page=1
		"source_cid":   srcCID,
		"source_pages": []int{1, 2},
	})

	if got, want := savedPageNumbers(t, dstCID), []int{1, 0, 0, 2}; !slices.Equal(got, want) {
		t.Errorf("目标页序 = %v, want %v", got, want)
	}
	dstFiles := pageFiles(t, (&api.ComicInfo{CID: dstCID}).SaveDir())
	want := map[string]string{
		"1.jpg": pageContent(1),
		"2.jpg": pageContent(1), // 来自源漫画第 1 页
		"3.jpg": pageContent(2), // 来自源漫画第 2 页
		"4.jpg": pageContent(2),
	}
	for name, content := range want {
		if dstFiles[name] != content {
			t.Errorf("目标 %s 内容 = %q, want %q", name, dstFiles[name], content)
		}
	}
	srcFiles := pageFiles(t, (&api.ComicInfo{CID: srcCID}).SaveDir())
	if len(srcFiles) != 2 || srcFiles["1.jpg"] != pageContent(1) || srcFiles["2.jpg"] != pageContent(2) {
		t.Errorf("源漫画文件被破坏: %v", srcFiles)
	}
}

// TestSavePages_MissingFilesDoNotFail 元数据存在但磁盘缺文件时，
// 保存不应整体失败（保留元数据即可），这是历史数据的容错路径。
func TestSavePages_MissingFilesDoNotFail(t *testing.T) {
	const cid = 7107
	ctx := context.Background()
	info := &api.ComicInfo{CID: cid}
	for i := 1; i <= 3; i++ {
		info.Images.Pages = append(info.Images.Pages, api.PicInfo{T: "j", H: i, Status: true})
	}
	if err := testMemStorage.Save(ctx, internalComic.NewComic(info)); err != nil {
		t.Fatalf("seed: %v", err)
	}
	t.Cleanup(func() { _ = testMemStorage.Delete(ctx, strconv.Itoa(cid)) })
	// 故意不创建任何文件。
	savePagesExpectOK(t, cid, map[string]any{"page": 2, "action": "delete"})
	if got, want := savedPageNumbers(t, cid), []int{1, 3}; !slices.Equal(got, want) {
		t.Errorf("页序 = %v, want %v", got, want)
	}
}

// TestSavePages_ReplaceRejectsNonUploadFile 拒绝引用漫画目录里的任意文件（含路径穿越）。
func TestSavePages_ReplaceRejectsNonUploadFile(t *testing.T) {
	const cid = 7108
	seedPagesComic(t, cid, 3)

	for _, name := range []string{"../outside.png", "/etc/passwd", "1.jpg", ".page-upload-nope.png"} {
		w := httptest.NewRecorder()
		SavePages(w, savePagesRequestFor(t, cid,
			map[string]any{"page": 2, "action": "replace", "file": name}))
		if w.Code != http.StatusBadRequest {
			t.Errorf("file=%q 应返回 400，实际 %d", name, w.Code)
		}
	}
}

// TestSavePages_ReplaceWithoutFileRejected 明确报错而不是静默把类型改成 jpg。
func TestSavePages_ReplaceWithoutFileRejected(t *testing.T) {
	const cid = 7109
	seedPagesComic(t, cid, 3)

	w := httptest.NewRecorder()
	SavePages(w, savePagesRequestFor(t, cid, map[string]any{"page": 2, "action": "replace"}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("缺少 file 的 replace 应返回 400，实际 %d", w.Code)
	}
}

func TestSavePages_InsertRequiresSourceOrUpload(t *testing.T) {
	const cid = 7110
	seedPagesComic(t, cid, 3)

	w := httptest.NewRecorder()
	SavePages(w, savePagesRequestFor(t, cid, map[string]any{"action": "insert", "page": 1}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("insert 无来源时应返回 400，实际 %d", w.Code)
	}
}

func TestUploadPage_RejectsUnsupportedExtension(t *testing.T) {
	const cid = 7111
	seedPagesComic(t, cid, 1)

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("cid", strconv.Itoa(cid))
	fw, _ := mw.CreateFormFile("file", "payload.exe")
	_, _ = fw.Write([]byte("MZ"))
	_ = mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/comic/uploadPage", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	UploadPage(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("非法后缀应返回 400，实际 %d", w.Code)
	}
}

func TestUploadPage_RejectsNonImageContent(t *testing.T) {
	const cid = 7112
	seedPagesComic(t, cid, 1)

	// 后缀合法但内容不是图片 → 必须拒收，否则会「保存成功但画廊打不开」。
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("cid", strconv.Itoa(cid))
	fw, _ := mw.CreateFormFile("file", "fake.png")
	_, _ = fw.Write([]byte("this is definitely not a png"))
	_ = mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/comic/uploadPage", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	UploadPage(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("非图片内容应返回 400，实际 %d", w.Code)
	}
}

func TestUploadPage_StoresStagedFile(t *testing.T) {
	const cid = 7113
	seedPagesComic(t, cid, 1)

	name := uploadTestPage(t, cid, "shot.png")
	if !strings.HasPrefix(name, pageUploadPrefix) {
		t.Errorf("暂存文件名 = %q, 应带前缀 %q", name, pageUploadPrefix)
	}
	path := filepath.Join((&api.ComicInfo{CID: cid}).SaveDir(), name)
	if !fileExists(path) {
		t.Fatalf("暂存文件不存在: %s", path)
	}
}

func TestUploadPage_RejectsMissingCID(t *testing.T) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "a.png")
	_, _ = fw.Write(pngBytes(t))
	_ = mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/comic/uploadPage", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	UploadPage(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("缺少 cid 应返回 400，实际 %d", w.Code)
	}
}

// TestUpdateComicInfo_ReplacesPageSliceWholesale 锁定内存存储的部分更新必须**整体替换**
// pages 切片。
//
// 回归背景：MemoryStorage.Update 曾把合并后的 JSON 直接 json.Unmarshal 进已有对象，
// 而 encoding/json 解码切片时对已有元素是「就地合并」——目标元素上、JSON 里未出现的
// 字段会被保留。于是往 pages 里插入一页时，新页会继承前一页的 H/W（尺寸错乱），
// 这类错误只在内存存储（E2E/单测路径）出现，Mongo 路径不受影响，因此很容易漏掉。
func TestUpdateComicInfo_ReplacesPageSliceWholesale(t *testing.T) {
	const cid = 7120
	ctx := context.Background()

	info := &api.ComicInfo{CID: cid}
	info.Images.Pages = []api.PicInfo{{H: 11, W: 22, T: "j", Status: true}}
	if err := testMemStorage.Save(ctx, internalComic.NewComic(info)); err != nil {
		t.Fatalf("seed: %v", err)
	}
	t.Cleanup(func() { _ = testMemStorage.Delete(ctx, strconv.Itoa(cid)) })

	update := map[string]any{
		"cid": cid,
		"images": map[string]any{
			"pages": []api.PicInfo{
				{H: 11, W: 22, T: "j", Status: true},
				{T: "p", Status: true}, // 新页，不带 H/W
			},
		},
	}
	if err := internalComic.UpdateComicInfo(ctx, cid, update); err != nil {
		t.Fatalf("update: %v", err)
	}

	var got api.ComicInfo
	if err := internalComic.GetComicInfo(ctx, cid, &got); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if len(got.Images.Pages) != 2 {
		t.Fatalf("pages = %d, want 2", len(got.Images.Pages))
	}
	if got.Images.Pages[1].H != 0 || got.Images.Pages[1].W != 0 {
		t.Errorf("新页继承了旧页的字段: %+v (H/W 应为 0)", got.Images.Pages[1])
	}
	if got.Images.Pages[1].T != "p" {
		t.Errorf("新页 T = %q, want \"p\"", got.Images.Pages[1].T)
	}
}
