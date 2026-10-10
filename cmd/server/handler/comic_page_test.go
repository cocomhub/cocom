// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"github.com/cocomhub/cocom/cmd/server/api"
	internalComic "github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/pkg/httpwrap"
)

func TestGetComicPages_InvalidCID(t *testing.T) {
	body := map[string]any{"cid": 0}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/comic/getComicPages", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	GetComicPages(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for invalid cid, got 0")
	}
}

func TestSavePages_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/comic/savePages", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	SavePages(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for empty request body, got 0")
	}
}

func TestSavePages_InvalidCID(t *testing.T) {
	body := map[string]any{"cid": 0, "pages": []any{}}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/comic/savePages", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	SavePages(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for invalid cid, got 0")
	}
}

// savePagesRequestFor 构造 savePages 请求。
func savePagesRequestFor(t *testing.T, cid int, pages ...map[string]any) *http.Request {
	t.Helper()
	b, err := json.Marshal(map[string]any{"cid": cid, "pages": pages})
	if err != nil {
		t.Fatalf("marshal savePages request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/comic/savePages", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// seedPagesComic 写入一部带 n 页的漫画；页序号记录在 PicInfo.H，便于断言具体删掉了哪一页。
func seedPagesComic(t *testing.T, cid, n int) {
	t.Helper()
	ctx := context.Background()

	info := &api.ComicInfo{CID: cid}
	for i := 1; i <= n; i++ {
		info.Images.Pages = append(info.Images.Pages, api.PicInfo{T: "j", H: i, W: i, Status: true})
	}
	// 同时写出真实页文件：savePages 现在会真正搬动磁盘文件，断言必须能看到效果。
	saveDir := info.SaveDir()
	if err := os.MkdirAll(saveDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", saveDir, err)
	}
	for i := 1; i <= n; i++ {
		writePageFile(t, saveDir, fmt.Sprintf("%d.jpg", i), pageContent(i))
	}
	if err := testMemStorage.Save(ctx, internalComic.NewComic(info)); err != nil {
		t.Fatalf("seed comic %d: %v", cid, err)
	}
	t.Cleanup(func() {
		_ = testMemStorage.Delete(ctx, strconv.Itoa(cid))
		_ = os.RemoveAll(saveDir)
	})
}

// savedPageNumbers 读回漫画的页序号（PicInfo.H）。
func savedPageNumbers(t *testing.T, cid int) []int {
	t.Helper()

	var got api.ComicInfo
	if err := internalComic.GetComicInfo(context.Background(), cid, &got); err != nil {
		t.Fatalf("read back comic %d: %v", cid, err)
	}
	nums := make([]int, 0, len(got.Images.Pages))
	for _, p := range got.Images.Pages {
		nums = append(nums, p.H)
	}
	return nums
}

func TestSavePages_DeleteMiddlePageRemovesExactlyOne(t *testing.T) {
	const cid = 7001
	seedPagesComic(t, cid, 5)

	w := httptest.NewRecorder()
	SavePages(w, savePagesRequestFor(t, cid, map[string]any{"page": 3, "action": "delete"}))

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Fatalf("savePages code = %d (%s), want 0", resp.Head.Code, resp.Head.Msg)
	}

	if got, want := savedPageNumbers(t, cid), []int{1, 2, 4, 5}; !slices.Equal(got, want) {
		t.Errorf("pages after deleting page 3 = %v, want %v", got, want)
	}
}

func TestSavePages_DeleteLastPageDoesNotPanic(t *testing.T) {
	const cid = 7002
	seedPagesComic(t, cid, 5)

	w := httptest.NewRecorder()
	SavePages(w, savePagesRequestFor(t, cid, map[string]any{"page": 5, "action": "delete"}))

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Fatalf("savePages code = %d (%s), want 0", resp.Head.Code, resp.Head.Msg)
	}

	if got, want := savedPageNumbers(t, cid), []int{1, 2, 3, 4}; !slices.Equal(got, want) {
		t.Errorf("pages after deleting page 5 = %v, want %v", got, want)
	}
}

func TestSavePages_DeleteOutOfRangeIsNoop(t *testing.T) {
	const cid = 7003
	seedPagesComic(t, cid, 3)

	w := httptest.NewRecorder()
	SavePages(w, savePagesRequestFor(t, cid, map[string]any{"page": 9, "action": "delete"}))

	if got, want := savedPageNumbers(t, cid), []int{1, 2, 3}; !slices.Equal(got, want) {
		t.Errorf("pages after out-of-range delete = %v, want %v", got, want)
	}
}

func TestSavePages_ReorderMovesPage(t *testing.T) {
	const cid = 7010
	seedPagesComic(t, cid, 5)

	w := httptest.NewRecorder()
	SavePages(w, savePagesRequestFor(t, cid, map[string]any{"page": 2, "action": "reorder", "to_page": 4}))

	// 前端 reorder 是「交换两页」，不是「移动」。
	if got, want := savedPageNumbers(t, cid), []int{1, 4, 3, 2, 5}; !slices.Equal(got, want) {
		t.Errorf("pages after swap(2,4) = %v, want %v", got, want)
	}
}

func TestSavePages_InsertAddsPage(t *testing.T) {
	const cid = 7011
	seedPagesComic(t, cid, 3)

	// insert 的 page 字段是 after_page（插到该页之后），文件来自上传接口。
	upload := uploadTestPage(t, cid, "new.png")

	w := httptest.NewRecorder()
	SavePages(w, savePagesRequestFor(t, cid,
		map[string]any{"action": "insert", "page": 2, "file": upload}))

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Fatalf("insert code = %d (%s), want 0", resp.Head.Code, resp.Head.Msg)
	}

	var info api.ComicInfo
	if err := internalComic.GetComicInfo(context.Background(), cid, &info); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if len(info.Images.Pages) != 4 {
		t.Fatalf("pages after insert = %d, want 4", len(info.Images.Pages))
	}
	// after_page=2 → 新页落在位置 3。
	if info.Images.Pages[2].T != "p" {
		t.Errorf("inserted page type = %q, want \"p\"（由 .png 推导）", info.Images.Pages[2].T)
	}
	if info.NumPages != 4 {
		t.Errorf("NumPages = %d, want 4", info.NumPages)
	}
	if got, want := savedPageNumbers(t, cid), []int{1, 2, 0, 3}; !slices.Equal(got, want) {
		t.Errorf("page order after insert = %v, want %v（0 = 新插入页）", got, want)
	}
}

