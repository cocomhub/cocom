// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/cocomhub/cocom/cmd/server/api"
	internalComic "github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/pkg/httpwrap"
)

func TestSaveComicInfo_InvalidBody(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/comic/saveComicInfo", nil)
	req.Header.Set("Content-Type", "application/json")
	SaveComicInfo(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for nil body, got 0")
	}
}

func TestSaveComicInfo_MissingCID(t *testing.T) {
	body := map[string]any{"title": "test"}
	b, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/comic/saveComicInfo", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	SaveComicInfo(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for missing cid, got 0")
	}
}

func TestSaveComicInfo_Valid(t *testing.T) {
	ctx := context.Background()
	// 自包含：创建临时 comic 再保存，避免依赖/污染共享测试数据
	testCID := 9999
	fresh := internalComic.NewComic(&api.ComicInfo{CID: testCID})
	if err := testMemStorage.Save(ctx, fresh); err != nil {
		t.Fatalf("seed comic %d failed: %v", testCID, err)
	}
	t.Cleanup(func() { _ = testMemStorage.Delete(ctx, strconv.Itoa(testCID)) })

	// 部分字段更新：num_pages 是标量字段，与 ComicInfo 结构化字段无类型冲突
	body := map[string]any{"cid": testCID, "num_pages": 99}
	b, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/comic/saveComicInfo", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	SaveComicInfo(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Fatalf("expected code 0 for existing comic, got %d: %s", resp.Head.Code, resp.Head.Msg)
	}

	// 读取验证写入落库
	gw := httptest.NewRecorder()
	greq := httptest.NewRequest(http.MethodGet, "/api/comic/getComicInfo?cid="+strconv.Itoa(testCID), nil)
	GetComicInfo(gw, greq)
	var gresp httpwrap.ResponseInfo[map[string]any]
	if err := json.NewDecoder(gw.Body).Decode(&gresp); err != nil {
		t.Fatalf("decode get response failed: %v", err)
	}
	if gresp.Head.Code != 0 {
		t.Fatalf("get comic info failed: %d: %s", gresp.Head.Code, gresp.Head.Msg)
	}
	if np, _ := gresp.Body["num_pages"].(float64); np != 99 {
		t.Errorf("num_pages = %v, want 99 (body: %v)", gresp.Body["num_pages"], gresp.Body)
	}
}

func TestGetComicInfo_InvalidCID(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/comic/getComicInfo", nil)
	GetComicInfo(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for missing cid, got 0")
	}
}

func TestGetComicInfo_Valid(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/comic/getComicInfo?cid=1001", nil)
	GetComicInfo(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	// cid=1001 exists in testMemStorage (injected in TestMain), so expect success
	if resp.Head.Code != 0 {
		t.Errorf("expected code 0 for existing comic, got %d: %s", resp.Head.Code, resp.Head.Msg)
	}
}

func TestDownloadComic_InvalidBody(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/comic/download", nil)
	req.Header.Set("Content-Type", "application/json")
	DownloadComic(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for nil body, got 0")
	}
}

func TestDownloadComic_Async(t *testing.T) {
	// 异步分支仅校验请求被接受（code 1000 视为“异步任务已投递”），不等待后台真正完成。
	// 后台 goroutine 走 memory store 下载流程，下载目录默认在临时/相对路径，测试终止即丢弃，不阻塞用例。
	// 若未来 download 流程改长驻或失败断言需要，再改用 no-op runner 注入或 t.Cleanup 等待完成。
	body := api.DownloadComicByIDRequest{Cid: 1001, IsSync: false}
	b, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/comic/download", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	DownloadComic(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 1000 {
		t.Errorf("expected code 1000 for async download, got %d: %s", resp.Head.Code, resp.Head.Msg)
	}
}

func TestRestoreComic_InvalidBody(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/comic/restore", nil)
	req.Header.Set("Content-Type", "application/json")
	RestoreComic(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for nil body, got 0")
	}
}

func TestRestoreComic_Async(t *testing.T) {
	// 异步恢复分支只断言请求被接受（code 1000），不等待后台完成。memory store 的 restore
	// 落 archive path 到内存 map，后台 goroutine 在用例结束前完成即可，无需同步。
	// 若采用“先存档再异步恢复”的强断言会依赖 archive 命令（7z）与文件系统，超出本用例范围。
	body := api.RestoreComicByIDRequest{Cid: 1001, IsSync: false}
	b, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/comic/restore", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	RestoreComic(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 1000 {
		t.Errorf("expected code 1000 for async restore, got %d: %s", resp.Head.Code, resp.Head.Msg)
	}
}
