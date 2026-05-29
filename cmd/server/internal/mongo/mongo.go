// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package mongo

import (
	"fmt"
	"sync"

	"github.com/cocomhub/cocom/pkg/mongowrap"
	"github.com/spf13/viper"

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
)

func init() {
	viper.SetDefault("comic.mongo.database", "cocom")
	viper.SetDefault("comic.mongo.collections.comicInfo", "comicInfo")
	viper.SetDefault("comic.mongo.collections.oneComicInfo", "oneComicInfo")
	viper.SetDefault("comic.mongo.collections.videoInfo", "videoInfo")
	viper.SetDefault("comic.mongo.collections.settings", "settings")
	viper.SetDefault("comic.mongo.collections.custom", "custom")
	viper.SetDefault("comic.mongo.collections.comicTag", "comicTag")
}

func DB() *mongo.Database {
	initDB.Do(func() {
		var err error
		db, err = mongowrap.DB(viper.GetString("comic.mongo.database"))
		if err != nil {
			panic(fmt.Errorf("failed to get mongo db: %w", err))
		}
	})
	return db
}

func ComicInfo() *mongo.Collection {
	initComicInfo.Do(func() {
		comicInfo = DB().Collection(viper.GetString("comic.mongo.collections.comicInfo"))
	})
	return comicInfo
}

func OneComicInfo() *mongo.Collection {
	initOneComicInfo.Do(func() {
		oneComicInfo = DB().Collection(viper.GetString("comic.mongo.collections.oneComicInfo"))
	})
	return oneComicInfo
}

func VideoInfo() *mongo.Collection {
	initVideoInfo.Do(func() {
		videoInfo = DB().Collection(viper.GetString("comic.mongo.collections.videoInfo"))
	})
	return videoInfo
}

func Settings() *mongo.Collection {
	initSettings.Do(func() {
		settings = DB().Collection(viper.GetString("comic.mongo.collections.settings"))
	})
	return settings
}

func Custom() *mongo.Collection {
	initCustom.Do(func() {
		custom = DB().Collection(viper.GetString("comic.mongo.collections.custom"))
	})
	return custom
}

func ComicTag() *mongo.Collection {
	initComicTag.Do(func() {
		comicTag = DB().Collection(viper.GetString("comic.mongo.collections.comicTag"))
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
