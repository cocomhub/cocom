// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package localfs

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cocomhub/cocom/pkg/storage"
)

func TestLocalFSBasic(t *testing.T) {
	dir := t.TempDir()
	fs := New("test", dir)
	ctx := context.Background()
	data := []byte("hello storage")
	_, err := fs.Put(ctx, "a/b.txt", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Put error: %v", err)
	}
	rc, meta, err := fs.Get(ctx, "a/b.txt")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	got, _ := io.ReadAll(rc)
	_ = rc.Close()
	if string(got) != string(data) {
		t.Fatalf("data mismatch: %q vs %q", got, data)
	}
	if meta.Size != int64(len(data)) {
		t.Fatalf("size mismatch: %d", meta.Size)
	}
	list, err := fs.List(ctx, "a")
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(list) != 1 || list[0].Key != storage.MustPath("a/b.txt") {
		t.Fatalf("list unexpected: %+v", list)
	}
	if err := fs.Delete(ctx, "a/b.txt"); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "a/b.txt")); !os.IsNotExist(err) {
		t.Fatalf("file still exists")
	}
}

func TestLocalFSTraversalBlocked(t *testing.T) {
	dir := t.TempDir()
	fs := New("sandbox", dir)
	ctx := context.Background()

	if meta, err := fs.Put(ctx, "../x.txt", bytes.NewReader([]byte("x"))); err == nil {
		t.Fatalf("expected error on traversal put")
	} else {
		t.Logf("traversal put meta: %s, err: %v", meta, err)
	}

	if meta, err := fs.Stat(ctx, "../x.txt"); err == nil {
		t.Fatalf("expected error on traversal stat")
	} else {
		t.Logf("traversal stat meta: %s, err: %v", meta, err)
	}

	if rc, meta, err := fs.Get(ctx, "../x.txt"); err == nil {
		rc.Close()
		t.Fatalf("expected error on traversal get")
	} else {
		t.Logf("traversal get meta: %s, err: %v", meta, err)
	}

	if entries, err := fs.List(ctx, "../"); err == nil {
		t.Fatalf("expected error on traversal list")
	} else {
		t.Logf("traversal list: %s, err: %v", entries, err)
	}

	if meta, err := fs.Put(ctx, "safe/ok.txt", bytes.NewReader([]byte("ok"))); err != nil {
		t.Fatalf("unexpected error on safe put: %v", err)
	} else {
		t.Logf("safe put meta: %s", meta)
	}

	if entries, err := fs.List(ctx, "safe"); err != nil {
		t.Fatalf("list error: %v", err)
	} else {
		t.Logf("safe list: %s", entries)
		found := false
		for _, e := range entries {
			if strings.HasSuffix(filepath.ToSlash(e.Key), "safe/ok.txt") {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("safe object not listed")
		}
	}

	if err := fs.Delete(ctx, "safe/ok.txt"); err != nil {
		t.Fatalf("delete error: %v", err)
	} else {
		if meta, err := fs.Stat(ctx, "safe/ok.txt"); err == nil {
			t.Fatalf("expected error on safe stat, got %+v", meta)
		} else {
			t.Logf("safe delete meta: %s, err: %v", meta, err)
		}
	}
}
