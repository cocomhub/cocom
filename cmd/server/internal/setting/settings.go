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
package setting

import (
	"context"

	"github.com/suixibing/cocom/cmd/server/internal/mongo"
	"github.com/suixibing/cocom/pkg/clog"
	"github.com/suixibing/cocom/pkg/conv"
	"github.com/suixibing/cocom/pkg/mongowrap"

	"go.mongodb.org/mongo-driver/bson"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	SettingKeyType string = "type"
	SettingKeyKey  string = "key"
	SettingKeyVal  string = "val"
)

func GetSettings(ctx context.Context, settingType string, keys ...string) (map[string]interface{}, error) {
	opts := options.Find()
	filter := bson.M{SettingKeyType: settingType}

	if len(keys) > 0 && keys[0] != "" {
		filter[SettingKeyKey] = bson.M{"$in": keys}
	}
	clog.Debugf(ctx, "GetSettings filters[%s]", conv.JSON(filter))

	cursor, err := mongo.Settings().Find(ctx, filter, opts)
	if err != nil {
		return nil, mongowrap.ErrMongoFindFailed.SetIErrF("filter[%s] errmsg: %s",
			conv.JSON(filter), err)
	}
	defer cursor.Close(ctx)

	settings := map[string]interface{}{}

	for cursor.Next(ctx) {
		var data bson.M
		if err := cursor.Decode(&data); err != nil {
			clog.Warnf(ctx, "mongo collection settings invalid. filter[%s] result[%s] errmsg: %s",
				conv.JSON(filter), conv.JSON(data), err)
			continue
		}
		if key, exist := data[SettingKeyKey]; exist {
			settings[key.(string)] = data[SettingKeyVal]
		}
	}

	err = cursor.Err()
	if err != nil {
		return nil, mongowrap.ErrMongoFindFailed.SetIErrF("filter[%s] settings[%s] errmsg: %s",
			conv.JSON(filter), conv.JSON(settings), err)
	}

	return settings, nil
}

func SetSettings(ctx context.Context, settingType string, kvs map[string]interface{}) error {
	models := make([]mongodriver.WriteModel, 0, len(kvs))
	for key, val := range kvs {
		models = append(models, mongodriver.NewUpdateOneModel().
			SetFilter(bson.M{SettingKeyType: settingType, SettingKeyKey: key}).
			SetUpdate(bson.M{"$set": bson.M{SettingKeyType: settingType, SettingKeyKey: key, SettingKeyVal: val}}).
			SetUpsert(true),
		)
	}

	opts := options.BulkWrite().SetOrdered(false)
	result, err := mongo.Settings().BulkWrite(ctx, models, opts)
	if err != nil {
		return mongowrap.ErrMongoUpdateFailed.SetIErrF("models[%s] errmsg: %s",
			conv.JSON(models), err)
	}
	clog.Debugf(ctx, "SetSettings collection update succ. result[%s]", conv.JSON(result))
	return nil
}

func DelSettings(ctx context.Context, settingType string, keys ...string) (int64, error) {
	opts := options.Delete()
	filter := bson.M{SettingKeyType: settingType}

	if len(keys) > 0 && keys[0] != "" {
		filter[SettingKeyKey] = bson.M{"$in": keys}
	}
	clog.Debugf(ctx, "DelSettings filters[%s]", conv.JSON(filter))

	result, err := mongo.Settings().DeleteMany(ctx, filter, opts)
	if err != nil {
		return 0, mongowrap.ErrMongoDeleteFailed.SetIErrF("filter[%s] errmsg: %s",
			conv.JSON(filter), err)
	}
	clog.Debugf(ctx, "DelSettings collection delete succ. deleted[%d]", result.DeletedCount)
	return result.DeletedCount, nil
}
