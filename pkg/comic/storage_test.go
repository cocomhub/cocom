// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package comic

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"
)

func TestMemoryStorage_SaveAndGet(t *testing.T) {
	ctx := t.Context()
	ms := NewMemoryStorage()

	comic := NewComic("1", "Test Comic", []Image{{ID: "1", Path: "p1.jpg"}})
	err := ms.Save(ctx, comic)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	got, err := ms.Get(ctx, "1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.GetID() != "1" {
		t.Errorf("Get got ID %q, want %q", got.GetID(), "1")
	}
	if got.GetTitle() != "Test Comic" {
		t.Errorf("Get got Title %q, want %q", got.GetTitle(), "Test Comic")
	}

	// Get non-existent returns error
	_, err = ms.Get(ctx, "999")
	if err == nil {
		t.Error("Get of non-existent comic should return error")
	}
}

func TestMemoryStorage_SaveDuplicate(t *testing.T) {
	ctx := t.Context()
	ms := NewMemoryStorage()

	c1 := NewComic("1", "First", nil)
	if err := ms.Save(ctx, c1); err != nil {
		t.Fatal(err)
	}

	c2 := NewComic("1", "Second", nil)
	if err := ms.Save(ctx, c2); err != nil {
		t.Fatal(err)
	}

	got, err := ms.Get(ctx, "1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.GetTitle() != "Second" {
		t.Errorf("Save duplicate should overwrite, got title %q", got.GetTitle())
	}
}

func TestMemoryStorage_Update(t *testing.T) {
	ctx := t.Context()
	ms := NewMemoryStorage()

	original := NewComic("1", "Original", []Image{{ID: "1", Path: "p1.jpg"}})
	if err := ms.Save(ctx, original); err != nil {
		t.Fatal(err)
	}

	updated := NewComic("1", "Updated", []Image{{ID: "1", Path: "p1_updated.jpg"}})
	err := ms.Update(ctx, updated.Object())
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, err := ms.Get(ctx, "1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.GetTitle() != "Updated" {
		t.Errorf("Update: title = %q, want %q", got.GetTitle(), "Updated")
	}
	if len(got.GetImages()) != 1 || got.GetImages()[0].Path != "p1_updated.jpg" {
		t.Errorf("Update: images not updated, got %v", got.GetImages())
	}

	// Update non-existent returns error
	nonExistent := NewComic("999", "Ghost", nil)
	err = ms.Update(ctx, nonExistent.Object())
	if err == nil {
		t.Error("Update of non-existent comic should return error")
	}
}

func TestMemoryStorage_UpdateByMap(t *testing.T) {
	ctx := t.Context()
	ms := NewMemoryStorage()

	original := NewComic("1", "Original", nil)
	if err := ms.Save(ctx, original); err != nil {
		t.Fatal(err)
	}

	// Update via map[string]any
	err := ms.Update(ctx, map[string]any{
		"id":    "1",
		"title": "ViaMap",
	})
	if err != nil {
		t.Fatalf("Update via map failed: %v", err)
	}

	got, err := ms.Get(ctx, "1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.GetTitle() != "ViaMap" {
		t.Errorf("Update via map: title = %q, want %q", got.GetTitle(), "ViaMap")
	}
}

