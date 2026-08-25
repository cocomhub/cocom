// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"fmt"
	"sync"

	"github.com/cocomhub/cocom/pkg/mongowrap"
	"github.com/cocomhub/cocom/pkg/storage"
)

var (
	indexFactoriesMu sync.RWMutex
	indexFactories   = map[string]func(IndexConfig) (IndexStore, error){}
)

func RegisterIndexStoreFactory(typ string, f func(IndexConfig) (IndexStore, error)) {
	indexFactoriesMu.Lock()
	defer indexFactoriesMu.Unlock()
	indexFactories[typ] = f
}

func init() {
	RegisterIndexStoreFactory("", func(cfg IndexConfig) (IndexStore, error) {
		return NewMemoryIndexStore(), nil
	})
	RegisterIndexStoreFactory("memory", func(cfg IndexConfig) (IndexStore, error) {
		return NewMemoryIndexStore(), nil
	})
	RegisterIndexStoreFactory("file", func(cfg IndexConfig) (IndexStore, error) {
		fs, ok := storage.Get(cfg.FileStoreName)
		if !ok || fs == nil {
			return nil, fmt.Errorf("index file store %q not found", cfg.FileStoreName)
		}
		return NewIndexStoreFS(fs, cfg.FileStorePrefix), nil
	})
	RegisterIndexStoreFactory("mongo", func(cfg IndexConfig) (IndexStore, error) {
		db, err := mongowrap.DB(cfg.GetMongoDatabase("archiveManager"))
		if err != nil {
			return nil, fmt.Errorf("mongo db %q unavailable: %w", cfg.GetMongoDatabase("archiveManager"), err)
		}
		return NewMongoIndexStore(
			db.Collection(cfg.GetMongoCollection("archiveInfo")),
			WithMongoPrefix(cfg.MongoPrefix),
			WithMongoIDField(cfg.MongoIDField),
			WithMongoNameField(cfg.MongoNameField),
		), nil
	})
	RegisterIndexStoreFactory("mongo-cocom", func(cfg IndexConfig) (IndexStore, error) {
		db, err := mongowrap.DB(cfg.GetMongoDatabase("cocom"))
		if err != nil {
			return nil, fmt.Errorf("mongo db %q unavailable: %w", cfg.GetMongoDatabase("cocom"), err)
		}
		return NewComicInfoArchiveIndexStore(db.Collection(cfg.GetMongoCollection("archiveInfo"))), nil
	})
	RegisterIndexStoreFactory("mongo-comicInfo", func(cfg IndexConfig) (IndexStore, error) {
		db, err := mongowrap.DB(cfg.GetMongoDatabase("cocom"))
		if err != nil {
			return nil, fmt.Errorf("mongo db %q unavailable: %w", cfg.GetMongoDatabase("cocom"), err)
		}
		return NewComicInfoArchiveIndexStore(db.Collection(cfg.GetMongoCollection("comicInfo"))), nil
	})
}
