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
	dbErr  error
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

// SetDefault 已迁移到 internal/config/manager.go setDefaultsOn()

func DB() *mongo.Database {
	initDB.Do(func() {
		var err error
		db, err = mongowrap.DB(config.Get().Comic.Mongo.Database)
		if err != nil {
			// 只记录失败，不在 sync.Once.f 内 panic：sync.Once 的 f panic 会把
			// done 置 1（Go 实现），导致 DB() 永远返回 nil，调用方混淆“DB 未初始化”
			// 与“DB 初始化失败”。失败留在 dbErr，后续滚动取 db==nil 时再统一 panic。
			// 注意：DB() 的每个调用在失败后都会再次 panic 同一错误——这是刻意的、
			// 调用一致的快速失败（而非首竞走后静默 nil），由调用方 Try 确认后选用保留方式。
			dbErr = fmt.Errorf("failed to get mongo db: %w", err)
		}
	})
	if dbErr != nil {
		// Do 外 panic：不烧毁 once，失败在每个调用点稳定复现（快速失败、错误信息一致）。
		panic(dbErr)
	}
	if db == nil {
		panic(fmt.Errorf("mongo: db is nil (DB 初始化异常)"))
	}
	return db
}

func ComicInfo() *mongo.Collection {
	initComicInfo.Do(func() {
		comicInfo = DB().Collection(config.Get().Comic.Mongo.Collections.ComicInfo)
	})
	// 防御：init 失败的调用应直接 panic，而非返回 nil 让下游解引用 panic。
	if comicInfo == nil {
		panic(fmt.Errorf("mongo: ComicInfo collection is nil"))
	}
	return comicInfo
}

func OneComicInfo() *mongo.Collection {
	initOneComicInfo.Do(func() {
		oneComicInfo = DB().Collection(config.Get().Comic.Mongo.Collections.OneComicInfo)
	})
	if oneComicInfo == nil {
		panic(fmt.Errorf("mongo: OneComicInfo collection is nil"))
	}
	return oneComicInfo
}

func VideoInfo() *mongo.Collection {
	initVideoInfo.Do(func() {
		videoInfo = DB().Collection(config.Get().Comic.Mongo.Collections.VideoInfo)
	})
	if videoInfo == nil {
		panic(fmt.Errorf("mongo: VideoInfo collection is nil"))
	}
	return videoInfo
}

func Settings() *mongo.Collection {
	initSettings.Do(func() {
		settings = DB().Collection(config.Get().Comic.Mongo.Collections.Settings)
	})
	if settings == nil {
		panic(fmt.Errorf("mongo: Settings collection is nil"))
	}
	return settings
}

func Custom() *mongo.Collection {
	initCustom.Do(func() {
		custom = DB().Collection(config.Get().Comic.Mongo.Collections.Custom)
	})
	if custom == nil {
		panic(fmt.Errorf("mongo: Custom collection is nil"))
	}
	return custom
}

func ComicTag() *mongo.Collection {
	initComicTag.Do(func() {
		comicTag = DB().Collection(config.Get().Comic.Mongo.Collections.ComicTag)
	})
	if comicTag == nil {
		panic(fmt.Errorf("mongo: ComicTag collection is nil"))
	}
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
	if tagRelation == nil {
		panic(fmt.Errorf("mongo: TagRelation collection is nil"))
	}
	return tagRelation
}

func TagRelationBuilder() *mongowrap.Builder {
	return mongowrap.NewBuilder(TagRelation())
}
