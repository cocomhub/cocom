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
	collection *mongo.Collection
	onceInit   sync.Once
)

func init() {
	viper.SetDefault("comic.mongo.database", "cocom")
	viper.SetDefault("comic.mongo.collection", "comicInfo")
}

func Collection() *mongo.Collection {
	onceInit.Do(func() {
		collection = mongowrap.DB(viper.GetString("comic.mongo.database")).
			Collection(viper.GetString("comic.mongo.collection"))
	})
	return collection
}
