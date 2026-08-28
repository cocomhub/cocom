// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package comic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/cache"
	"github.com/cocomhub/cocom/cmd/server/internal/mongo"
	"github.com/cocomhub/cocom/pkg/conv"
	"github.com/cocomhub/cocom/pkg/mongowrap"

	"github.com/cocomhub/cocom/pkg/comic"
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

func CacheKeyComicInfo(cid int) string {
	return fmt.Sprintf("comicInfo:%d", cid)
}

func CacheKeyRangeComicInfos(limit int64, skip int64, filters ...any) string {
	return fmt.Sprintf("comicInfos:limit:%d:skip:%d:%s", limit, skip, CacheKeyFilter(filters...))
}

func CacheKeyCountTotalComicInfos(filters ...any) string {
	return fmt.Sprintf("comicInfos:count:%s", CacheKeyFilter(filters...))
}

func CacheKeyTagList(tagType string, limit int64, skip int64, sortType int) string {
	return fmt.Sprintf("comicInfos:tagList:%s:limit:%d:skip:%d:sortType:%d", tagType, limit, skip, sortType)
}

func CacheKeyTagSectionIndices(tagType string, pageTagNum int) string {
	return fmt.Sprintf("comicInfos:tagSectionIndices:%s:pageTagNum:%d", tagType, pageTagNum)
}

func UpdateComicInfo(ctx context.Context, cid int, comicInfo map[string]any) (err error) {
	if s := GetDefaultStorage(); s != nil && asStorage(s) == nil {
		return s.Update(ctx, comicInfo)
	}

	return UpdateComicInfoDirect(ctx, cid, comicInfo)
}

// UpdateComicInfoDirect 绕过了 default-storage 分支，直接对底层（Mongo/cache）落库。
// 由本包 *Storage.Update 内部调用，防止 default-branch 自递归：
// UpdateComicInfo → s.Update → NewComicByObject → util.ToMap → UpdateComicInfo… 无限循环。
func UpdateComicInfoDirect(ctx context.Context, cid int, comicInfo map[string]any) (err error) {
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
		slog.ErrorContext(ctx, "delete comic info cache failed", slog.String("key", cacheKey), slog.String("err", errSet.Error()))
	} else {
		slog.DebugContext(ctx, "delete comic info cache succ", slog.String("key", cacheKey))
	}
	return
}

func GetComicInfo(ctx context.Context, cid int, info any) (err error) {
	if s := GetDefaultStorage(); s != nil && asStorage(s) == nil {
		c, cErr := s.Get(ctx, strconv.Itoa(cid))
		if cErr != nil {
			return fmt.Errorf("default storage get failed: %w", cErr)
		}
		// 将 storage 返回的 comic.Comic 序列化后再反序列化到目标类型（通常是 api.ComicInfo）。
		// comic.Comic 接口的 MarshalJSON 由 cmd/server/internal/comic.Comic 实现，
		// 输出的是 api.ComicInfo 的 JSON 格式，可以直接反序列化到 api.ComicInfo。
		data, marshalErr := json.Marshal(c)
		if marshalErr != nil {
			return fmt.Errorf("marshal comic from default storage failed: %w", marshalErr)
		}
		return json.Unmarshal(data, info)
	}

	return GetComicInfoDirect(ctx, cid, info)
}

// GetComicInfoDirect 绕过 default-storage 分支，直接对底层 Mongo/cache 读取。
// 由本包 *Storage.Get 内部调用，防止 default-branch 自递归：
// *Storage.Get → GetComicInfo → s.Get(→本方法) → … 无限循环。
func GetComicInfoDirect(ctx context.Context, cid int, info any) (err error) {
	cacheKey := CacheKeyComicInfo(cid)
	err = cache.Get(cacheKey, info)
	if err == nil {
		return
	}
	slog.DebugContext(ctx, "miss cache key", slog.String("key", cacheKey))

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
		slog.ErrorContext(ctx, "set comic info cache failed", slog.String("key", cacheKey), slog.String("err", errSet.Error()))
	} else {
		slog.DebugContext(ctx, "set comic info cache cache succ", slog.String("key", cacheKey))
	}
	return
}

