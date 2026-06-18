// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package mongo

import (
	"fmt"
	"sync"

	"github.com/cocomhub/cocom/internal/config"
	"github.com/cocomhub/cocom/pkg/mongowrap"

	"go.mongodb.org/mongo-driver/mongo"
)

var (
	db     *mongo.Database
	initDB sync.Once

	comicInfo     *mongo.Collection
	initComicInfo sync.Once

	oneComicInfo     *mongo.Collection
	initOneComicInfo sync.Once

	videoInfo     *mongo.Collection
	initVideoInfo sync.Once

	settings     *mongo.Collection
	initSettings sync.Once

	custom     *mongo.Collection
	initCustom sync.Once

	comicTag     *mongo.Collection
	initComicTag sync.Once

	tagRelation     *mongo.Collection
	initTagRelation sync.Once
)

// SetDefault 已迁移到 internal/config/config.go setDefaults()
// 保留空 init() 以保持 import side-effect 兼容。
func init() {}

func DB() *mongo.Database {
	initDB.Do(func() {
		var err error
		db, err = mongowrap.DB(config.Get().Comic.Mongo.Database)
		if err != nil {
			panic(fmt.Errorf("failed to get mongo db: %w", err))
		}
	})
	return db
}

func ComicInfo() *mongo.Collection {
	initComicInfo.Do(func() {
		comicInfo = DB().Collection(config.Get().Comic.Mongo.Collections.ComicInfo)
	})
	return comicInfo
}

func OneComicInfo() *mongo.Collection {
	initOneComicInfo.Do(func() {
		oneComicInfo = DB().Collection(config.Get().Comic.Mongo.Collections.OneComicInfo)
	})
	return oneComicInfo
}

func VideoInfo() *mongo.Collection {
	initVideoInfo.Do(func() {
		videoInfo = DB().Collection(config.Get().Comic.Mongo.Collections.VideoInfo)
	})
	return videoInfo
}

func Settings() *mongo.Collection {
	initSettings.Do(func() {
		settings = DB().Collection(config.Get().Comic.Mongo.Collections.Settings)
	})
	return settings
}

func Custom() *mongo.Collection {
	initCustom.Do(func() {
		custom = DB().Collection(config.Get().Comic.Mongo.Collections.Custom)
	})
	return custom
}

func ComicTag() *mongo.Collection {
	initComicTag.Do(func() {
		comicTag = DB().Collection(config.Get().Comic.Mongo.Collections.ComicTag)
	})
	return comicTag
}

func ComicInfoBuilder() *mongowrap.Builder {
	return mongowrap.NewBuilder(ComicInfo())
}

func OneComicInfoBuilder() *mongowrap.Builder {
	return mongowrap.NewBuilder(OneComicInfo())
}

func VideoInfoBuilder() *mongowrap.Builder {
	return mongowrap.NewBuilder(VideoInfo())
}

func ComicInfoSettings() *mongowrap.Builder {
	return mongowrap.NewBuilder(Settings())
}

func ComicInfoCustom() *mongowrap.Builder {
	return mongowrap.NewBuilder(Custom())
}

func ComicTagBuilder() *mongowrap.Builder {
	return mongowrap.NewBuilder(ComicTag())
}

func TagRelation() *mongo.Collection {
	initTagRelation.Do(func() {
		tagRelation = DB().Collection(config.Get().Comic.Mongo.Collections.TagRelation)
	})
	return tagRelation
}

func TagRelationBuilder() *mongowrap.Builder {
	return mongowrap.NewBuilder(TagRelation())
}
