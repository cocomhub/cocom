// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package comic

import (
	"context"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/mongo"
	"github.com/cocomhub/cocom/pkg/util"
	"go.mongodb.org/mongo-driver/bson"
)

func GetMoreLikeThis(ctx context.Context, cid int, tags api.Tags, limit int64) (infos []*api.ComicInfo, err error) {
	infos = []*api.ComicInfo{}
	if limit <= 0 {
		return infos, nil
	}
	filter := bson.M{"cid": bson.M{"$ne": cid}}
	var idList []int
	for _, t := range tags {
		if t.ID != 0 {
			idList = append(idList, t.ID)
		}
	}
	if len(idList) > 0 {
		filter["tags"] = bson.M{"$elemMatch": bson.M{"id": bson.M{"$in": idList}}}
	} else if len(tags) > 0 {
		var pairs []bson.M
		for _, t := range tags {
			if t.Name != "" && t.Type != "" {
				pairs = append(pairs, bson.M{"name": t.Name, "type": t.Type})
			}
		}
		if len(pairs) > 0 {
			filter["tags"] = bson.M{"$elemMatch": bson.M{"$or": pairs}}
		}
	}
	candidateLimit := max(limit*4, limit)
	builder := mongo.ComicInfoBuilder().Filters().FilterKV("cid", bson.M{"$ne": cid})
	for k, v := range filter {
		builder.FilterKV(k, v)
	}
	err = builder.SortKV("cid", -1).Limit(candidateLimit).All(ctx, &infos)
	if err != nil {
		return nil, err
	}
	if len(infos) == 0 {
		infos = []*api.ComicInfo{}
	}
	util.Shuffle(len(infos), func(i, j int) { infos[i], infos[j] = infos[j], infos[i] })
	if int64(len(infos)) > limit {
		infos = infos[:limit]
	}
	return infos, nil
}

// GetByTagType 根据 tag 类型获取推荐漫画
// 从当前漫画提取指定 tagType 的标签 ID，查询有相同标签的漫画，排除当前漫画并随机打乱
func GetByTagType(ctx context.Context, cid int, tags api.Tags, tagType string, limit int) (infos []*api.ComicInfo, err error) {
	infos = []*api.ComicInfo{}
	if limit <= 0 {
		return infos, nil
	}

	// 提取该类型的标签 ID
	var idList []int
	for _, t := range tags {
		if t.Type == tagType && t.ID != 0 {
			idList = append(idList, t.ID)
		}
	}

	// 如果没有该类型的标签，返回空
	if len(idList) == 0 {
		return infos, nil
	}

	// 查询有相同标签的其他漫画
	candidateLimit := int64(max(limit*4, limit))
	builder := mongo.ComicInfoBuilder().
		FilterKV("cid", bson.M{"$ne": cid}).
		FilterKV("tags", bson.M{"$elemMatch": bson.M{"id": bson.M{"$in": idList}}}).
		SortKV("cid", -1).
		Limit(candidateLimit)

	if err = builder.All(ctx, &infos); err != nil {
		return nil, err
	}
	if len(infos) == 0 {
		infos = []*api.ComicInfo{}
		return infos, nil
	}

	// 随机打乱并截取
	util.Shuffle(len(infos), func(i, j int) { infos[i], infos[j] = infos[j], infos[i] })
	if len(infos) > limit {
		infos = infos[:limit]
	}
	return infos, nil
}
