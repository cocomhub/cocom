// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package tag

import (
	"context"
	"testing"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/pkg/comic"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// searchTestComic 包装 *comic.ComicImpl 并填充多语言标题，
// 用于 GetSearchUniqueTags 的标题过滤测试（ComicImpl 的多语言 getter 恒返回空）。
type searchTestComic struct {
	*comic.ComicImpl
	english  string
	japanese string
	pretty   string
}

func (c *searchTestComic) GetTitleEnglish() string  { return c.english }
func (c *searchTestComic) GetTitleJapanese() string { return c.japanese }
func (c *searchTestComic) GetTitlePretty() string   { return c.pretty }

func TestMemoryTagStore_GetSearchUniqueTags_QueryFilter(t *testing.T) {
	ctx := context.Background()
	ms := comic.NewMemoryStorage()

	c1 := &searchTestComic{
		ComicImpl: &comic.ComicImpl{
			ID:    "1",
			Title: "Test Comic One",
			Tags: []comic.Tag{
				{ID: 1, Name: "test", Type: "tag"},
				{ID: 2, Name: "artist1", Type: "artist"},
			},
		},
		english: "Test Comic One",
		pretty:  "Test Comic One",
	}
	c2 := &searchTestComic{
		ComicImpl: &comic.ComicImpl{
			ID:    "2",
			Title: "Other Story",
			Tags: []comic.Tag{
				{ID: 3, Name: "romance", Type: "tag"},
			},
		},
		english: "Other Story",
		pretty:  "Other Story",
	}
	if err := ms.Save(ctx, c1); err != nil {
		t.Fatalf("Save c1 failed: %v", err)
	}
	if err := ms.Save(ctx, c2); err != nil {
		t.Fatalf("Save c2 failed: %v", err)
	}

	SetDefaultComicStore(ms)
	defer ResetDefaultComicStore()

	store := NewMemoryTagStore()

	t.Run("query filters to matching comics", func(t *testing.T) {
		// query "Test"（大小写不敏感）只匹配 c1
		tags, cidList, total, err := store.GetSearchUniqueTags(ctx, "Test", 100, 0)
		if err != nil {
			t.Fatalf("GetSearchUniqueTags failed: %v", err)
		}
		if total != 1 {
			t.Errorf("total = %d, want 1 (only c1 matches)", total)
		}
		if len(cidList) != 1 || cidList[0] != 1 {
			t.Errorf("cidList = %v, want [1]", cidList)
		}
		got := make(map[int]bool)
		for _, tg := range tags {
			got[tg.ID] = true
		}
		if !got[1] || !got[2] {
			t.Errorf("tags should include c1's tags {1,2}, got %v", tags)
		}
		if got[3] {
			t.Error("tags should not include c2's tag id 3")
		}
	})

	t.Run("empty query scans all comics", func(t *testing.T) {
		tags, cidList, total, err := store.GetSearchUniqueTags(ctx, "", 100, 0)
		if err != nil {
			t.Fatalf("GetSearchUniqueTags(empty) failed: %v", err)
		}
		if total != 2 {
			t.Errorf("total = %d, want 2 (all comics)", total)
		}
		if len(cidList) != 2 {
			t.Errorf("cidList len = %d, want 2", len(cidList))
		}
		if len(tags) != 3 {
			t.Errorf("unique tags len = %d, want 3", len(tags))
		}
	})
}

func TestMemoryTagStore_UpdateComicTagIncremental_MaxTag(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryTagStore()

	maxID, err := store.GetMaxTagID(ctx)
	if err != nil {
		t.Fatalf("GetMaxTagID failed: %v", err)
	}
	if maxID != 1000000000 {
		t.Fatalf("initial maxTag = %d, want 1000000000", maxID)
	}

	// 模拟 handler 用 GetMaxTagID()+1 分配新 tag ID
	if err := store.UpdateComicTagIncremental(ctx, "tag", 1000000001, "new1", "", 1); err != nil {
		t.Fatalf("UpdateComicTagIncremental #1 failed: %v", err)
	}
	maxID, _ = store.GetMaxTagID(ctx)
	if maxID != 1000000001 {
		t.Errorf("maxTag after first add = %d, want 1000000001", maxID)
	}

	if err := store.UpdateComicTagIncremental(ctx, "tag", 1000000002, "new2", "", 1); err != nil {
		t.Fatalf("UpdateComicTagIncremental #2 failed: %v", err)
	}
	maxID, _ = store.GetMaxTagID(ctx)
	if maxID != 1000000002 {
		t.Errorf("maxTag after second add = %d, want 1000000002", maxID)
	}
}

func TestMemoryRelationStore_HexIDRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryRelationStore()

	id, err := s.CreateRelation(ctx, []api.TagBrief{
		{ID: 1, Name: "a", Type: "tag"},
		{ID: 2, Name: "b", Type: "tag"},
	})
	if err != nil {
		t.Fatalf("CreateRelation failed: %v", err)
	}
	if id == "" || id == "000000000000000000000000" {
		t.Errorf("relation id = %q, want non-zero hex", id)
	}
	if _, oidErr := primitive.ObjectIDFromHex(id); oidErr != nil {
		t.Errorf("relation id %q is not a valid hex ObjectID: %v", id, oidErr)
	}

	// GetRelationsForTag 应返回包含该 id 的关系组
	groups, err := s.GetRelationsForTag(ctx, "tag", 1)
	if err != nil {
		t.Fatalf("GetRelationsForTag failed: %v", err)
	}
	if len(groups) != 1 || groups[0].ID != id {
		t.Errorf("groups = %+v, want 1 group with id %q", groups, id)
	}

	// 按 id 删除应成功
	if err := s.DeleteRelation(ctx, id); err != nil {
		t.Errorf("DeleteRelation by id failed: %v", err)
	}
	groups, _ = s.GetRelationsForTag(ctx, "tag", 1)
	if len(groups) != 0 {
		t.Errorf("after delete groups len = %d, want 0", len(groups))
	}
}
