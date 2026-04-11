// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/cocomhub/cocom/pkg/storage"
	"github.com/cocomhub/cocom/pkg/storage/localfs"
)

func TestMigrateLocalToLocal(t *testing.T) {
	ctx := context.Background()
	src := localfs.New("src", t.TempDir())
	dst := localfs.New("dst", t.TempDir())

	data := []byte("migrate me")
	if _, err := src.Put(ctx, "x/y.txt", bytes.NewReader(data)); err != nil {
		t.Fatalf("src put: %v", err)
	}

	res, err := storage.Migrate(ctx, src, dst, []string{"x/y.txt"})
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if len(res.Success) != 1 {
		t.Fatalf("expect success 1, got %d", len(res.Success))
	}
	rc, _, err := dst.Get(ctx, "x/y.txt")
	if err != nil {
		t.Fatalf("dst get: %v", err)
	}
	defer rc.Close()
	got, _ := io.ReadAll(rc)
	if string(got) != string(data) {
		t.Fatalf("mismatch: %q vs %q", got, data)
	}
}
