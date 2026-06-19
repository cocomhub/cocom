// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/cache"
	internalComic "github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/cmd/server/internal/custom"
	"github.com/cocomhub/cocom/cmd/server/internal/onecomic"
	"github.com/cocomhub/cocom/cmd/server/internal/tag"
	"github.com/cocomhub/cocom/cmd/server/internal/video"
	"github.com/cocomhub/cocom/pkg/comic"
)

var (
	testMemStorage    *comic.MemoryStorage
	testTagLikeStore  *tag.MemoryLikeStore
	testTagStore      *tag.MemoryTagStore
	testVideoStore    *video.MemoryVideoStore
	testOneComicStore *onecomic.MemoryOneComicStore
	testCustomStore   *custom.MemoryCustomStore
	testRelationStore *tag.MemoryRelationStore
)

func TestMain(m *testing.M) {
	// ---- MemoryStorage setup for comic storage tests ----
	testMemStorage = comic.NewMemoryStorage()
	internalComic.SetDefaultStorage(testMemStorage)

	// ---- Tag stores setup ----
	testTagLikeStore = tag.NewMemoryLikeStore()
	tag.SetDefaultLikeStore(testTagLikeStore)
	tag.SetDefaultComicStore(testMemStorage)

	testTagStore = tag.NewMemoryTagStore()
	tag.SetDefaultTagStore(testTagStore)
	testRelationStore = tag.NewMemoryRelationStore()
	tag.SetDefaultRelationStore(testRelationStore)

	// ---- Video store setup ----
	testVideoStore = video.NewMemoryVideoStore()
	video.SetDefaultVideoStore(testVideoStore)

	// ---- OneComic store setup ----
	testOneComicStore = onecomic.NewMemoryOneComicStore()
	onecomic.SetDefaultOneComicStore(testOneComicStore)

	// ---- Custom store setup ----
	testCustomStore = custom.NewMemoryCustomStore()
	custom.SetDefaultCustomStore(testCustomStore)

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
		{
			CID: 1004,
			Title: struct {
				English  string `json:"english,omitempty" bson:"english"`
				Japanese string `json:"japanese,omitempty" bson:"japanese"`
				Pretty   string `json:"pretty,omitempty" bson:"pretty"`
			}{Pretty: "Love Story", English: "Love Story"},
			Tags: api.Tags{
				{ID: 5, Name: "romance", Type: "tag"},
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
		cache.Init(context.Background(), 10*time.Minute, 1*time.Minute)
	}()

	os.Exit(m.Run())
}
