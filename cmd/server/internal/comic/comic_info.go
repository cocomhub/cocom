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
package comic

import (
	"context"
	"fmt"
	"strings"

	"github.com/suixibing/cocom/cmd/server/api"
	"github.com/suixibing/cocom/cmd/server/internal/cache"
	"github.com/suixibing/cocom/cmd/server/internal/mongo"
	"github.com/suixibing/cocom/pkg/clog"
	"github.com/suixibing/cocom/pkg/conv"
	"github.com/suixibing/cocom/pkg/mongowrap"

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

func CacheKeyComicInfo(cid int) string {
	return fmt.Sprintf("comicInfo:%d", cid)
}

func CacheKeyRangeComicInfos(limit int64, skip int64, filters ...interface{}) string {
	return fmt.Sprintf("comicInfos:limit:%d:skip:%d:%s", limit, skip, CacheKeyFilter(filters...))
}

func CacheKeyCountTotalComicInfos(filters ...interface{}) string {
	return fmt.Sprintf("comicInfos:count:%s", CacheKeyFilter(filters...))
}

func UpdateComicInfo(ctx context.Context, cid int, comicInfo map[string]interface{}) (err error) {
	opts := options.Update().SetUpsert(true)
	filter := bson.M{"cid": cid}
	update := bson.M{"$set": comicInfo}
	delete(comicInfo, "_id")

	_, err = mongo.ComicInfo().UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return mongowrap.ErrMongoUpdateFailed.SetIErrF("mongo collection update failed. filter[%s] update[%s] opts[%s] errmsg: %s",
			conv.JSON(filter), conv.JSON(update), conv.JSON(opts), err.Error())
	}

	cacheKey := CacheKeyComicInfo(cid)
	errSet := cache.Delete(cacheKey)
	if errSet != nil {
		clog.Errorf(ctx, "delete comic info cache failed. key[%s] errmsg: %s", cacheKey, errSet.Error())
		return
	}
	clog.Debugf(ctx, "delete comic info cache succ. key[%s]", cacheKey)
	return
}

func GetComicInfo(ctx context.Context, cid int, info interface{}) (err error) {
	cacheKey := CacheKeyComicInfo(cid)
	err = cache.Get(cacheKey, info)
	if err == nil {
		return
	}
	clog.Debugf(ctx, "miss cache key[%s]", cacheKey)

	opts := options.FindOne()
	filter := bson.M{"cid": cid}

	result := mongo.ComicInfo().FindOne(ctx, filter, opts)
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
		clog.Errorf(ctx, "set comic info cache failed. key[%s] info[%v] errmsg: %s",
			cacheKey, conv.JSON(info), errSet.Error())
		return
	}
	clog.Debugf(ctx, "set comic info cache cache succ. key[%s]", cacheKey)
	return
}

func GetRangeComicInfos(ctx context.Context, limit int64, skip int64, filters ...interface{}) (infos []*api.ComicInfo, err error) {
	cacheKey := CacheKeyRangeComicInfos(limit, skip, filters...)
	infos = []*api.ComicInfo{}
	err = cache.Get(cacheKey, &infos)
	if err == nil {
		return
	}
	clog.Debugf(ctx, "miss cache key[%s]", cacheKey)

	err = mongo.ComicInfoBuilder().
		Filters(filters...).
		SortKV("cid", -1).
		Limit(limit).Skip(skip).
		All(ctx, &infos)
	if err != nil {
		return
	}

	errSet := cache.Set(cacheKey, infos)
	if errSet != nil {
		clog.Errorf(ctx, "set latest comic infos cache failed. key[%s] infos[%s] errmsg: %s",
			cacheKey, conv.JSON(infos), errSet.Error())
		return
	}
	clog.Debugf(ctx, "set latest comic infos cache succ. key[%s] infos[%d]", cacheKey, len(infos))
	return
}

func CountTotalComicInfos(ctx context.Context, filters ...interface{}) (count int64, err error) {
	cacheKey := CacheKeyCountTotalComicInfos(filters...)
	err = cache.Get(cacheKey, &count)
	if err == nil {
		return
	}
	clog.Debugf(ctx, "miss cache key[%s]", cacheKey)

	count, err = mongo.ComicInfoBuilder().
		Filters(filters...).
		NoLimit().
		Count(ctx)
	if err != nil {
		return
	}

	setErr := cache.Set(cacheKey, count)
	if setErr != nil {
		clog.Errorf(ctx, "set comic info total count cache failed. key[%s] count[%v] errmsg: %s",
			cacheKey, count, setErr.Error())
		return
	}
	clog.Debugf(ctx, "set comic info total count cache succ. key[%s] count[%v]", cacheKey, count)
	return
}
