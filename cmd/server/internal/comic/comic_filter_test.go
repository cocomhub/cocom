// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package comic

import (
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

// TestBuildComicFilterFromFiltersBool 覆盖布尔值形态的 filters：
// status/redirect_to/deleted 都能正确翻译为 ComicFilter 强类型字段。
func TestBuildComicFilterFromFiltersBool(t *testing.T) {
	f := buildComicFilterFromFilters(
		"status", true,
		"redirect_to", bson.M{"$exists": false},
		"deleted", bson.M{"$ne": true},
	)
	if f.Status == nil || !*f.Status {
		t.Fatal("Status: expected true")
	}
	if f.HasRedirect == nil || *f.HasRedirect {
		t.Fatal("HasRedirect: expected false (no redirect)")
	}
	if f.Deleted == nil || *f.Deleted {
		t.Fatal("Deleted: expected false (not deleted)")
	}
}

// TestBuildComicFilterFromFiltersInt 覆盖 Mongo 查询习惯的整型形态：
// bson.M 值解出的是 int，必须安全转换为 bool 语义 — 此前 e.(bool) 会 panic。
func TestBuildComicFilterFromFiltersInt(t *testing.T) {
	f := buildComicFilterFromFilters(
		"redirect_to", bson.M{"$exists": 0}, // view/index.go 与 recommend.go 传的就是这个形态
		"deleted", bson.M{"$ne": 1},
	)
	if f.HasRedirect == nil || *f.HasRedirect {
		t.Fatal("HasRedirect: $exists=0 → hasRedirect=false expected")
	}
	if f.Deleted == nil || *f.Deleted {
		t.Fatal("Deleted: $ne=1 → not deleted → SetDeleted(false) expected")
	}

	f1 := buildComicFilterFromFilters(
		"redirect_to", bson.M{"$exists": 1},
	)
	if f1.HasRedirect == nil || !*f1.HasRedirect {
		t.Fatal("HasRedirect: $exists=1 → hasRedirect=true expected")
	}
}

// TestBuildComicFilterFromFiltersNoOp 覆盖未知键 / 未知类型保持 no-op 不 panic。
func TestBuildComicFilterFromFiltersNoOp(t *testing.T) {
	f := buildComicFilterFromFilters(
		"unknown", "ignored",
		"redirect_to", bson.M{"$exists": "nope"},
		"status", // 奇数个，尾部未配对应跳过
	)
	if f.Status != nil {
		t.Errorf("Status should be nil (odd tail)")
	}
	// 改为反射断言，确认全字段均未被设置
	if !reflect.DeepEqual(f, buildComicFilterFromFilters()) {
		t.Errorf("no-op input should produce default filter")
	}
}
