// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build !memory_storage_integration

package setting

import (
	"context"
	"log/slog"

	"github.com/cocomhub/cocom/cmd/server/internal/mongo"
	"github.com/cocomhub/cocom/pkg/conv"
	"github.com/cocomhub/cocom/pkg/mongowrap"

	"go.mongodb.org/mongo-driver/bson"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	SettingKeyType string = "type"
	SettingKeyKey  string = "key"
	SettingKeyVal  string = "val"
)

func GetSettings(ctx context.Context, settingType string, keys ...string) (map[string]any, error) {
	opts := options.Find()
	filter := bson.M{SettingKeyType: settingType}

	if len(keys) > 0 && keys[0] != "" {
		filter[SettingKeyKey] = bson.M{"$in": keys}
	}
	slog.DebugContext(ctx, "GetSettings filters", slog.String("filter", conv.JSON(filter)))

	cursor, err := mongo.Settings().Find(ctx, filter, opts)
	if err != nil {
		return nil, mongowrap.ErrMongoFindFailed.SetIErrF("filter[%s] errmsg: %s",
			conv.JSON(filter), err)
	}
	defer cursor.Close(ctx)

	settings := map[string]any{}

	for cursor.Next(ctx) {
		var data bson.M
		if err := cursor.Decode(&data); err != nil {
			slog.WarnContext(ctx, "mongo collection settings invalid",
				slog.String("filter", conv.JSON(filter)),
				slog.String("result", conv.JSON(data)),
				slog.String("err", err.Error()))
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

func SetSettings(ctx context.Context, settingType string, kvs map[string]any) error {
	models := make([]mongodriver.WriteModel, 0, len(kvs))
	for key, val := range kvs {
		models = append(
			models, mongodriver.NewUpdateOneModel().
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
	slog.DebugContext(ctx, "SetSettings collection update succ", slog.String("result", conv.JSON(result)))
	return nil
}

func DelSettings(ctx context.Context, settingType string, keys ...string) (int64, error) {
	opts := options.Delete()
	filter := bson.M{SettingKeyType: settingType}

	if len(keys) > 0 && keys[0] != "" {
		filter[SettingKeyKey] = bson.M{"$in": keys}
	}
	slog.DebugContext(ctx, "DelSettings filters", slog.String("filter", conv.JSON(filter)))

	result, err := mongo.Settings().DeleteMany(ctx, filter, opts)
	if err != nil {
		return 0, mongowrap.ErrMongoDeleteFailed.SetIErrF("filter[%s] errmsg: %s",
			conv.JSON(filter), err)
	}
	slog.DebugContext(ctx, "DelSettings collection delete succ", slog.Int64("deleted", result.DeletedCount))
	return result.DeletedCount, nil
}
