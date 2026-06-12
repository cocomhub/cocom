// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package comic

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"
)

func TestMemoryStorage_SaveAndGet(t *testing.T) {
	ctx := context.Background()
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
	ctx := context.Background()
	ms := NewMemoryStorage()

	c1 := NewComic("1", "First", nil)
	if err := ms.Save(ctx, c1); err != nil {
		t.Fatal(err)
	}

	c2 := NewComic("1", "Second", nil)
	if err := ms.Save(ctx, c2); err != nil {
		t.Fatal(err)
	}

	got, _ := ms.Get(ctx, "1")
	if got.GetTitle() != "Second" {
		t.Errorf("Save duplicate should overwrite, got title %q", got.GetTitle())
	}
}

func TestMemoryStorage_Update(t *testing.T) {
	ctx := context.Background()
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

	got, _ := ms.Get(ctx, "1")
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
	ctx := context.Background()
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

	got, _ := ms.Get(ctx, "1")
	if got.GetTitle() != "ViaMap" {
		t.Errorf("Update via map: title = %q, want %q", got.GetTitle(), "ViaMap")
	}
}

func TestMemoryStorage_Delete(t *testing.T) {
	ctx := context.Background()
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
	ctx := context.Background()
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
		if err := ms.Save(ctx, NewComic(id, "Comic "+id, nil)); err != nil {
			t.Fatal(err)
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
	ctx := context.Background()
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
		if err := ms.Save(ctx, NewComic(id, "C"+id, nil)); err != nil {
			t.Fatal(err)
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
	ctx := context.Background()
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
	ctx := context.Background()
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
	if err := ms.RestoreByID(ctx, "1"); err != nil {
		t.Errorf("RestoreByID of unarchived comic should not error, got %v", err)
	}

	// Restore non-existent returns error
	err = ms.RestoreByID(ctx, "999")
	if err == nil {
		t.Error("RestoreByID of non-existent comic should return error")
	}
}

func TestMemoryStorage_FindWithFilter(t *testing.T) {
	ctx := context.Background()
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
	ctx := context.Background()
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

	got, _ := ms.Get(ctx, "1")
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
	ctx := context.Background()
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
	ctx := context.Background()
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
	ctx := context.Background()
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