func GetRangeComicInfos(ctx context.Context, limit int64, skip int64, filters ...any) (infos []*api.ComicInfo, err error) {
	if s := GetDefaultStorage(); s != nil && asStorage(s) == nil {
		// 注意：GetRangeComicInfos 的 filters 是 MongoDB 原生过滤（键值对/bson），
		// 无法直接翻译成 pkg/comic.ComicFilter 的强类型字段；此处从 filters 提取
		// k/v 映射后手动翻译为 ComicFilter（status→SetStatus、redirect_to exists:0→SetHasRedirect(false)、
		// deleted→SetDeleted），再传给 s.Find。未识别键按历史行为忽略（保底不丢分页）。
		filter := buildComicFilterFromFilters(filters...)
		filter.SetLimit(limit)
		filter.SetSkip(skip)
		comics, findErr := s.Find(ctx, filter)
		if findErr != nil {
			return nil, findErr
		}
		infos = make([]*api.ComicInfo, 0, len(comics))
		for _, c := range comics {
			data, marshalErr2 := json.Marshal(c)
			if marshalErr2 != nil {
				return nil, fmt.Errorf("marshal comic failed: %w", marshalErr2)
			}
			var info api.ComicInfo
			if unmarshalErr := json.Unmarshal(data, &info); unmarshalErr != nil {
				return nil, fmt.Errorf("unmarshal to ComicInfo failed: %w", unmarshalErr)
			}
			infos = append(infos, &info)
		}
		// 排序对齐：Mongo 分支 SortKV("cid", -1)（最新在前）；MemoryStorage.Find 返回升序，
		// 需反转（仅当当前页多于 1 条时反转，避免破坏单条/空的顺序）。
		for i, j := 0, len(infos)-1; i < j; i, j = i+1, j-1 {
			infos[i], infos[j] = infos[j], infos[i]
		}
		return infos, nil
	}
	cacheKey := CacheKeyRangeComicInfos(limit, skip, filters...)
	infos = []*api.ComicInfo{}
	err = cache.Get(cacheKey, &infos)
	if err == nil {
		return
	}
	slog.DebugContext(ctx, "miss cache key", slog.String("key", cacheKey))

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
		slog.ErrorContext(ctx, "set latest comic infos cache failed", slog.String("key", cacheKey), slog.String("err", errSet.Error()))
	} else {
		slog.DebugContext(ctx, "set latest comic infos cache succ", slog.String("key", cacheKey), slog.Int("count", len(infos)))
	}
	return
}

// buildComicFilterFromFilters 将 GetRangeComicInfos/CountTotalComicInfos 的 MongoDB
// 原生 filters（键值对列表）翻译为 pkg/comic.ComicFilter。仅识别首页所需的三类：
//   - ("status", bool)          → SetStatus(bool)
//   - ("redirect_to", bson.M{"$exists": bool|int})
//   - 值 1/true → SetHasRedirect(true)（保留有重定向关系的从属漫画）
//   - 值 0/false → SetHasRedirect(false)（过滤掉从属漫画）
//   - ("deleted", bson.M{"$ne": bool|int}) → SetDeleted(!ne)
//
// 注意：Mongo 查询常以整型 {"$exists": 1/0} 传值（bson.M = map[string]any），
// 与布尔混用。这里对 boolean 与 integer 均做安全转换，其它类型忽略（no-op）不 panic。
//
// 其余键未知不做转换（保留原分页行为），不 panic，避免 P0。
func buildComicFilterFromFilters(filters ...any) *comic.ComicFilter {
	filter := comic.NewComicFilter()
	for i := 0; i+1 < len(filters); i += 2 {
		key, ok := filters[i].(string)
		if !ok {
			continue
		}
		val := filters[i+1]
		switch key {
		case "status":
			switch b := val.(type) {
			case bool:
				filter.SetStatus(b)
			case int:
				// Mongo 查询习惯可能直接传整型 1/0，兼容安全转换。
				filter.SetStatus(b != 0)
			}
		case "redirect_to":
			// 兼容 Mongo 查询语法 {"$exists": 1/0} 与布尔 {"$exists": true/false}。
			// 语义：存在 redirect_to 字段 → hasRedirect=true；不存在 → false。
			// 值 1/true 表示有重定向，0/false 表示没有（过滤掉从属漫画）。
			// 其它类型一律忽略（保持 no-op，不 panic）。
			if m, ok := val.(bson.M); ok {
				switch e := m["$exists"].(type) {
				case bool:
					filter.SetHasRedirect(e)
				case int:
					filter.SetHasRedirect(e != 0)
				}
			}
		case "deleted":
			// Mongo 习惯传 {"$ne": true/false} 或整型 1/0；翻译为 SetDeleted(!ne)：
			// "$ne": true → deleted != true → not deleted → SetDeleted(false)。
			if m, ok := val.(bson.M); ok {
				switch ne := m["$ne"].(type) {
				case bool:
					filter.SetDeleted(!ne)
				case int:
					filter.SetDeleted(ne == 0)
				}
			}
		}
	}
	return filter
}