func TestSavePages_ReplaceUpdatesType(t *testing.T) {
	const cid = 7012
	seedPagesComic(t, cid, 3)

	upload := uploadTestPage(t, cid, "x.png")

	w := httptest.NewRecorder()
	SavePages(w, savePagesRequestFor(t, cid,
		map[string]any{"page": 2, "action": "replace", "file": upload}))

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Fatalf("replace code = %d (%s), want 0", resp.Head.Code, resp.Head.Msg)
	}

	var info api.ComicInfo
	if err := internalComic.GetComicInfo(context.Background(), cid, &info); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if info.Images.Pages[1].T != "p" {
		t.Errorf("replaced page type = %q, want \"p\"（由 .png 推导）", info.Images.Pages[1].T)
	}
	if len(info.Images.Pages) != 3 {
		t.Errorf("replace 不应改变页数, got %d", len(info.Images.Pages))
	}
	// 磁盘上第 2 页应变成 webp，旧的 2.jpg 不再存在。
	dir := info.SaveDir()
	if !fileExists(filepath.Join(dir, "2.png")) {
		t.Error("替换后缺少 2.png")
	}
	if fileExists(filepath.Join(dir, "2.jpg")) {
		t.Error("替换后旧的 2.jpg 仍然存在")
	}
}

func TestSavePages_UnknownActionRejected(t *testing.T) {
	const cid = 7013
	seedPagesComic(t, cid, 3)

	w := httptest.NewRecorder()
	SavePages(w, savePagesRequestFor(t, cid, map[string]any{"page": 1, "action": "explode"}))

	if got, want := savedPageNumbers(t, cid), []int{1, 2, 3}; !slices.Equal(got, want) {
		t.Errorf("未知动作不得改动页面: %v", got)
	}
}

// TestSavePages_DeleteMultiplePagesUsesOriginalNumbers 锁定「同一请求内多条删除」的语义。
//
// p.Page 是提交时的原始页号，而删除是在逐步变短的切片上按下标进行的：若按请求顺序
// （升序）执行，「先删第 2 页、再删第 3 页」第二条实际删掉的是原第 4 页、原第 3 页残留。
// 因此删除必须按页号降序执行。这里断言删掉的正是原第 2、3 页（H 标记原始页号）。
func TestSavePages_DeleteMultiplePagesUsesOriginalNumbers(t *testing.T) {
	const cid = 7011
	seedPagesComic(t, cid, 5)

	w := httptest.NewRecorder()
	SavePages(w, savePagesRequestFor(t, cid,
		map[string]any{"page": 2, "action": "delete"},
		map[string]any{"page": 3, "action": "delete"},
	))

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Fatalf("savePages code = %d (%s), want 0", resp.Head.Code, resp.Head.Msg)
	}

	if got, want := savedPageNumbers(t, cid), []int{1, 4, 5}; !slices.Equal(got, want) {
		t.Errorf("pages after deleting 2 and 3 = %v, want %v", got, want)
	}
}

// TestSavePages_DeleteAllPagesLeavesEmpty 覆盖全部删除（降序执行时不能越界）。
func TestSavePages_DeleteAllPagesLeavesEmpty(t *testing.T) {
	const cid = 7012
	seedPagesComic(t, cid, 3)

	w := httptest.NewRecorder()
	SavePages(w, savePagesRequestFor(t, cid,
		map[string]any{"page": 1, "action": "delete"},
		map[string]any{"page": 2, "action": "delete"},
		map[string]any{"page": 3, "action": "delete"},
	))

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Fatalf("savePages code = %d (%s), want 0", resp.Head.Code, resp.Head.Msg)
	}
	if got := savedPageNumbers(t, cid); len(got) != 0 {
		t.Errorf("pages after deleting all = %v, want empty", got)
	}
}
