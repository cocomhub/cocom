// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	if err := testMemStorage.Save(ctx, internalComic.NewComic(info)); err != nil {
		t.Fatalf("seed comic %d: %v", cid, err)
	}
	t.Cleanup(func() { _ = testMemStorage.Delete(ctx, strconv.Itoa(cid)) })
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

	if got, want := savedPageNumbers(t, cid), []int{1, 3, 4, 2, 5}; !slices.Equal(got, want) {
		t.Errorf("pages after reorder = %v, want %v", got, want)
	}
}

func TestSavePages_InsertAddsPage(t *testing.T) {
	const cid = 7011
	seedPagesComic(t, cid, 3)

	w := httptest.NewRecorder()
	SavePages(w, savePagesRequestFor(t, cid, map[string]any{"action": "insert", "target_page": 2, "file": "new.png"}))

	got := savedPageNumbers(t, cid)
	if len(got) != 4 {
		t.Fatalf("pages after insert = %v, want 4 页", got)
	}
	var info api.ComicInfo
	if err := internalComic.GetComicInfo(context.Background(), cid, &info); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if info.Images.Pages[1].T != "p" {
		t.Errorf("inserted page type = %q, want \"p\"(由 .png 推导)", info.Images.Pages[1].T)
	}
	if info.NumPages != 4 {
		t.Errorf("NumPages = %d, want 4", info.NumPages)
	}
}

func TestSavePages_ReplaceUpdatesType(t *testing.T) {
	const cid = 7012
	seedPagesComic(t, cid, 3)

	w := httptest.NewRecorder()
	SavePages(w, savePagesRequestFor(t, cid, map[string]any{"page": 2, "action": "replace", "file_name": "x.webp"}))

	var info api.ComicInfo
	if err := internalComic.GetComicInfo(context.Background(), cid, &info); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if info.Images.Pages[1].T != "w" {
		t.Errorf("replaced page type = %q, want \"w\"(由 .webp 推导)", info.Images.Pages[1].T)
	}
	if len(info.Images.Pages) != 3 {
		t.Errorf("replace 不应改变页数, got %d", len(info.Images.Pages))
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
