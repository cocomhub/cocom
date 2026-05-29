// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"fmt"

	"github.com/cocomhub/cocom/pkg/mongowrap"
	"github.com/cocomhub/cocom/pkg/storage"
)

var indexFactories = map[string]func(IndexConfig) IndexStore{}

func RegisterIndexStoreFactory(typ string, f func(IndexConfig) IndexStore) {
	indexFactories[typ] = f
}

func init() {
	RegisterIndexStoreFactory("", func(cfg IndexConfig) IndexStore {
		return NewMemoryIndexStore()
	})
	RegisterIndexStoreFactory("memory", func(cfg IndexConfig) IndexStore {
		return NewMemoryIndexStore()
	})
	RegisterIndexStoreFactory("file", func(cfg IndexConfig) IndexStore {
		fs, ok := storage.Get(cfg.FileStoreName)
		if !ok || fs == nil {
			panic(fmt.Errorf("index file store %q not found", cfg.FileStoreName))
		}
		return NewIndexStoreFS(fs, cfg.FileStorePrefix)
	})
	RegisterIndexStoreFactory("mongo", func(cfg IndexConfig) IndexStore {
		db, err := mongowrap.DB(cfg.GetMongoDatabase("archiveManager"))
		if err != nil {
			panic(fmt.Errorf("mongo db %q unavailable: %w", cfg.GetMongoDatabase("archiveManager"), err))
		}
		return NewMongoIndexStore(
			db.Collection(cfg.GetMongoCollection("archiveInfo")),
			WithMongoPrefix(cfg.MongoPrefix),
			WithMongoIDField(cfg.MongoIDField),
			WithMongoNameField(cfg.MongoNameField),
		)
	})
	RegisterIndexStoreFactory("mongo-cocom", func(cfg IndexConfig) IndexStore {
		db, err := mongowrap.DB(cfg.GetMongoDatabase("cocom"))
		if err != nil {
			panic(fmt.Errorf("mongo db %q unavailable: %w", cfg.GetMongoDatabase("cocom"), err))
		}
		return NewComicInfoArchiveIndexStore(db.Collection(cfg.GetMongoCollection("archiveInfo")))
	})
	RegisterIndexStoreFactory("mongo-comicInfo", func(cfg IndexConfig) IndexStore {
		db, err := mongowrap.DB(cfg.GetMongoDatabase("cocom"))
		if err != nil {
			panic(fmt.Errorf("mongo db %q unavailable: %w", cfg.GetMongoDatabase("cocom"), err))
		}
		return NewComicInfoArchiveIndexStore(db.Collection(cfg.GetMongoCollection("comicInfo")))
	})
}
