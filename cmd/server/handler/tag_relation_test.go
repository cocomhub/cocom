// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/pkg/httpwrap"
)

// CreateTagRelation 测试

func TestCreateTagRelation_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/comic/tags/relation", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	CreateTagRelation(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Errorf("expected non-zero code for nil body, got 0")
	}
}

func TestCreateTagRelation_LessThan2Tags(t *testing.T) {
	body := map[string]any{
		"tags": []map[string]any{{"id": 1, "name": "a", "type": "tag"}},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/comic/tags/relation", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	CreateTagRelation(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Errorf("expected non-zero code with less than 2 tags, got 0")
	}
}

func TestCreateTagRelation_Valid(t *testing.T) {
	body := map[string]any{
		"tags": []map[string]any{
			{"id": 1, "name": "a", "type": "tag"},
			{"id": 2, "name": "b", "type": "tag"},
		},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/comic/tags/relation", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	CreateTagRelation(w, req)

	var resp httpwrap.ResponseInfo[map[string]any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Errorf("expected code 0 for create relation, got %d: %s", resp.Head.Code, resp.Head.Msg)
	}
}

// DeleteTagRelation 测试

func TestDeleteTagRelation_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/comic/tags/relation/delete", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	DeleteTagRelation(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Errorf("expected non-zero code for nil body, got 0")
	}
}

func TestDeleteTagRelation_EmptyID(t *testing.T) {
	body := map[string]any{"id": ""}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/comic/tags/relation/delete", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	DeleteTagRelation(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Errorf("expected non-zero code with empty id, got 0")
	}
}

func TestCreateDeleteTagRelation_RoundTrip(t *testing.T) {
	// 创建关系应返回非零 hex ID，且该 ID 可用于删除
	body := map[string]any{
		"tags": []map[string]any{
			{"id": 9001, "name": "rt-a", "type": "tag"},
			{"id": 9002, "name": "rt-b", "type": "tag"},
		},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/comic/tags/relation", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	CreateTagRelation(w, req)

	var resp httpwrap.ResponseInfo[api.CreateRelationResponse]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Fatalf("create relation failed: %d: %s", resp.Head.Code, resp.Head.Msg)
	}
	if resp.Body.ID == "" || resp.Body.ID == "000000000000000000000000" {
		t.Fatalf("created relation id = %q, want non-zero hex", resp.Body.ID)
	}

	// 按返回的 ID 删除应成功
	delBody := map[string]any{"id": resp.Body.ID}
	delData, _ := json.Marshal(delBody)
	delReq := httptest.NewRequest(http.MethodPost, "/api/comic/tags/relation/delete", bytes.NewReader(delData))
	delReq.Header.Set("Content-Type", "application/json")
	delW := httptest.NewRecorder()
	DeleteTagRelation(delW, delReq)

	var delResp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(delW.Body).Decode(&delResp); err != nil {
		t.Fatalf("decode delete response failed: %v", err)
	}
	if delResp.Head.Code != 0 {
		t.Errorf("delete relation by returned id failed: %d: %s", delResp.Head.Code, delResp.Head.Msg)
	}
}

// GetTagRelations 测试

func TestGetTagRelations_MissingParams(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/comic/tags/relation", nil)
	w := httptest.NewRecorder()

	GetTagRelations(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Errorf("expected non-zero code without type or name, got 0")
	}
}

func TestGetTagRelations_NonExistent(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/comic/tags/relation?type=tag&name=nonexistent", nil)
	w := httptest.NewRecorder()

	GetTagRelations(w, req)

	var resp httpwrap.ResponseInfo[map[string]any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	// 内存 store 找不到 tag 时返回 code 0 + 空 groups（而非错误）
	if resp.Head.Code != 0 {
		t.Errorf("expected code 0 for nonexistent tag, got %d: %s", resp.Head.Code, resp.Head.Msg)
	}
	groups, _ := resp.Body["groups"].([]any)
	if len(groups) != 0 {
		t.Errorf("groups = %v, want empty", resp.Body["groups"])
	}
}

func TestGetTagRelations_Valid(t *testing.T) {
	ctx := context.Background()

	// 自包含准备：向内存 tag store 注入唯一 tag，并创建一个包含它的关系组
	tagName := "rt-valid"
	if err := testTagStore.UpdateComicTagIncremental(ctx, "tag", 9001, tagName, "", 1); err != nil {
		t.Fatalf("seed tag failed: %v", err)
	}
	relID, err := testRelationStore.CreateRelation(ctx, []api.TagBrief{
		{ID: 9001, Name: tagName, Type: "tag"},
		{ID: 9002, Name: "rt-b", Type: "tag"},
	})
	if err != nil {
		t.Fatalf("seed relation failed: %v", err)
	}
	defer func() {
		_ = testRelationStore.DeleteRelation(ctx, relID)
		_ = testTagStore.UpdateComicTagIncremental(ctx, "tag", 9001, tagName, "", -1)
	}()

	req := httptest.NewRequest(http.MethodGet, "/api/comic/tags/relation?type=tag&name="+tagName, nil)
	w := httptest.NewRecorder()

	GetTagRelations(w, req)

	var resp httpwrap.ResponseInfo[map[string]any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Fatalf("expected code 0 for existing tag, got %d: %s", resp.Head.Code, resp.Head.Msg)
	}
	groups, _ := resp.Body["groups"].([]any)
	if len(groups) != 1 {
		t.Fatalf("groups len = %d, want 1: %v", len(groups), resp.Body["groups"])
	}
	first, ok := groups[0].(map[string]any)
	if !ok {
		t.Fatalf("groups[0] type = %T, want map[string]any", groups[0])
	}
	if id, _ := first["id"].(string); id != relID {
		t.Errorf("groups[0].id = %q, want %q", id, relID)
	}
}