func CountTotalComicInfos(ctx context.Context, filters ...any) (count int64, err error) {
	if s := GetDefaultStorage(); s != nil && asStorage(s) == nil {
		// 与 GetRangeComicInfos 相同：把 MongoDB 原生 filters 翻译为 ComicFilter 后传给 FindTotal，
		// 保证首页总数与列表过滤条件一致（此前丢 filters 导致总数/列表不匹配）。
		cf := buildComicFilterFromFilters(filters...)
		cf.NoLimit()
		total, totalErr := s.FindTotal(ctx, cf)
		if totalErr != nil {
			return 0, totalErr
		}
		return total, nil
	}
	cacheKey := CacheKeyCountTotalComicInfos(filters...)
	err = cache.Get(cacheKey, &count)
	if err == nil {
		return
	}
	slog.DebugContext(ctx, "miss cache key", slog.String("key", cacheKey))

	count, err = mongo.ComicInfoBuilder().
		Filters(filters...).
		NoLimit().
		Count(ctx)
	if err != nil {
		return
	}

	setErr := cache.Set(cacheKey, count)
	if setErr != nil {
		slog.ErrorContext(ctx, "set comic info total count cache failed", slog.String("key", cacheKey), slog.String("err", setErr.Error()))
	} else {
		slog.DebugContext(ctx, "set comic info total count cache succ", slog.String("key", cacheKey), slog.Int64("count", count))
	}
	return
}

var (
	SortTypeByName    = 0
	SortTypeByPopular = 1
)

func aggregateTagSort(sortType int) bson.M {
	if sortType == SortTypeByName {
		return bson.M{"$sort": bson.M{"_id.name": 1}}
	}
	return bson.M{"$sort": bson.M{"count": -1}}
}