func TestMemoryStorage_Delete(t *testing.T) {
	ctx := t.Context()
	ms := NewMemoryStorage()

	if err := ms.Save(ctx, NewComic("1", "ToDelete", nil)); err != nil {
		t.Fatal(err)
	}

	if err := ms.Delete(ctx, "1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := ms.Get(ctx, "1")
	if err == nil {
		t.Error("Get after Delete should return error")
	}

	// Delete non-existent returns error
	err = ms.Delete(ctx, "999")
	if err == nil {
		t.Error("Delete of non-existent comic should return error")
	}
}

func TestMemoryStorage_FindTotal(t *testing.T) {
	ctx := t.Context()
	ms := NewMemoryStorage()

	// Empty storage
	total, err := ms.FindTotal(ctx, nil)
	if err != nil {
		t.Fatalf("FindTotal empty failed: %v", err)
	}
	if total != 0 {
		t.Errorf("FindTotal empty: got %d, want 0", total)
	}

	// Add comics
	for i := 1; i <= 5; i++ {
		id := fmt.Sprintf("%d", i)
		if saveErr := ms.Save(ctx, NewComic(id, "Comic "+id, nil)); saveErr != nil {
			t.Fatal(saveErr)
		}
	}

	total, err = ms.FindTotal(ctx, nil)
	if err != nil {
		t.Fatalf("FindTotal failed: %v", err)
	}
	if total != 5 {
		t.Errorf("FindTotal: got %d, want 5", total)
	}

	// FindTotal with filter
	pat := "Comic [12]"
	total, err = ms.FindTotal(ctx, &ComicFilter{
		TitlePattern: &pat,
	})
	if err != nil {
		t.Fatalf("FindTotal with filter failed: %v", err)
	}
	if total != 2 {
		t.Errorf("FindTotal with filter: got %d, want 2", total)
	}
}

func TestMemoryStorage_FindChannel(t *testing.T) {
	ctx := t.Context()
	ms := NewMemoryStorage()

	// Empty
	ch, err := ms.FindChannel(ctx, nil)
	if err != nil {
		t.Fatalf("FindChannel empty failed: %v", err)
	}
	if v, ok := <-ch; ok {
		t.Errorf("empty channel should be closed immediately, got %v", v)
	}

	// Add comics
	for i := 1; i <= 3; i++ {
		id := fmt.Sprintf("%d", i)
		if saveErr2 := ms.Save(ctx, NewComic(id, "C"+id, nil)); saveErr2 != nil {
			t.Fatal(saveErr2)
		}
	}

	ch, err = ms.FindChannel(ctx, nil)
	if err != nil {
		t.Fatalf("FindChannel failed: %v", err)
	}

	var collected []Comic
	for c := range ch {
		collected = append(collected, c)
	}

	if len(collected) != 3 {
		t.Errorf("FindChannel: got %d comics, want 3", len(collected))
	}
}

func TestMemoryStorage_ArchiveByID(t *testing.T) {
	ctx := t.Context()
	ms := NewMemoryStorage()

	comic := NewComic("1", "ToArchive", []Image{{ID: "1", Path: "p1.jpg"}})
	if err := ms.Save(ctx, comic); err != nil {
		t.Fatal(err)
	}

	err := ms.ArchiveByID(ctx, "1")
	if err != nil {
		t.Fatalf("ArchiveByID failed: %v", err)
	}

	got, err := ms.Get(ctx, "1")
	if err != nil {
		t.Fatalf("Get after ArchiveByID failed: %v", err)
	}
	if got.GetArchivePath() == "" {
		t.Error("ArchiveByID: archive path should not be empty after archiving")
	}

	// Archive non-existent returns error
	err = ms.ArchiveByID(ctx, "999")
	if err == nil {
		t.Error("ArchiveByID of non-existent comic should return error")
	}
}

func TestMemoryStorage_RestoreByID(t *testing.T) {
	ctx := t.Context()
	ms := NewMemoryStorage()

	comic := NewComic("1", "ToRestore", nil)
	if err := ms.Save(ctx, comic); err != nil {
		t.Fatal(err)
	}

	if err := ms.ArchiveByID(ctx, "1"); err != nil {
		t.Fatal(err)
	}

	if err := ms.RestoreByID(ctx, "1"); err != nil {
		t.Fatalf("RestoreByID failed: %v", err)
	}

	got, err := ms.Get(ctx, "1")
	if err != nil {
		t.Fatalf("Get after RestoreByID failed: %v", err)
	}
	if got.GetArchivePath() != "" {
		t.Errorf("RestoreByID: archive path should be empty after restore, got %q", got.GetArchivePath())
	}

	// Restore unarchived comic is a no-op (no error)
	if restoreErr := ms.RestoreByID(ctx, "1"); restoreErr != nil {
		t.Errorf("RestoreByID of unarchived comic should not error, got %v", restoreErr)
	}

	// Restore non-existent returns error
	err = ms.RestoreByID(ctx, "999")
	if err == nil {
		t.Error("RestoreByID of non-existent comic should return error")
	}
}

func TestMemoryStorage_FindWithFilter(t *testing.T) {
	ctx := t.Context()
	ms := NewMemoryStorage()

	comics := []struct {
		id    string
		title string
	}{
		{"1", "Naruto Chapter 1"},
		{"2", "Naruto Chapter 2"},
		{"3", "One Piece Chapter 1"},
		{"4", "Bleach Chapter 1"},
	}
	for _, c := range comics {
		if err := ms.Save(ctx, NewComic(c.id, c.title, nil)); err != nil {
			t.Fatal(err)
		}
	}

	// Filter by title pattern
	pat := "Naruto"
	filter := &ComicFilter{TitlePattern: &pat}
	results, err := ms.Find(ctx, filter)
	if err != nil {
		t.Fatalf("Find with filter failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Find with 'Naruto' filter: got %d, want 2", len(results))
	}

	// Filter with no match
	pat2 := "Dragon Ball"
	filter = &ComicFilter{TitlePattern: &pat2}
	results, err = ms.Find(ctx, filter)
	if err != nil {
		t.Fatalf("Find with no-match filter failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Find with no-match: got %d, want 0", len(results))
	}

	// Nil filter returns all
	results, err = ms.Find(ctx, nil)
	if err != nil {
		t.Fatalf("Find with nil filter failed: %v", err)
	}
	if len(results) != 4 {
		t.Errorf("Find all: got %d, want 4", len(results))
	}
}

func TestMemoryStorage_SaveVerifyResult(t *testing.T) {
	ctx := t.Context()
	ms := NewMemoryStorage()

	comic := NewComic("1", "ToVerify", nil)
	if err := ms.Save(ctx, comic); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	result := &VerifyResult{
		ComicID:                 "1",
		Valid:                   true,
		InvalidCount:            0,
		InvalidSubsamplingCount: 0,
		FixedCount:              0,
		Timestamp:               now,
	}

	err := ms.SaveVerifyResult(ctx, result)
	if err != nil {
		t.Fatalf("SaveVerifyResult failed: %v", err)
	}

	got, err := ms.Get(ctx, "1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !got.IsValid() {
		t.Error("SaveVerifyResult: comic should be valid")
	}

	// Non-existent
	result.ComicID = "999"
	err = ms.SaveVerifyResult(ctx, result)
	if err == nil {
		t.Error("SaveVerifyResult of non-existent comic should return error")
	}
}

func TestMemoryStorage_Concurrency(t *testing.T) {
	ctx := t.Context()
	ms := NewMemoryStorage()

	const goroutines = 20
	var wg sync.WaitGroup

	// Concurrent writes
	for i := range goroutines {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			id := fmt.Sprintf("%d", i)
			_ = ms.Save(ctx, NewComic(id, "Concurrent", nil))
		}()
	}
	wg.Wait()

	// Verify all were saved
	total, err := ms.FindTotal(ctx, nil)
	if err != nil {
		t.Fatalf("FindTotal after concurrent writes failed: %v", err)
	}
	if total != int64(goroutines) {
		t.Errorf("Concurrent writes: got %d, want %d", total, goroutines)
	}

	// Concurrent reads
	var readWg sync.WaitGroup
	for i := range goroutines {
		readWg.Add(1)
		i := i
		go func() {
			defer readWg.Done()
			id := fmt.Sprintf("%d", i)
			_, _ = ms.Get(ctx, id)
		}()
	}
	readWg.Wait()
}

