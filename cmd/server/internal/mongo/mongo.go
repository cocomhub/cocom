/*
Copyright © 2023 suixibing <suixibing@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package mongo

import (
	"sync"

	"github.com/spf13/viper"
	"github.com/suixibing/cocom/pkg/mongowrap"

	"go.mongodb.org/mongo-driver/mongo"
)

var (
	db     *mongo.Database
	initDB sync.Once

	comicInfo     *mongo.Collection
	initComicInfo sync.Once

	settings     *mongo.Collection
	initSettings sync.Once

	custom     *mongo.Collection
	initCustom sync.Once
)

func init() {
	viper.SetDefault("comic.mongo.database", "cocom")
	viper.SetDefault("comic.mongo.collections.comicInfo", "comicInfo")
	viper.SetDefault("comic.mongo.collections.settings", "settings")
	viper.SetDefault("comic.mongo.collections.custom", "custom")
}

func DB() *mongo.Database {
	initDB.Do(func() {
		db = mongowrap.DB(viper.GetString("comic.mongo.database"))
	})
	return db
}

func ComicInfo() *mongo.Collection {
	initComicInfo.Do(func() {
		comicInfo = DB().Collection(viper.GetString("comic.mongo.collections.comicInfo"))
	})
	return comicInfo
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

func ComicInfoBuilder() *mongowrap.Builder {
	return mongowrap.NewBuilder(ComicInfo())
}

func ComicInfoSettings() *mongowrap.Builder {
	return mongowrap.NewBuilder(Settings())
}

func ComicInfoCustom() *mongowrap.Builder {
	return mongowrap.NewBuilder(Custom())
}
