// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package localfs

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/cocomhub/cocom/pkg/storage"
)

func TestPutZeroByteAndGet(t *testing.T) {
	root := t.TempDir()
	fs := New("tfs", root)
	ctx := context.Background()

	key := storage.MustPath("empty", "file.bin")
	meta, err := fs.Put(ctx, key, bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if meta.Size != 0 {
		t.Fatalf("size want=0 got=%d", meta.Size)
	}
	rc, m2, err := fs.Get(ctx, key)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	_ = rc.Close()
	if m2.Size != 0 {
		t.Fatalf("get size want=0 got=%d", m2.Size)
	}
}

func TestTraversalBlocked(t *testing.T) {
	root := t.TempDir()
	fs := New("tfs", root)
	ctx := context.Background()

	// Absolute/traversal keys should be blocked
	badKeys := []string{"../x.bin", "/../x.bin"}
	for _, k := range badKeys {
		if _, err := fs.Put(ctx, k, bytes.NewReader([]byte("x"))); err == nil {
			t.Fatalf("expected traversal blocked for key=%q", k)
		}
	}
}

func TestUnicodeAndSpaceFilenames(t *testing.T) {
	root := t.TempDir()
	fs := New("tfs", root)
	ctx := context.Background()

	names := []string{"a b.txt", "中 文 名 称.bin", "mix-空 格-Üñïçødé.dat"}
	for _, n := range names {
		key := storage.MustPath("dir", n)
		if _, err := fs.Put(ctx, key, bytes.NewReader([]byte("data"))); err != nil {
			t.Fatalf("put %q: %v", n, err)
		}
		meta, err := fs.Stat(ctx, key)
		if err != nil {
			t.Fatalf("stat %q: %v", n, err)
		}
		if !strings.HasSuffix(meta.Key, filepath.ToSlash(n)) {
			t.Fatalf("key suffix mismatch: %s", meta.Key)
		}
	}
}

func TestConcurrentPutDelete(t *testing.T) {
	root := t.TempDir()
	fs := New("tfs", root)
	ctx := context.Background()
	key := storage.MustPath("concurrent", "x.bin")

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for range 50 {
			_, _ = fs.Put(ctx, key, bytes.NewReader([]byte("x")), storage.WithOverwrite(true))
		}
	}()
	go func() {
		defer wg.Done()
		for range 50 {
			_ = fs.Delete(ctx, key)
		}
	}()
	wg.Wait()
}