// AggregateTagList 聚合标签列表
/*
db.comicInfo.aggregate([
  // 步骤1：过滤出至少包含一个 tags.type = "artist" 的文档
  { $match: { "tags.type": "artist" } },

  // 步骤2: 展开 tags 数组（每个标签生成独立文档）
  { $unwind: "$tags" },

  // 步骤3: 筛选 artist 类型标签
  { $match: { "tags.type": "artist" } },

  // 步骤4: 按四元组分组并统计真实次数
  { $group: {
      _id: {
        id: "$tags.id",
        name: "$tags.name",
        type: "$tags.type",
        url: "$tags.url"
      },
      count: { $sum: 1 }  // 统计该组合出现的总次数
    }
  },

  // 步骤5: 分页和总数统计
  { $facet: {
      // 分页数据管道
      paginatedResults: [
        { $sort: { count: -1 } },  // 排序（按需调整）
        { $skip: 20 },             // 跳过前N条
        { $limit: 10 },            // 返回M条
        { $project: {              // 重组输出格式
            _id: 0,                // 隐藏分组ID
            id: "$_id.id",         // 提取原始字段
            name: "$_id.name",     // 提取原始字段
            type: "$_id.type",     // 提取原始字段
            url: "$_id.url",       // 提取原始字段
            count: 1               // 保留统计值
          }
        }
      ],

      // 总数统计管道
      totalCount: [
        { $count: "count" }  // 统计总记录数
      ]
    }
  },

  // 阶段6: 重组输出格式
  { $project: {
      data: "$paginatedResults",
      total: { $arrayElemAt: ["$totalCount.count", 0] }
    }
  }
])
*/
func AggregateTagList(ctx context.Context, tagType string, sortType int, skip, limit int64) (tags []*api.TagInfo, total int64, err error) {
	cacheKey := CacheKeyTagList(tagType, limit, skip, sortType)
	var results []struct {
		Data  []*api.TagInfo `bson:"data"`
		Total int            `bson:"total"`
	}
	err = cache.Get(cacheKey, &results)
	if err == nil && len(results) > 0 {
		return results[0].Data, int64(results[0].Total), nil
	}
	slog.DebugContext(ctx, "miss cache key", slog.String("key", cacheKey))

	pipe := []bson.M{
		{"$match": bson.M{"tags.type": tagType}},
		{"$unwind": "$tags"},
		{"$match": bson.M{"tags.type": tagType}},
		{"$group": bson.M{
			"_id":   bson.M{"id": "$tags.id", "name": "$tags.name", "type": "$tags.type", "url": "$tags.url"},
			"count": bson.M{"$sum": 1},
		}},
		{"$facet": bson.M{
			"paginatedResults": []bson.M{
				aggregateTagSort(sortType),
				{"$skip": skip},
				{"$limit": limit},
				{"$project": bson.M{
					"_id":   0,
					"id":    "$_id.id",
					"name":  "$_id.name",
					"type":  "$_id.type",
					"url":   "$_id.url",
					"count": 1,
				}},
			},
			"totalCount": []bson.M{
				{"$count": "count"},
			},
		}},
		{"$project": bson.M{
			"data":  "$paginatedResults",
			"total": bson.M{"$arrayElemAt": bson.A{"$totalCount.count", 0}},
		}},
	}

	err = mongo.ComicInfoBuilder().
		Aggregate(ctx, pipe, &results)
	if err != nil {
		return
	}
	if len(results) == 0 || len(results[0].Data) == 0 {
		err = errors.New("AggregateTagList result is empty")
		return
	}

	errSet := cache.Set(cacheKey, results)
	if errSet != nil {
		slog.ErrorContext(ctx, "set tag list cache failed",
			slog.String("key", cacheKey), slog.String("err", errSet.Error()))
	} else {
		slog.DebugContext(ctx, "set tag list cache succ",
			slog.String("key", cacheKey), slog.Int("count", len(results[0].Data)))
	}
	return results[0].Data, int64(results[0].Total), nil
}

