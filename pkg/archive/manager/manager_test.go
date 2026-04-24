// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cocomhub/cocom/pkg/archive"
	"github.com/cocomhub/cocom/pkg/storage"
	"github.com/cocomhub/cocom/pkg/storage/localfs"
	"github.com/cocomhub/cocom/pkg/util"
)

func TestIndexStoreFS_CRUDAndList(t *testing.T) {
	root := t.TempDir()
	st := localfs.New("archive-index", root)
	store := NewIndexStoreFS(st, "index")
	ctx := context.Background()

	m := &ArchiveMeta{ID: 1, Name: "a", Path: filepath.Join(root, "a.7z"), Size: 10, ModTime: time.Now(), Version: 1, Type: archive.TypeSingle}
	if err := store.Create(ctx, m); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := store.Create(ctx, m); err != ErrAlreadyExists {
		t.Fatalf("create: %v", err)
	}
	got, err := store.Get(ctx, 1)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "a" {
		t.Fatalf("name: %s", got.Name)
	}
	m.Name = "b"
	if err := store.Update(ctx, m); err != nil {
		t.Fatalf("update: %v", err)
	}
	list, err := store.List(ctx, IndexFilter{Name: "b"})
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v len=%d", err, len(list))
	}
	if err := store.Delete(ctx, 1); err != nil {
		t.Fatalf("delete: %v", err)
	}
	list, err = store.List(ctx, IndexFilter{})
	if err != nil || len(list) != 0 {
		t.Fatalf("list after delete: %v len=%d", err, len(list))
	}
}

func TestCheckAndUpdate(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.bin")
	if err := os.WriteFile(p, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	md5 := util.MustFileMD5(p)
	mgr := New()
	idx := mgr.(*manager).index
	ctx := context.Background()
	meta := ArchiveMeta{
		ID:      101,
		Name:    "x",
		Path:    p,
		Version: 1,
		Type:    archive.TypeSingle,
		Checksum: storage.Checksum{
			Algorithm: "md5",
			Value:     md5,
		},
	}
	if err := idx.Create(ctx, &meta); err != nil {
		t.Fatalf("create: %v", err)
	}
	rep, err := newHelper(mgr).Check(ctx, 101, false)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !rep.ReplicaHealth.Healthy {
		t.Fatalf("healthy false")
	}
}

func TestReplicateMore_LocalFS(t *testing.T) {
	srcDir := t.TempDir()
	p := filepath.Join(srcDir, "a.7z")
	if err := os.WriteFile(p, []byte("data"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	mgr := New()
	idx := mgr.(*manager).index
	ctx := context.Background()
	meta := &ArchiveMeta{ID: 2001, Name: "a", Path: p, Version: 1, Type: archive.TypeSingle}
	if err := idx.Create(ctx, meta); err != nil {
		t.Fatalf("create: %v", err)
	}
	dstRoot := t.TempDir()
	dst := localfs.New("dstfs", dstRoot)
	metas, err := newHelper(mgr).ReplicateMore(ctx, dst, "rep", IndexFilter{ID: 2001})
	if err != nil || len(metas) != 1 {
		t.Fatalf("replicate: %v len=%d", err, len(metas))
	}
	key := storage.MustPath("rep", filepath.Base(p))
	exists, err := dst.Exists(ctx, key)
	if err != nil || !exists {
		t.Fatalf("exists: %v", err)
	}
	m2, err := idx.Get(ctx, 2001)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	found := false
	for _, loc := range m2.Locators {
		if loc.Backend == "dstfs" && loc.Key == key {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("locator not updated")
	}
}

func TestApplyRetention_LocalFS(t *testing.T) {
	srcDir := t.TempDir()
	p := filepath.Join(srcDir, "b.7z")
	if err := os.WriteFile(p, []byte("data2"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	mgr := New()
	idx := mgr.(*manager).index
	ctx := context.Background()
	meta := &ArchiveMeta{
		ID:            3001,
		Name:          "b",
		Path:          p,
		Version:       1,
		Type:          archive.TypeSingle,
		ReplicaHealth: storage.NewHealthy(true),
		Locators: []storage.StorageLocator{
			{Backend: "dstfs", Key: "rep/b.7z", ReplicaHealth: storage.NewHealthy(true)},
		},
	}
	if err := idx.Create(ctx, meta); err != nil {
		t.Fatalf("create: %v", err)
	}
	n, err := newHelper(mgr).ApplyRetention(ctx, IndexFilter{ID: 3001})
	if err != nil || n != 1 {
		t.Fatalf("retention: %v n=%d", err, n)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatalf("file still exists")
	}
	m2, err := idx.Get(ctx, 3001)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	for _, loc := range m2.Locators {
		if loc.Backend == "localfs" {
			t.Fatalf("localfs locator not removed")
		}
	}
}

func TestReplicateMoreIdempotent(t *testing.T) {
	srcDir := t.TempDir()
	p := filepath.Join(srcDir, "c.7z")
	if err := os.WriteFile(p, []byte("data3"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	mgr := New()
	idx := mgr.(*manager).index
	ctx := context.Background()
	meta := &ArchiveMeta{ID: 4001, Name: "c", Path: p, Version: 1, Type: mgr.Algorithm()}
	if err := idx.Create(ctx, meta); err != nil {
		t.Fatalf("create: %v", err)
	}
	dstRoot := t.TempDir()
	dst := localfs.New("dstfs2", dstRoot)
	// replicate twice
	for range 2 {
		metas, err := newHelper(mgr).ReplicateMore(ctx, dst, "rep2", IndexFilter{ID: 4001})
		if err != nil || len(metas) != 1 {
			t.Fatalf("replicate: %v len=%d", err, len(metas))
		}
	}
	key := storage.MustPath("rep2", filepath.Base(p))
	m2, err := idx.Get(ctx, 4001)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	cnt := 0
	for _, loc := range m2.Locators {
		if loc.Backend == "dstfs2" && loc.Key == key {
			cnt++
		}
	}
	if cnt != 1 {
		t.Fatalf("expected single locator for backend, got=%d", cnt)
	}
}
