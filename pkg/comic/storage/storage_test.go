// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"context"
	"testing"

	"github.com/cocomhub/cocom/pkg/comic"
)

func TestStorage_FindChannelHelper(t *testing.T) {
	ctx := context.Background()
	filter := &comic.ComicFilter{}
	filter.SetLimit(10)

	advance := func(impls []comic.Comic, f *comic.ComicFilter) {
		f.Skip += int64(len(impls))
	}

	ch, err := FindChannelHelper(ctx, filter, func(ctx context.Context, f *comic.ComicFilter) ([]comic.Comic, error) {
		return []comic.Comic{}, nil
	}, advance)
	if err != nil {
		t.Fatalf("FindChannelHelper failed: %v", err)
	}
	if ch == nil {
		t.Error("FindChannelHelper should return a channel")
	}
	// Channel should close immediately (no results)
	for range ch {
	}
	t.Log("FindChannelHelper with empty results completed")
}