// TestMemoryStorage_ArchivePathPersistence verifies that archive paths
// survive Get/Find cycles.
func TestMemoryStorage_ArchivePathPersistence(t *testing.T) {
	ctx := t.Context()
	ms := NewMemoryStorage()

	if err := ms.Save(ctx, NewComic("1", "Archivable", nil)); err != nil {
		t.Fatal(err)
	}
	if err := ms.ArchiveByID(ctx, "1"); err != nil {
		t.Fatal(err)
	}

	// Get
	got, err := ms.Get(ctx, "1")
	if err != nil {
		t.Fatal(err)
	}
	if got.GetArchivePath() == "" {
		t.Error("Get after ArchiveByID: archive path is empty")
	}
}

func TestMemoryStorage_FindChannelSortOrder(t *testing.T) {
	ctx := t.Context()
	ms := NewMemoryStorage()

	ids := []string{"3", "1", "2"}
	for _, id := range ids {
		if err := ms.Save(ctx, NewComic(id, "C"+id, nil)); err != nil {
			t.Fatal(err)
		}
	}

	ch, err := ms.FindChannel(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	var collected []Comic
	for c := range ch {
		collected = append(collected, c)
	}

	if !sort.SliceIsSorted(collected, func(i, j int) bool {
		return collected[i].GetID() < collected[j].GetID()
	}) {
		t.Error("FindChannel: results not sorted by ID ascending")
	}
}

func TestMemoryStorage_FindByTags(t *testing.T) {
	ctx := t.Context()
	ms := NewMemoryStorage()

	comic1 := &ComicImpl{
		ID:    "1",
		Title: "Naruto",
		Tags: []Tag{
			{ID: 1, Name: "action", Type: "genre"},
			{ID: 2, Name: "shounen", Type: "genre"},
		},
	}
	comic2 := &ComicImpl{
		ID:    "2",
		Title: "One Piece",
		Tags: []Tag{
			{ID: 1, Name: "action", Type: "genre"},
			{ID: 3, Name: "adventure", Type: "genre"},
		},
	}
	comic3 := &ComicImpl{
		ID:    "3",
		Title: "Bleach",
		Tags: []Tag{
			{ID: 2, Name: "shounen", Type: "genre"},
			{ID: 4, Name: "supernatural", Type: "genre"},
		},
	}
	comic4 := &ComicImpl{
		ID:    "4",
		Title: "Tokyo Ghoul",
		Tags: []Tag{
			{ID: 5, Name: "horror", Type: "genre"},
			{ID: 6, Name: "seinen", Type: "genre"},
		},
	}

	for _, c := range []Comic{comic1, comic2, comic3, comic4} {
		if err := ms.Save(ctx, c); err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

	t.Run("matching tags returns correct count", func(t *testing.T) {
		tags := comic1.GetTags()
		results, err := ms.FindByTags(ctx, tags, "genre", 1, 10)
		if err != nil {
			t.Fatalf("FindByTags failed: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("FindByTags: got %d results, want 2", len(results))
		}
		gotIDs := make(map[string]bool)
		for _, r := range results {
			gotIDs[r.GetID()] = true
		}
		if !gotIDs["2"] {
			t.Error("FindByTags: result should include comic 2")
		}
		if !gotIDs["3"] {
			t.Error("FindByTags: result should include comic 3")
		}
		if gotIDs["1"] {
			t.Error("FindByTags: result should not include self (comic 1)")
		}
	})

	t.Run("exclude self by cid", func(t *testing.T) {
		tags := []Tag{{ID: 1, Name: "action", Type: "genre"}}
		results, err := ms.FindByTags(ctx, tags, "genre", 2, 10)
		if err != nil {
			t.Fatalf("FindByTags failed: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("FindByTags with cid=2: got %d results, want 1", len(results))
		}
		if len(results) > 0 && results[0].GetID() != "1" {
			t.Errorf("FindByTags with cid=2: expected comic 1, got %s", results[0].GetID())
		}
	})

	t.Run("tagType filter", func(t *testing.T) {
		// tagType 过滤输入标签：传入多个不同类型的标签，tagType="artist" 只收集 artist 类型的 ID
		// 所有漫画都没有 artist 类型的标签 ID=7，因此不应匹配
		tags := []Tag{
			{ID: 7, Name: "mangaka", Type: "artist"},
			{ID: 1, Name: "action", Type: "genre"},
		}
		results, err := ms.FindByTags(ctx, tags, "artist", 0, 10)
		if err != nil {
			t.Fatalf("FindByTags failed: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("FindByTags with unmatched tagType: got %d results, want 0", len(results))
		}
	})

	t.Run("empty tags returns empty slice", func(t *testing.T) {
		results, err := ms.FindByTags(ctx, []Tag{}, "", 0, 10)
		if err != nil {
			t.Fatalf("FindByTags failed: %v", err)
		}
		if results == nil {
			t.Error("FindByTags with empty tags: results should be empty slice, not nil")
		}
		if len(results) != 0 {
			t.Errorf("FindByTags with empty tags: got %d results, want 0", len(results))
		}
	})

	t.Run("limit truncation", func(t *testing.T) {
		tags := []Tag{{ID: 1, Name: "action", Type: "genre"}}
		results, err := ms.FindByTags(ctx, tags, "", 0, 1)
		if err != nil {
			t.Fatalf("FindByTags failed: %v", err)
		}
		if len(results) > 1 {
			t.Errorf("FindByTags with limit=1: got %d results, want at most 1", len(results))
		}
	})
}

// TestMemoryStorage_ArchiveNonComicImplType 验证 ArchiveByID 不再类型锁定 *ComicImpl（I10 回归）。
// 存储中放入非 *ComicImpl 的 Comic 实现时，归档应成功并反映到 GetArchivePath/NotArchived 过滤。
func TestMemoryStorage_ArchiveNonComicImplType(t *testing.T) {
	ctx := t.Context()
	ms := NewMemoryStorage()

	// 用一个非 *ComicImpl 的 Comic 实现（模拟 E2E 中存储 *internalComic.Comic 的场景）
	nonImpl := &fakeComic{id: "9", title: "NotImpl"}
	if err := ms.Save(ctx, nonImpl); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if err := ms.ArchiveByID(ctx, "9"); err != nil {
		t.Fatalf("ArchiveByID should not require *ComicImpl, got: %v", err)
	}

	// NotArchived 过滤应排除已归档的 9
	notArchived, err := ms.Find(ctx, NewComicFilter().SetNotArchived(true))
	if err != nil {
		t.Fatalf("Find notArchived failed: %v", err)
	}
	for _, c := range notArchived {
		if c.GetID() == "9" {
			t.Error("NotArchived filter should exclude archived comic 9")
		}
	}

	if err := ms.RestoreByID(ctx, "9"); err != nil {
		t.Fatalf("RestoreByID should not require *ComicImpl, got: %v", err)
	}
}

// fakeComic 是一个非 *ComicImpl 的 Comic 实现，用于验证存储层不依赖具体类型。
type fakeComic struct {
	id    string
	title string
}

func (f *fakeComic) GetID() string                 { return f.id }
func (f *fakeComic) GetTitle() string              { return f.title }
func (f *fakeComic) GetTitleEnglish() string       { return "" }
func (f *fakeComic) GetTitleJapanese() string      { return "" }
func (f *fakeComic) GetTitlePretty() string        { return "" }
func (f *fakeComic) GetImages() []Image            { return nil }
func (f *fakeComic) GetTags() []Tag                { return nil }
func (f *fakeComic) Object() any                   { return f }
func (f *fakeComic) GetArchivePath() string        { return "" }
func (f *fakeComic) IsValid() bool                 { return false }
func (f *fakeComic) IsStatus() bool                { return true }
func (f *fakeComic) IsDeleted() bool               { return false }
func (f *fakeComic) GetRedirectCID() int           { return 0 }
func (f *fakeComic) GetInvalidCount() int32        { return 0 }
func (f *fakeComic) GetFixedCount() int32          { return 0 }
func (f *fakeComic) GetLastVerify() time.Time      { return time.Time{} }
func (f *fakeComic) SetVerifyResult(*VerifyResult) {}
func (f *fakeComic) MarshalJSON() ([]byte, error)  { return nil, nil }
func (f *fakeComic) UnmarshalJSON([]byte) error    { return nil }
func (f *fakeComic) SetArchivePath(string)         {}

// TestMemoryStorage_GetReturnsCopy 回归 R21-C1：Get 必须返回深拷贝，
// 对返回对象在锁外的修改（如 SetVerifyResult）不得泄漏到存储中的原对象。
func TestMemoryStorage_GetReturnsCopy(t *testing.T) {
	ctx := t.Context()
	ms := NewMemoryStorage()
	c := NewComic("1", "Original", []Image{{ID: "1", Path: "p1.jpg"}})
	if err := ms.Save(ctx, c); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	got, err := ms.Get(ctx, "1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got == c {
		t.Fatal("Get returned the live pointer; want a copy")
	}

	// 修改副本不应影响存储中的原对象。
	got.(*ComicImpl).Title = "Mutated"
	got.(*ComicImpl).Images[0].Path = "mutated.jpg"

	again, err := ms.Get(ctx, "1")
	if err != nil {
		t.Fatalf("second Get failed: %v", err)
	}
	if again.GetTitle() != "Original" {
		t.Errorf("mutating a Get copy leaked into storage: title = %q", again.GetTitle())
	}
	if again.GetImages()[0].Path != "p1.jpg" {
		t.Errorf("mutating a Get copy leaked into storage: image path = %q", again.GetImages()[0].Path)
	}
}

// TestMemoryStorage_FindReturnsCopies 回归 R21-C1：Find 必须返回深拷贝，
// 不得把存储中的活指针暴露给调用方。
func TestMemoryStorage_FindReturnsCopies(t *testing.T) {
	ctx := t.Context()
	ms := NewMemoryStorage()
	c := NewComic("1", "Original", nil)
	if err := ms.Save(ctx, c); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	results, err := ms.Find(ctx, nil)
	if err != nil {
		t.Fatalf("Find failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Find len = %d, want 1", len(results))
	}
	if results[0] == c {
		t.Fatal("Find returned the live pointer; want a copy")
	}
}

// TestMemoryStorage_ConcurrentGetAndSaveVerifyResult 回归 R21-C1：
// Get→MarshalJSON 读与 SaveVerifyResult 写并发时不应有数据竞争
// （Get 返回副本后两者操作不同对象）。
func TestMemoryStorage_ConcurrentGetAndSaveVerifyResult(t *testing.T) {
	ctx := t.Context()
	ms := NewMemoryStorage()
	if err := ms.Save(ctx, NewComic("1", "Original", nil)); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			got, err := ms.Get(ctx, "1")
			if err == nil {
				_, _ = json.Marshal(got) // 模拟 API 序列化读
			}
		}()
		go func() {
			defer wg.Done()
			_ = ms.SaveVerifyResult(ctx, &VerifyResult{
				ComicID:   "1",
				Valid:     true,
				Timestamp: time.Now(),
			})
		}()
	}
	wg.Wait()
}

// TestMemoryStorage_ListTags_Sort 验证 ListTags 排序语义：
// sortType==0（SortTypeByName）按名称升序，其余按 count 降序。
func TestMemoryStorage_ListTags_Sort(t *testing.T) {
	ctx := t.Context()
	ms := NewMemoryStorage()

	comics := []*ComicImpl{
		{ID: "1", Title: "C1", Tags: []Tag{{ID: 1, Name: "alpha", Type: "tag"}, {ID: 2, Name: "zeta", Type: "tag"}}},
		{ID: "2", Title: "C2", Tags: []Tag{{ID: 1, Name: "alpha", Type: "tag"}}},
	}
	for _, c := range comics {
		if err := ms.Save(ctx, c); err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

	// sortType=0 → 按名称升序：alpha 在 zeta 前
	byName, _, err := ms.ListTags(ctx, "tag", 0, 0, 100, false)
	if err != nil {
		t.Fatalf("ListTags sort=0 failed: %v", err)
	}
	if len(byName) != 2 {
		t.Fatalf("ListTags len = %d, want 2", len(byName))
	}
	if byName[0].Name != "alpha" || byName[1].Name != "zeta" {
		t.Errorf("ListTags sort=0 order = [%s, %s], want [alpha, zeta]", byName[0].Name, byName[1].Name)
	}

	// sortType=1 → 按 count 降序：alpha(count=2) 在 zeta(count=1) 前
	byCount, _, err := ms.ListTags(ctx, "tag", 1, 0, 100, false)
	if err != nil {
		t.Fatalf("ListTags sort=1 failed: %v", err)
	}
	if len(byCount) != 2 {
		t.Fatalf("ListTags len = %d, want 2", len(byCount))
	}
	if byCount[0].Name != "alpha" || byCount[1].Name != "zeta" {
		t.Errorf("ListTags sort=1 order = [%s, %s], want [alpha, zeta]", byCount[0].Name, byCount[1].Name)
	}
}

// TestMemoryStorage_FindTotal_IgnorePagination 验证 S6 回归：
// FindTotal 返回真实总数，不受 Limit/Skip 影响（即使 filter 带分页）。
func TestMemoryStorage_FindTotal_IgnorePagination(t *testing.T) {
	ms := NewMemoryStorage()
	for i := 1; i <= 10; i++ {
		id := fmt.Sprintf("%d", i)
		if err := ms.Save(t.Context(), NewComic(id, "C"+id, nil)); err != nil {
			t.Fatal(err)
		}
	}

	// 带分页的 filter：Limit=3, Skip=4，FindTotal 应仍返回 10（真实总数）
	filter := NewComicFilter().SetLimit(3).SetSkip(4)
	total, err := ms.FindTotal(t.Context(), filter)
	if err != nil {
		t.Fatalf("FindTotal failed: %v", err)
	}
	if total != 10 {
		t.Errorf("FindTotal with pagination = %d, want 10 (真实总数不受分页影响)", total)
	}

	// 确认 Find（分页生效）与 FindTotal（不分页）确实不同
	results, err := ms.Find(t.Context(), filter)
	if err != nil {
		t.Fatalf("Find failed: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("Find with Limit=3 = %d results, want 3", len(results))
	}
}

// TestMemoryStorage_Find_FilterID 回归 Infra 8/Infra 9：MemoryStorage.Find
// 必须支持 filter.ID 精确匹配（此前只实现范围过滤，ID 字段被忽略）。
func TestMemoryStorage_Find_FilterID(t *testing.T) {
	ctx := t.Context()
	ms := NewMemoryStorage()
	for i := 1; i <= 3; i++ {
		id := fmt.Sprintf("%d", i)
		if err := ms.Save(ctx, NewComic(id, "C"+id, nil)); err != nil {
			t.Fatal(err)
		}
	}

	got, err := ms.Find(ctx, NewComicFilter().SetID("2"))
	if err != nil {
		t.Fatalf("Find filter ID failed: %v", err)
	}
	if len(got) != 1 || got[0].GetID() != "2" {
		t.Errorf("Find filter ID = %d results (%v), want exactly [2]", len(got), idsOf(got))
	}

	// 不存在的 ID → 空结果
	got, err = ms.Find(ctx, NewComicFilter().SetID("999"))
	if err != nil {
		t.Fatalf("Find filter ID 999 failed: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Find filter ID 999 = %d results, want 0", len(got))
	}
}

// idsOf 提取 []Comic 的 ID 集合（用于断言）
func idsOf(comics []Comic) []string {
	ds := make([]string, len(comics))
	for i, c := range comics {
		ds[i] = c.GetID()
	}
	return ds
}

// TestMemoryStorage_FindByTags_ReturnsCopies 回归：FindByTags 必须返回深拷贝，
// 不得把存储中的活指针暴露给调用方（与 Find/Get 拷贝语义一致）。
func TestMemoryStorage_FindByTags_ReturnsCopies(t *testing.T) {
	ctx := t.Context()
	ms := NewMemoryStorage()

	src := &ComicImpl{
		ID:    "1",
		Title: "Src",
		Tags:  []Tag{{ID: 1, Name: "action", Type: "genre"}},
	}
	dst := &ComicImpl{
		ID:    "2",
		Title: "Dst",
		Tags:  []Tag{{ID: 1, Name: "action", Type: "genre"}},
	}
	if err := ms.Save(ctx, src); err != nil {
		t.Fatal(err)
	}
	if err := ms.Save(ctx, dst); err != nil {
		t.Fatal(err)
	}

	results, err := ms.FindByTags(ctx, src.GetTags(), "genre", 1, 10)
	if err != nil {
		t.Fatalf("FindByTags failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("FindByTags len = %d, want 1", len(results))
	}
	if results[0] == dst {
		t.Fatal("FindByTags returned live pointer; want a copy")
	}
}
