// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package tag

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/mongo"
	"github.com/cocomhub/cocom/pkg/mongowrap"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TagRelationDoc 表示一组 tag 之间的显式关联关系
type TagRelationDoc struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Tags      []TagBriefDoc      `bson:"tags"`
	CreatedAt time.Time          `bson:"created_at"`
}

// TagBriefDoc tagRelation 中单个 tag 的文档结构
type TagBriefDoc struct {
	ID   int    `bson:"id"`
	Name string `bson:"name"`
	Type string `bson:"type"`
	URL  string `bson:"url"`
}

// CreateRelation 创建关系组
func CreateRelation(ctx context.Context, tags []api.TagBrief) (*TagRelationDoc, error) {
	if s := GetDefaultRelationStore(); s != nil {
		_, err := s.CreateRelation(ctx, tags)
		if err != nil {
			return nil, err
		}
		return &TagRelationDoc{ID: primitive.ObjectID{}, Tags: nil, CreatedAt: time.Now()}, nil
	}
	if len(tags) < 2 {
		return nil, fmt.Errorf("at least 2 tags required for a relation")
	}

	docs := make([]TagBriefDoc, len(tags))
	for i, t := range tags {
		docs[i] = TagBriefDoc{ID: t.ID, Name: t.Name, Type: t.Type, URL: t.URL}
	}

	doc := TagRelationDoc{
		ID:        primitive.NewObjectID(),
		Tags:      docs,
		CreatedAt: time.Now(),
	}

	if _, err := mongo.TagRelation().InsertOne(ctx, doc); err != nil {
		return nil, mongowrap.ErrMongoUpdateFailed.SetIErrF("tagRelation insert failed: %s", err.Error())
	}

	return &doc, nil
}

// DeleteRelation 按 ID 删除关系组
func DeleteRelation(ctx context.Context, groupID string) error {
	if s := GetDefaultRelationStore(); s != nil {
		return s.DeleteRelation(ctx, groupID)
	}
	oid, err := primitive.ObjectIDFromHex(groupID)
	if err != nil {
		return fmt.Errorf("invalid group id: %w", err)
	}

	filter := bson.M{"_id": oid}
	result, err := mongo.TagRelation().DeleteOne(ctx, filter)
	if err != nil {
		return mongowrap.ErrMongoDeleteFailed.SetIErrF("tagRelation delete failed: %s", err.Error())
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("relation not found")
	}

	slog.DebugContext(ctx, "tagRelation deleted", slog.String("id", groupID))
	return nil
}

// GetRelationsForTag 获取指定 tag 所属的所有关系组
func GetRelationsForTag(ctx context.Context, tagType string, tagID int) ([]TagRelationDoc, error) {
	var docs []TagRelationDoc
	if err := mongo.TagRelationBuilder().
		FilterKV("tags", bson.M{"$elemMatch": bson.M{"type": tagType, "id": tagID}}).
		NoLimit().
		All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

// GetRelatedTagsFromRelations 获取指定 tag 通过关系组关联的其他 tag（去重）
func GetRelatedTagsFromRelations(ctx context.Context, tagType string, tagID int) ([]*api.TagInfo, error) {
	groups, err := GetRelationsForTag(ctx, tagType, tagID)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	excludeKey := fmt.Sprintf("%s:%d", tagType, tagID)

	var result []*api.TagInfo
	for _, g := range groups {
		for _, t := range g.Tags {
			key := fmt.Sprintf("%s:%d", t.Type, t.ID)
			if key == excludeKey {
				continue
			}
			if seen[key] {
				continue
			}
			seen[key] = true

			// 获取 like 状态
			liked := false
			doc, _ := GetTagByID(ctx, t.Type, t.ID)
			if doc != nil && doc.Like {
				liked = true
			}

			// 构造 URL（如 /artist/azuma-tesshou/）
			urlPath := fmt.Sprintf("/%s/%s/", t.Type, t.Name)

			result = append(result, &api.TagInfo{
				ID:       t.ID,
				Name:     t.Name,
				Type:     t.Type,
				URL:      urlPath,
				Count:    0,
				Like:     liked,
				Explicit: true,
			})
		}
	}

	// 按 type 排序分组展示
	sort.Slice(result, func(i, j int) bool {
		if result[i].Type != result[j].Type {
			return result[i].Type < result[j].Type
		}
		return result[i].Name < result[j].Name
	})

	return result, nil
}

// GetRelationsGroupList 获取指定 tag 所属的所有关系组（含完整组信息，用于前端管理）
func GetRelationsGroupList(ctx context.Context, tagType string, tagID int) ([]api.RelationGroup, error) {
	groups, err := GetRelationsForTag(ctx, tagType, tagID)
	if err != nil {
		return nil, err
	}

	result := make([]api.RelationGroup, 0, len(groups))
	for _, g := range groups {
		tags := make([]api.TagBrief, len(g.Tags))
		for i, t := range g.Tags {
			tags[i] = api.TagBrief{ID: t.ID, Name: t.Name, Type: t.Type, URL: t.URL}
		}

		var createdAt string
		if !g.CreatedAt.IsZero() {
			createdAt = g.CreatedAt.Format(time.RFC3339)
		}

		result = append(result, api.RelationGroup{
			ID:        g.ID.Hex(),
			Tags:      tags,
			CreatedAt: createdAt,
		})
	}

	// 按创建时间倒序
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt > result[j].CreatedAt
	})

	return result, nil
}
