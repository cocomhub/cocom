package comic

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

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
	cid, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("invalid comic id: %w", err)
	}

	info := &api.ComicInfo{}
	err = GetComicInfo(ctx, cid, info)
	if err != nil {
		return nil, fmt.Errorf("failed to get comic: %w", err)
	}

	return NewComic(info), nil
}

// Update 更新漫画数据
func (s *Storage) Update(ctx context.Context, obj interface{}) error {
	c, err := NewComicByObject(obj)
	if err != nil {
		return err
	}

	if c == nil || c.CID == 0 {
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

	err = UpdateComicInfo(ctx, c.CID, v)
	if err != nil {
		return fmt.Errorf("failed to save comic: %w", err)
	}
	return nil
}

// Find 列出符合条件的漫画
func (s *Storage) Find(ctx context.Context, filter *comic.ComicFilter) ([]comic.Comic, error) {
	cursor, err := mongo.ComicInfo().Find(ctx, s.toMongoFilter(ctx, filter), &options.FindOptions{
		Sort:  bson.M{"cid": 1},
		Limit: &filter.Limit,
		Skip:  &filter.Skip,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var infos []api.ComicInfo
	if err := cursor.All(ctx, &infos); err != nil {
		return nil, err
	}

	// 转换为接口类型
	comics := make([]comic.Comic, len(infos))
	for i := range infos {
		comics[i] = NewComic(&infos[i])
	}
	return comics, nil
}

func (s *Storage) toMongoFilter(ctx context.Context, filter *comic.ComicFilter) bson.M {
	mongoFilter := bson.M{}
	if filter == nil {
		return mongoFilter
	}

	if filter.ID != nil {
		cid, err := strconv.Atoi(*filter.ID)
		if err != nil {
			clog.Errorf(ctx, "invalid comic id: %s", *filter.ID)
		} else {
			mongoFilter["cid"] = cid
		}
	}
	if filter.TitlePattern != nil {
		mongoFilter["$or"] = []bson.M{
			{"title.english": bson.M{"$regex": primitive.Regex{Pattern: *filter.TitlePattern, Options: "i"}}},
			{"title.japanese": bson.M{"$regex": primitive.Regex{Pattern: *filter.TitlePattern, Options: "i"}}},
			{"title.pretty": bson.M{"$regex": primitive.Regex{Pattern: *filter.TitlePattern, Options: "i"}}},
		}
	}
	if filter.PageMin != nil && filter.PageMax != nil {
		mongoFilter["num_pages"] = bson.M{"$gte": *filter.PageMin, "$lte": *filter.PageMax}
	} else if filter.PageMin != nil {
		mongoFilter["num_pages"] = bson.M{"$gte": *filter.PageMin}
	} else if filter.PageMax != nil {
		mongoFilter["num_pages"] = bson.M{"$lte": *filter.PageMax}
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

	return mongoFilter
}

// SaveVerifyResult 保存验证结果
func (s *Storage) SaveVerifyResult(ctx context.Context, result *comic.VerifyResult) error {
	cid, err := strconv.Atoi(result.ComicID)
	if err != nil {
		return fmt.Errorf("invalid comic id: %w", err)
	}

	verifyInfo := comic.VerifyInfo{}
	verifyInfo.SetVerifyResult(result)
	err = UpdateComicInfo(ctx, cid, map[string]interface{}{
		"verify": verifyInfo,
	})
	if err != nil {
		return fmt.Errorf("failed to save verify result: %w", err)
	}
	return nil
}
