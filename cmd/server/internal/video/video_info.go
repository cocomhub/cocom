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
package video

import (
	"context"
	"fmt"
	"strings"

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

func CacheKeyVideoInfo(vid string) string {
	return fmt.Sprintf("videoInfo:%s", vid)
}

func CacheKeyRangeVideoInfos(limit int64, skip int64, filters ...interface{}) string {
	return fmt.Sprintf("videoInfos:limit:%d:skip:%d:%s", limit, skip, CacheKeyFilter(filters...))
}

func CacheKeyCountTotalVideoInfos(filters ...interface{}) string {
	return fmt.Sprintf("videoInfos:count:%s", CacheKeyFilter(filters...))
}

func UpdateVideoInfo(ctx context.Context, vid string, videoInfo map[string]interface{}) (err error) {
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
		clog.Errorf(ctx, "delete video info cache failed. key[%s] errmsg: %s", cacheKey, errSet.Error())
		return
	}
	clog.Debugf(ctx, "delete video info cache succ. key[%s]", cacheKey)
	return
}

func GetVideoInfo(ctx context.Context, vid string, info interface{}) (err error) {
	cacheKey := CacheKeyVideoInfo(vid)
	err = cache.Get(cacheKey, info)
	if err == nil {
		return
	}
	clog.Debugf(ctx, "miss cache key[%s]", cacheKey)

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
		clog.Errorf(ctx, "set video info cache failed. key[%s] info[%v] errmsg: %s",
			cacheKey, conv.JSON(info), errSet.Error())
		return
	}
	clog.Debugf(ctx, "set video info cache cache succ. key[%s]", cacheKey)
	return
}
