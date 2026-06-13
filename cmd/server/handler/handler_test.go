// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/cache"
	internalComic "github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/pkg/comic"
	"github.com/cocomhub/cocom/pkg/mongowrap"
)

var (
	testMongoAvailable bool
	testMemStorage     *comic.MemoryStorage
)

func TestMain(m *testing.M) {
	// ---- MemoryStorage setup for comic storage tests ----
	testMemStorage = comic.NewMemoryStorage()
	internalComic.SetDefaultStorage(testMemStorage)

	// ---- Inject test data ----
	ctx := context.Background()

	testComics := []*api.ComicInfo{
		{
			CID: 1001,
			Title: struct {
				English  string `json:"english,omitempty" bson:"english"`
				Japanese string `json:"japanese,omitempty" bson:"japanese"`
				Pretty   string `json:"pretty,omitempty" bson:"pretty"`
			}{Pretty: "Test Comic 1", English: "Test Comic 1"},
			Tags: api.Tags{
				{ID: 1, Name: "test", Type: "tag"},
				{ID: 2, Name: "artist1", Type: "artist"},
			},
		},
		{
			CID: 1002,
			Title: struct {
				English  string `json:"english,omitempty" bson:"english"`
				Japanese string `json:"japanese,omitempty" bson:"japanese"`
				Pretty   string `json:"pretty,omitempty" bson:"pretty"`
			}{Pretty: "Test Comic 2", English: "Test Comic 2"},
			Tags: api.Tags{
				{ID: 1, Name: "test", Type: "tag"},
				{ID: 3, Name: "artist2", Type: "artist"},
			},
		},
		{
			CID: 1003,
			Title: struct {
				English  string `json:"english,omitempty" bson:"english"`
				Japanese string `json:"japanese,omitempty" bson:"japanese"`
				Pretty   string `json:"pretty,omitempty" bson:"pretty"`
			}{Pretty: "Another Comic", English: "Another Comic"},
			Tags: api.Tags{
				{ID: 4, Name: "char1", Type: "character"},
			},
		},
	}

	for _, info := range testComics {
		c := internalComic.NewComic(info)
		if err := testMemStorage.Save(ctx, c); err != nil {
			slog.Error("failed to save test comic", "cid", info.CID, "err", err)
		}
	}

	// ---- Cache init (may panic if config not loaded) ----
	func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Warn("cache init panicked, continuing without cache", "recover", r)
			}
		}()
		cache.Init(context.Background())
	}()

	// ---- MongoDB init (preserved for tag-related tests that still require it) ----
	if err := mongowrap.Init(); err != nil {
		slog.Warn("MongoDB not available, tag-related tests will be skipped")
	} else {
		testMongoAvailable = true
	}

	os.Exit(m.Run())
}