// AggregateTagNameSectionIndex 聚合标签名称和索引
/*
db.comicInfo.aggregate([
  // 步骤1：过滤出至少包含一个 tags.type = "artist" 的文档
  { $match: { "tags.type": "artist" } },

  // 步骤2: 展开 tags 数组（每个标签生成独立文档）
  { $unwind: "$tags" },

  // 步骤3: 筛选 artist 类型标签
  { $match: { "tags.type": "artist" } },

  // 步骤4: 按四元组分组并统计真实次数
  { $group: {
      _id: {
        id: "$tags.id",
        name: "$tags.name",
        type: "$tags.type",
        url: "$tags.url"
      },
      count: { $sum: 1 }  // 统计该组合出现的总次数
    }
  },

  // 步骤5：计算字母分组
  { $addFields: {
      alphaGroup: {
        $cond: [
          { $regexMatch: {
              input: { $substrCP: [ "$_id.name", 0, 1 ] },
              regex: /^[A-Z]$/i
          }},
          { $toUpper: { $substrCP: [ "$_id.name", 0, 1 ] } },
          "#"
        ]
      }
    }
  },

  // 步骤6：按字母分组统计数量
  { $group: {
      _id: "$alphaGroup",
      count: { $sum: 1 }
    }
  }
])
*/
func AggregateTagSectionIndices(ctx context.Context, tagType string, pageTagNum int) ([]*api.TagSectionIndex, error) {
	cacheKey := CacheKeyTagSectionIndices(tagType, pageTagNum)
	tagSectionIndices := make([]*api.TagSectionIndex, 0, 27)
	err := cache.Get(cacheKey, &tagSectionIndices)
	if err == nil && len(tagSectionIndices) > 0 {
		return tagSectionIndices, nil
	}
	slog.DebugContext(ctx, "miss cache key", slog.String("key", cacheKey))

	pipe := []bson.M{
		{"$match": bson.M{"tags.type": tagType}},
		{"$unwind": "$tags"},
		{"$match": bson.M{"tags.type": tagType}},
		{"$group": bson.M{
			"_id":   bson.M{"id": "$tags.id", "name": "$tags.name", "type": "$tags.type", "url": "$tags.url"},
			"count": bson.M{"$sum": 1},
		}},
		{"$addFields": bson.M{
			"alphaGroup": bson.M{
				"$cond": bson.A{
					bson.M{
						"$regexMatch": bson.M{
							"input":   bson.M{"$substrCP": bson.A{"$_id.name", 0, 1}},
							"regex":   "^[A-Z]$",
							"options": "i",
						},
					},
					bson.M{"$toUpper": bson.M{"$substrCP": bson.A{"$_id.name", 0, 1}}},
					"#",
				},
			},
		}},
		{"$group": bson.M{
			"_id":   "$alphaGroup",
			"count": bson.M{"$sum": 1},
		}},
	}

	var results []struct {
		ID    string `bson:"_id"`
		Count int    `bson:"count"`
	}
	err = mongo.ComicInfoBuilder().
		Aggregate(ctx, pipe, &results)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, errors.New("AggregateTagNameSectionIndex result is empty")
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].ID < results[j].ID
	})
	var sectionIndex int
	for _, result := range results {
		tagSectionIndices = append(tagSectionIndices, &api.TagSectionIndex{
			Name:  result.ID,
			Index: sectionIndex,
			Page:  sectionIndex/pageTagNum + 1,
		})
		sectionIndex += result.Count
	}

	errSet := cache.Set(cacheKey, tagSectionIndices)
	if errSet != nil {
		slog.ErrorContext(ctx, "set tag section indices cache failed",
			slog.String("key", cacheKey), slog.String("err", errSet.Error()))
	} else {
		slog.DebugContext(ctx, "set tag section indices cache succ",
			slog.String("key", cacheKey), slog.Int("count", len(tagSectionIndices)))
	}
	return tagSectionIndices, nil
}

// DeleteComicByID 软删除 comic：原文档删除 + 插入最小 tombstone 记录
func DeleteComicByID(ctx context.Context, cid int) error {
	if s := GetDefaultStorage(); s != nil {
		return s.ArchiveByID(ctx, strconv.Itoa(cid))
	}

	// 1. 获取原漫画信息（用于清理文件和归档）
	var info api.ComicInfo
	if err := GetComicInfo(ctx, cid, &info); err != nil {
		return fmt.Errorf("get comic info failed: %w", err)
	}

	// 2. 删除原 MongoDB 文档
	filter := bson.M{"cid": cid}
	if _, err := mongo.ComicInfo().DeleteOne(ctx, filter); err != nil {
		return fmt.Errorf("delete comic document failed: %w", err)
	}

	// 3. 插入 tombstone 记录
	tombstone := bson.M{
		"cid":        cid,
		"deleted":    true,
		"deleted_at": time.Now(),
	}
	if _, err := mongo.ComicInfo().InsertOne(ctx, tombstone); err != nil {
		return fmt.Errorf("insert tombstone failed: %w", err)
	}

	// 4. 清理图片目录（非阻塞，不阻止删除流程）
	saveDir := info.SaveDir()
	if saveDir != "" {
		if err := os.RemoveAll(saveDir); err != nil {
			slog.WarnContext(ctx, "DeleteComicByID: remove save dir failed",
				slog.Int("cid", cid),
				slog.String("dir", saveDir),
				slog.String("errmsg", err.Error()))
		}
	}

	// 5. 清理归档文件（非阻塞，不阻止删除流程）
	if info.Archive != nil && info.Archive.Path != "" {
		if err := os.Remove(info.Archive.Path); err != nil {
			slog.WarnContext(ctx, "DeleteComicByID: remove archive file failed",
				slog.Int("cid", cid),
				slog.String("path", info.Archive.Path))
		}
	}

	// 6. 清除缓存
	_ = cache.Reset()

	return nil
}
