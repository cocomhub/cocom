// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package video

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/cocomhub/cocom/cmd/server/internal/cache"
	"github.com/cocomhub/cocom/cmd/server/internal/mongo"
	"github.com/cocomhub/cocom/pkg/conv"
	"github.com/cocomhub/cocom/pkg/mongowrap"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func CacheKeyFilter(filters ...any) string {
	if len(filters) == 0 {
		return "total"
	}
	builder := strings.Builder{}
	builder.WriteString("filters")
	for _, v := range filters {
		fmt.Fprintf(&builder, ":%v", v)
	}
	return builder.String()
}

func CacheKeyVideoInfo(vid string) string {
	return fmt.Sprintf("videoInfo:%s", vid)
}

func CacheKeyRangeVideoInfos(limit int64, skip int64, filters ...any) string {
	return fmt.Sprintf("videoInfos:limit:%d:skip:%d:%s", limit, skip, CacheKeyFilter(filters...))
}

func CacheKeyCountTotalVideoInfos(filters ...any) string {
	return fmt.Sprintf("videoInfos:count:%s", CacheKeyFilter(filters...))
}

func UpdateVideoInfo(ctx context.Context, vid string, videoInfo map[string]any) (err error) {
	if s := defaultStore; s != nil {
		return s.Update(ctx, vid, videoInfo)
	}

	opts := options.Update().SetUpsert(true)
	filter := bson.M{"vid": vid}
	update := bson.M{"$set": videoInfo}
	delete(videoInfo, "_id")

	_, err = mongo.VideoInfo().UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return mongowrap.ErrMongoUpdateFailed.SetIErrF("mongo collection update failed. filter[%s] update[%s] opts[%s] errmsg: %s",
			conv.JSON(filter), conv.JSON(update), conv.JSON(opts), err.Error())
	}

	cacheKey := CacheKeyVideoInfo(vid)
	errSet := cache.Delete(cacheKey)
	if errSet != nil {
		slog.ErrorContext(ctx, "delete video info cache failed", slog.String("key", cacheKey), slog.String("err", errSet.Error()))
		return
	}
	slog.DebugContext(ctx, "delete video info cache succ", slog.String("key", cacheKey))
	return
}

func GetVideoInfo(ctx context.Context, vid string, info any) (err error) {
	if s := defaultStore; s != nil {
		return s.Get(ctx, vid, info)
	}

	cacheKey := CacheKeyVideoInfo(vid)
	err = cache.Get(cacheKey, info)
	if err == nil {
		return
	}
	slog.DebugContext(ctx, "miss cache key", slog.String("key", cacheKey))

	opts := options.FindOne()
	filter := bson.M{"vid": vid}

	result := mongo.VideoInfo().FindOne(ctx, filter, opts)
	if result.Err() != nil {
		return mongowrap.ErrMongoFindFailed.SetIErrF("filter[%s] opts[%s] errmsg: %s",
			conv.JSON(filter), conv.JSON(opts), result.Err().Error())
	}

	err = result.Decode(info)
	if err != nil {
		return mongowrap.ErrMongoDecodeFailed.SetIErrF("filter[%s] opts[%s] errmsg: %s",
			conv.JSON(filter), conv.JSON(opts), err.Error())
	}

	errSet := cache.Set(cacheKey, info)
	if errSet != nil {
		slog.ErrorContext(ctx, "set video info cache failed", slog.String("key", cacheKey), slog.String("err", errSet.Error()))
		return
	}
	slog.DebugContext(ctx, "set video info cache cache succ", slog.String("key", cacheKey))
	return
}
