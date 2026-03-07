// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package onecomic

import (
	"context"
	"fmt"
	"strings"

	"github.com/cocomhub/cocom/cmd/server/internal/cache"
	"github.com/cocomhub/cocom/cmd/server/internal/mongo"
	"github.com/cocomhub/cocom/pkg/clog"
	"github.com/cocomhub/cocom/pkg/conv"
	"github.com/cocomhub/cocom/pkg/mongowrap"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func CacheKeyFilter(filters ...interface{}) string {
	if len(filters) == 0 {
		return "total"
	}
	builder := strings.Builder{}
	builder.WriteString("filters")
	for _, v := range filters {
		builder.WriteString(fmt.Sprintf(":%v", v))
	}
	return builder.String()
}

func CacheKeyOneComicInfo(cid string) string {
	return fmt.Sprintf("oneComicInfo:%s", cid)
}

func CacheKeyRangeOneComicInfos(limit int64, skip int64, filters ...interface{}) string {
	return fmt.Sprintf("oneComicInfos:limit:%d:skip:%d:%s", limit, skip, CacheKeyFilter(filters...))
}

func CacheKeyCountTotalOneComicInfos(filters ...interface{}) string {
	return fmt.Sprintf("oneComicInfos:count:%s", CacheKeyFilter(filters...))
}

func UpdateOneComicInfo(ctx context.Context, cid string, oneComicInfo map[string]interface{}) (err error) {
	opts := options.Update().SetUpsert(true)
	filter := bson.M{"cid": cid}
	update := bson.M{"$set": oneComicInfo}
	delete(oneComicInfo, "_id")

	_, err = mongo.OneComicInfo().UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return mongowrap.ErrMongoUpdateFailed.SetIErrF("mongo collection update failed. filter[%s] update[%s] opts[%s] errmsg: %s",
			conv.JSON(filter), conv.JSON(update), conv.JSON(opts), err.Error())
	}

	cacheKey := CacheKeyOneComicInfo(cid)
	errSet := cache.Delete(cacheKey)
	if errSet != nil {
		clog.Errorf(ctx, "delete oneComic info cache failed. key[%s] errmsg: %s", cacheKey, errSet.Error())
		return
	}
	clog.Debugf(ctx, "delete oneComic info cache succ. key[%s]", cacheKey)
	return
}

func GetOneComicInfo(ctx context.Context, cid string, info interface{}) (err error) {
	cacheKey := CacheKeyOneComicInfo(cid)
	err = cache.Get(cacheKey, info)
	if err == nil {
		return
	}
	clog.Debugf(ctx, "miss cache key[%s]", cacheKey)

	opts := options.FindOne()
	filter := bson.M{"cid": cid}

	result := mongo.OneComicInfo().FindOne(ctx, filter, opts)
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
		clog.Errorf(ctx, "set oneComic info cache failed. key[%s] info[%v] errmsg: %s",
			cacheKey, conv.JSON(info), errSet.Error())
		return
	}
	clog.Debugf(ctx, "set oneComic info cache cache succ. key[%s]", cacheKey)
	return
}
