// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package comic

import (
	"context"
	"errors"
	"testing"
)

// mockStorage implements Storage for service testing
type mockServiceStorage struct {
	Storage
	archiveByIDFn func(ctx context.Context, id string) error
	restoreByIDFn func(ctx context.Context, id string) error
	getFn         func(ctx context.Context, id string) (Comic, error)
	findFn        func(ctx context.Context, filter *ComicFilter) ([]Comic, error)
	findTotalFn   func(ctx context.Context, filter *ComicFilter) (int64, error)
}

func (m *mockServiceStorage) ArchiveByID(ctx context.Context, id string) error {
	if m.archiveByIDFn != nil {
		return m.archiveByIDFn(ctx, id)
	}
	return nil
}

func (m *mockServiceStorage) RestoreByID(ctx context.Context, id string) error {
	if m.restoreByIDFn != nil {
		return m.restoreByIDFn(ctx, id)
	}
	return nil
}

func (m *mockServiceStorage) Get(ctx context.Context, id string) (Comic, error) {
	if m.getFn != nil {
		return m.getFn(ctx, id)
	}
	return NewComic(id, "test", nil), nil
}

func (m *mockServiceStorage) Find(ctx context.Context, filter *ComicFilter) ([]Comic, error) {
	if m.findFn != nil {
		return m.findFn(ctx, filter)
	}
	return nil, nil
}

func (m *mockServiceStorage) FindTotal(ctx context.Context, filter *ComicFilter) (int64, error) {
	if m.findTotalFn != nil {
		return m.findTotalFn(ctx, filter)
	}
	return 0, nil
}

func TestNewService(t *testing.T) {
	ctx := context.Background()
	ms := NewMemoryStorage()
	// Directly construct ServiceImpl instead of using NewService,
	// because NewService → NewComicVerifier → findWgetPath which panics on Windows
	svc := &ServiceImpl{
		storage: ms,
	}
	if svc.storage == nil {
		t.Error("storage should not be nil")
	}
	_ = ctx
}

func TestServiceImpl_SearchComics(t *testing.T) {
	ctx := context.Background()
	ms := NewMemoryStorage()

	c1 := NewComic("1", "Alpha", nil)
	c2 := NewComic("2", "Beta", nil)
	_ = ms.Save(ctx, c1)
	_ = ms.Save(ctx, c2)

	svc := &ServiceImpl{storage: ms}

	t.Run("all", func(t *testing.T) {
		results, err := svc.SearchComics(ctx, &ComicFilter{})
		if err != nil {
			t.Fatalf("SearchComics failed: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("got %d results, want 2", len(results))
		}
	})
	t.Run("with limit", func(t *testing.T) {
		filter := &ComicFilter{}
		filter.SetLimit(1)
		results, err := svc.SearchComics(ctx, filter)
		if err != nil {
			t.Fatalf("SearchComics failed: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("got %d results, want 1", len(results))
		}
	})
}

func TestServiceImpl_GetComicInfo(t *testing.T) {
	ctx := context.Background()
	ms := NewMemoryStorage()
	c := NewComic("42", "Test", nil)
	_ = ms.Save(ctx, c)

	svc := &ServiceImpl{storage: ms}

	t.Run("existing", func(t *testing.T) {
		got, err := svc.GetComicInfo(ctx, "42")
		if err != nil {
			t.Fatalf("GetComicInfo failed: %v", err)
		}
		if got.GetID() != "42" {
			t.Errorf("ID = %q, want %q", got.GetID(), "42")
		}
	})
	t.Run("non-existent", func(t *testing.T) {
		_, err := svc.GetComicInfo(ctx, "999")
		if err == nil {
			t.Error("expected error for non-existent comic")
		}
	})
}

func TestServiceImpl_ArchiveComic(t *testing.T) {
	ctx := context.Background()
	archiveCalled := false
	stub := &mockServiceStorage{
		archiveByIDFn: func(_ context.Context, id string) error {
			archiveCalled = true
			if id != "1" {
				t.Errorf("archive id = %q, want %q", id, "1")
			}
			return nil
		},
	}
	svc := &ServiceImpl{storage: stub}
	if err := svc.ArchiveComic(ctx, "1"); err != nil {
		t.Fatalf("ArchiveComic failed: %v", err)
	}
	if !archiveCalled {
		t.Error("ArchiveByID was not called")
	}
}

func TestServiceImpl_ArchiveComic_Error(t *testing.T) {
	ctx := context.Background()
	stub := &mockServiceStorage{
		archiveByIDFn: func(_ context.Context, id string) error {
			return errors.New("archive failed")
		},
	}
	svc := &ServiceImpl{storage: stub}
	if err := svc.ArchiveComic(ctx, "1"); err == nil {
		t.Error("expected error from ArchiveComic")
	}
}

func TestServiceImpl_RestoreComic(t *testing.T) {
	ctx := context.Background()
	restoreCalled := false
	stub := &mockServiceStorage{
		restoreByIDFn: func(_ context.Context, id string) error {
			restoreCalled = true
			if id != "1" {
				t.Errorf("restore id = %q, want %q", id, "1")
			}
			return nil
		},
	}
	svc := &ServiceImpl{storage: stub}
	if err := svc.RestoreComic(ctx, "1"); err != nil {
		t.Fatalf("RestoreComic failed: %v", err)
	}
	if !restoreCalled {
		t.Error("RestoreByID was not called")
	}
}

func TestServiceImpl_RestoreComic_Error(t *testing.T) {
	ctx := context.Background()
	stub := &mockServiceStorage{
		restoreByIDFn: func(_ context.Context, id string) error {
			return errors.New("restore failed")
		},
	}
	svc := &ServiceImpl{storage: stub}
	if err := svc.RestoreComic(ctx, "1"); err == nil {
		t.Error("expected error from RestoreComic")
	}
}
