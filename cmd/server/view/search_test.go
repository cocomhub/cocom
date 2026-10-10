// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package view

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestBuildSearchTitleFilter_EscapesRegexMeta(t *testing.T) {
	filter := buildSearchTitleFilter(".*")
	if len(filter) != 3 {
		t.Fatalf("filter len = %d, want 3 (english/japanese/pretty)", len(filter))
	}

	for _, cond := range filter {
		for field, v := range cond {
			sub, ok := v.(bson.M)
			if !ok {
				t.Fatalf("%s: value type = %T, want bson.M", field, v)
			}
			re, ok := sub["$regex"].(primitive.Regex)
			if !ok {
				t.Fatalf("%s: $regex type = %T, want primitive.Regex", field, sub["$regex"])
			}
			if re.Pattern != `\.\*` {
				t.Errorf("%s: pattern = %q, want %q（用户输入必须转义为字面量）", field, re.Pattern, `\.\*`)
			}
			if re.Options != "i" {
				t.Errorf("%s: options = %q, want \"i\"", field, re.Options)
			}
		}
	}
}

func TestBuildSearchTitleFilter_PlainQueryUnchanged(t *testing.T) {
	cond := buildSearchTitleFilter("cocom")[0]
	re := cond["title.english"].(bson.M)["$regex"].(primitive.Regex)
	if re.Pattern != "cocom" {
		t.Errorf("pattern = %q, want \"cocom\"", re.Pattern)
	}
}

func newSearchCtx(t *testing.T, target string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, target, nil)
	return c
}

func TestParseSearchResultPageArgs(t *testing.T) {
	t.Run("默认页码为 1", func(t *testing.T) {
		page, query, err := parseSearchResultPageArgs(newSearchCtx(t, "/search?q=abc"))
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if page != 1 || query != "abc" {
			t.Errorf("page=%d query=%q, want 1/\"abc\"", page, query)
		}
	})

	t.Run("缺 q 报错", func(t *testing.T) {
		if _, _, err := parseSearchResultPageArgs(newSearchCtx(t, "/search")); err == nil {
			t.Fatal("缺 q 必须报错")
		}
	})

	t.Run("超长 q 报错", func(t *testing.T) {
		long := strings.Repeat("a", maxSearchQueryLen+1)
		if _, _, err := parseSearchResultPageArgs(newSearchCtx(t, "/search?q="+long)); err == nil {
			t.Fatal("超长 q 必须报错")
		}
	})

	t.Run("边界长度 q 允许", func(t *testing.T) {
		ok := strings.Repeat("a", maxSearchQueryLen)
		if _, _, err := parseSearchResultPageArgs(newSearchCtx(t, "/search?q="+ok)); err != nil {
			t.Errorf("长度 = maxSearchQueryLen 应允许，got %v", err)
		}
	})
}
