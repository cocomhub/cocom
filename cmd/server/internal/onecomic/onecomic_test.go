// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package onecomic

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/pkg/comic"
)

func TestMemoryOneComicStore_RoundTrip(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryOneComicStore()

	if err := s.Update(ctx, "c1", map[string]any{"comicid": "c1", "name": "One"}); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	var got map[string]any
	if err := s.Get(ctx, "c1", &got); err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got["name"] != "One" {
		t.Errorf("Get name = %v, want One", got["name"])
	}
}

func TestMemoryOneComicStore_GetNotFound(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryOneComicStore()

	var info map[string]any
	if err := s.Get(ctx, "missing", &info); err == nil {
		t.Error("Get(missing) = nil error, want error")
	}
}

func TestMemoryOneComicStore_UpdateMerges(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryOneComicStore()

	if err := s.Update(ctx, "c2", map[string]any{"a": 1}); err != nil {
		t.Fatalf("Update #1 failed: %v", err)
	}
	if err := s.Update(ctx, "c2", map[string]any{"b": 2}); err != nil {
		t.Fatalf("Update #2 failed: %v", err)
	}

	var got map[string]any
	if err := s.Get(ctx, "c2", &got); err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got["a"] != float64(1) || got["b"] != float64(2) {
		t.Errorf("merged doc = %v, want {a:1 b:2}", got)
	}
}

func TestCacheKeyFunctions(t *testing.T) {
	if got := CacheKeyFilter(); got != "total" {
		t.Errorf("CacheKeyFilter() = %q, want total", got)
	}
	if got := CacheKeyFilter("a", 1); got != "filters:a:1" {
		t.Errorf("CacheKeyFilter(a,1) = %q, want filters:a:1", got)
	}
	if got := CacheKeyOneComicInfo("c1"); got != "oneComicInfo:c1" {
		t.Errorf("CacheKeyOneComicInfo = %q, want oneComicInfo:c1", got)
	}
	if got := CacheKeyRangeOneComicInfos(10, 20, "a"); got != "oneComicInfos:limit:10:skip:20:filters:a" {
		t.Errorf("CacheKeyRangeOneComicInfos = %q, want oneComicInfos:limit:10:skip:20:filters:a", got)
	}
	if got := CacheKeyCountTotalOneComicInfos("a", 1); got != "oneComicInfos:count:filters:a:1" {
		t.Errorf("CacheKeyCountTotalOneComicInfos = %q, want oneComicInfos:count:filters:a:1", got)
	}
}

func TestComic_ImplementsInterface(t *testing.T) {
	c := NewComic(&api.OneComicInfo{Comicid: "9", Name: "Nine"})

	if c.GetID() != "9" {
		t.Errorf("GetID() = %q, want 9", c.GetID())
	}
	if c.GetTitle() != "Nine" {
		t.Errorf("GetTitle() = %q, want Nine", c.GetTitle())
	}
	if c.GetTitleEnglish() != "Nine" || c.GetTitlePretty() != "Nine" {
		t.Errorf("GetTitleEnglish/Pretty = %q/%q, want Nine/Nine", c.GetTitleEnglish(), c.GetTitlePretty())
	}
	if c.GetTags() != nil {
		t.Errorf("GetTags() = %v, want nil", c.GetTags())
	}

	var _ comic.Comic = c
}

func TestNewComicByObject(t *testing.T) {
	// map 分支应正确解码到 Comic
	c, err := NewComicByObject(map[string]any{"comicid": "5", "name": "Five"})
	if err != nil {
		t.Fatalf("NewComicByObject(map) failed: %v", err)
	}
	if c == nil || c.GetID() != "5" || c.GetTitle() != "Five" {
		t.Errorf("NewComicByObject(map) = %+v, want id=5 title=Five", c)
	}

	// 非法类型应返回错误
	if _, err := NewComicByObject(42); err == nil {
		t.Error("NewComicByObject(42) = nil error, want error")
	}
}

func TestStorage_DelegatesToInner(t *testing.T) {
	ctx := context.Background()
	inner := comic.NewMemoryStorage()

	impl := &comic.ComicImpl{ID: "3", Title: "Inner Comic"}
	if err := inner.Save(ctx, impl); err != nil {
		t.Fatalf("Save inner comic failed: %v", err)
	}

	s := NewTestStorage(inner)

	// Get 委托给 inner
	got, err := s.Get(ctx, "3")
	if err != nil {
		t.Fatalf("Storage.Get failed: %v", err)
	}
	if got == nil || got.GetID() != "3" {
		t.Errorf("Storage.Get = %+v, want id=3", got)
	}

	// 不存在的 id 应返回 inner 的 ErrComicNotFound
	if _, err := s.Get(ctx, "missing"); !errors.Is(err, comic.ErrComicNotFound) {
		t.Errorf("Storage.Get(missing) err = %v, want ErrComicNotFound", err)
	}
}

// TestDefaultOneComicStore_ConcurrentSetResetGet 回归 Batch B：包级 defaultStore
// 并发 Set/Reset/Get 无数据竞争（-race 守护）。
func TestDefaultOneComicStore_ConcurrentSetResetGet(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if n%2 == 0 {
				SetDefaultOneComicStore(NewMemoryOneComicStore())
			} else {
				_ = GetDefaultOneComicStore()
			}
			if n == 7 {
				ResetDefaultOneComicStore()
			}
		}(i)
	}
	wg.Wait()
	ResetDefaultOneComicStore()
}
