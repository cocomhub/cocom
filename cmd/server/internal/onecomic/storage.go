package onecomic

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/suixibing/cocom/cmd/server/api"
	"github.com/suixibing/cocom/cmd/server/internal/mongo"
	"github.com/suixibing/cocom/pkg/clog"
	"github.com/suixibing/cocom/pkg/comic"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Storage 实现comic.ComicStorage接口
type Storage struct{}

// NewStorage 创建存储实例
func NewStorage() *Storage {
	return &Storage{}
}

// Get 获取漫画信息
func (s *Storage) Get(ctx context.Context, id string) (comic.Comic, error) {
	info := &api.OneComicInfo{}
	err := GetOneComicInfo(ctx, id, info)
	if err != nil {
		return nil, fmt.Errorf("failed to get onecomic: %w", err)
	}
	return NewComic(info), nil
}

// Update 更新漫画数据
func (s *Storage) Update(ctx context.Context, obj interface{}) error {
	c, err := NewComicByObject(obj)
	if err != nil {
		return err
	}

	if c == nil || c.Comicid == "" {
		return fmt.Errorf("invalid comic info")
	}

	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal comic info: %w", err)
	}
	v := map[string]interface{}{}
	err = json.Unmarshal(data, &v)
	if err != nil {
		return fmt.Errorf("failed to unmarshal comic info: %w", err)
	}

	err = UpdateOneComicInfo(ctx, c.Comicid, v)
	if err != nil {
		return fmt.Errorf("failed to save comic: %w", err)
	}
	return nil
}

// Find 列出符合条件的漫画
func (s *Storage) Find(ctx context.Context, filter *comic.ComicFilter) ([]comic.Comic, error) {
	cursor, err := mongo.OneComicInfo().Find(ctx, s.toMongoFilter(filter), &options.FindOptions{
		Sort:  bson.M{"comicid": 1},
		Limit: &filter.Limit,
		Skip:  &filter.Skip,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var impls []Comic
	if err := cursor.All(ctx, &impls); err != nil {
		return nil, err
	}

	// 转换为接口类型
	comics := make([]comic.Comic, len(impls))
	for i := range impls {
		comics[i] = &impls[i]
	}
	return comics, nil
}

// FindTotal 列出符合条件的漫画总数
func (s *Storage) FindTotal(ctx context.Context, filter *comic.ComicFilter) (int64, error) {
	return mongo.OneComicInfo().CountDocuments(ctx, s.toMongoFilter(filter), &options.CountOptions{
		Limit: &filter.Limit,
		Skip:  &filter.Skip,
	})
}

// FindChannel 列出符合条件的漫画，返回通道
func (s *Storage) FindChannel(ctx context.Context, filter *comic.ComicFilter) (chan comic.Comic, error) {
	comics := make(chan comic.Comic, 100)
	go func() {
		defer close(comics)
		oriLimit := filter.Limit + filter.Skip
		filter.Limit = min(100, oriLimit)
		for filter.Limit+filter.Skip <= oriLimit {
			impls, err := s.Find(ctx, filter)
			if err != nil {
				clog.Errorf(ctx, "failed to find comics: %s", err)
				return
			}
			if len(impls) == 0 {
				break
			}
			for _, c := range impls {
				comics <- c
			}
			filter.Skip += int64(len(impls))
		}
	}()
	return comics, nil
}

func (s *Storage) toMongoFilter(filter *comic.ComicFilter) bson.M {
	mongoFilter := bson.M{}
	if filter == nil {
		return mongoFilter
	}

	if filter.ID != nil {
		mongoFilter["comicid"] = *filter.ID
	} else {
		idFilter := bson.M{}
		if filter.IDRangeLeft != nil {
			idFilter["$gte"] = *filter.IDRangeLeft
		}
		if filter.IDRangeRight != nil {
			idFilter["$lte"] = *filter.IDRangeRight
		}
		if len(idFilter) != 0 {
			mongoFilter["comicid"] = idFilter
		}
	}
	if filter.TitlePattern != nil {
		mongoFilter["name"] = bson.M{"$regex": primitive.Regex{Pattern: *filter.TitlePattern, Options: "i"}}
	}
	if filter.Valid != nil {
		mongoFilter["verify.valid"] = *filter.Valid
	}
	if filter.HasValid != nil {
		if *filter.HasValid {
			mongoFilter["verify.valid"] = bson.M{"$exists": 1}
		} else {
			mongoFilter["verify.valid"] = bson.M{"$exists": 0}
		}
	}
	if filter.NotArchived != nil {
		if *filter.NotArchived {
			mongoFilter["archive.path"] = bson.M{"$exists": 0}
		} else {
			mongoFilter["archive.path"] = bson.M{"$exists": 1}
		}
	}

	return mongoFilter
}

// SaveVerifyResult 保存验证结果
func (s *Storage) SaveVerifyResult(ctx context.Context, result *comic.VerifyResult) error {
	verifyInfo := comic.VerifyInfo{}
	verifyInfo.SetVerifyResult(result)
	err := UpdateOneComicInfo(ctx, result.ComicID, map[string]interface{}{
		"verify": verifyInfo,
	})
	if err != nil {
		return fmt.Errorf("failed to save verify result: %w", err)
	}
	return nil
}
